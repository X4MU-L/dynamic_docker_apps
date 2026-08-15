use crate::api::handlers::{
    deregister_upstream_handler, health_check_handler, list_upstreams_handler,
    register_upstream_handler,
};
use crate::domain::models::DynamicLBState;
use axum::{
    routing::{delete, get, post},
    Router,
};
use tokio::net::TcpListener;

pub fn create_router(state: DynamicLBState) -> Router {
    Router::new()
        .route("/upstreams", post(register_upstream_handler))
        .route("/upstreams", delete(deregister_upstream_handler))
        .route("/upstreams", get(list_upstreams_handler))
        .route("/health", get(health_check_handler))
        .with_state(state)
}

pub async fn start_api_server(addr: &str, state: DynamicLBState) {
    let app = create_router(state);
    let listener = match TcpListener::bind(addr).await {
        Ok(l) => l,
        Err(e) => {
            tracing::error!("Failed to bind API server to {}: {}", addr, e);
            return;
        }
    };
    tracing::info!("Internal Control API listening on http://{}", addr);
    if let Err(e) = axum::serve(listener, app).await {
        tracing::error!("API server error: {}", e);
    }
}

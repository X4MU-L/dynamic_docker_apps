use crate::api::handlers::{
    deregister_upstream_handler, health_check_handler, list_upstreams_handler,
    register_upstream_handler,
};
use crate::domain::models::DynamicLBState;
use async_trait::async_trait;
use axum::{
    routing::{delete, get, post},
    Router,
};
use pingora::server::ShutdownWatch;
use pingora::services::background::BackgroundService;
use tokio::net::TcpListener;

pub struct ApiServerService {
    pub api_addr: String,
    pub state: DynamicLBState,
}

impl ApiServerService {
    pub fn new(api_addr: String, state: DynamicLBState) -> Self {
        Self { api_addr, state }
    }
}

#[async_trait]
impl BackgroundService for ApiServerService {
    async fn start(&self, shutdown: ShutdownWatch) {
        start_api_server(&self.api_addr, self.state.clone(), shutdown).await;
    }
}

pub fn create_router(state: DynamicLBState) -> Router {
    Router::new()
        .route("/upstreams", post(register_upstream_handler))
        .route("/upstreams", delete(deregister_upstream_handler))
        .route("/upstreams", get(list_upstreams_handler))
        .route("/health", get(health_check_handler))
        .with_state(state)
}

pub async fn start_api_server(addr: &str, state: DynamicLBState, mut shutdown: ShutdownWatch) {
    let app = create_router(state);
    let listener = match TcpListener::bind(addr).await {
        Ok(l) => l,
        Err(e) => {
            tracing::error!("Failed to bind API server to {}: {}", addr, e);
            return;
        }
    };
    tracing::info!("Internal Control API listening on http://{}", addr);

    let shutdown_signal = async move {
        loop {
            let res = shutdown.changed().await;
            if res.is_err() || *shutdown.borrow() {
                tracing::info!("API server received shutdown signal. Exiting gracefully...");
                break;
            }
        }
    };

    if let Err(e) = axum::serve(listener, app).with_graceful_shutdown(shutdown_signal).await {
        tracing::error!("API server error during shutdown: {}", e);
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::domain::models::Algorithm;

    #[test]
    fn test_api_server_service_constructor() {
        let state = DynamicLBState::new(Algorithm::RoundRobin);
        let svc = ApiServerService::new("127.0.0.1:8081".to_string(), state);
        assert_eq!(svc.api_addr, "127.0.0.1:8081");
    }

    #[test]
    fn test_create_router_binding() {
        let state = DynamicLBState::new(Algorithm::RoundRobin);
        let _router = create_router(state);
    }
}

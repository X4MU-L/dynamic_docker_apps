use crate::domain::models::{
    BackendItem, DynamicLBState, UpstreamDeregistrationPayload, UpstreamRegistrationPayload,
};
use crate::domain::routing::{deregister_upstream, register_upstream};
use axum::{extract::State, http::StatusCode, Json};
use serde_json::{json, Value};

pub async fn register_upstream_handler(
    State(state): State<DynamicLBState>,
    Json(payload): Json<UpstreamRegistrationPayload>,
) -> (StatusCode, Json<Value>) {
    let item = BackendItem::new(
        payload.ip.clone(),
        payload.port,
        payload.sni_name,
        payload.health_endpoint,
    );
    register_upstream(&state, item);
    (
        StatusCode::OK,
        Json(json!({"status": "registered", "ip": payload.ip, "port": payload.port})),
    )
}

pub async fn deregister_upstream_handler(
    State(state): State<DynamicLBState>,
    Json(payload): Json<UpstreamDeregistrationPayload>,
) -> (StatusCode, Json<Value>) {
    deregister_upstream(&state, &payload.ip, payload.port);
    (
        StatusCode::OK,
        Json(json!({"status": "deregistered", "ip": payload.ip})),
    )
}

pub async fn list_upstreams_handler(
    State(state): State<DynamicLBState>,
) -> (StatusCode, Json<Value>) {
    let items = state.items.load();
    (StatusCode::OK, Json(json!(**items)))
}

pub async fn health_check_handler() -> (StatusCode, Json<Value>) {
    (StatusCode::OK, Json(json!({"status": "healthy"})))
}

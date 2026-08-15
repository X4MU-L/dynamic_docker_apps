use axum::body::Body;
use axum::http::{Request, StatusCode};
use dynamic_docker_apps::api::server::create_router;
use dynamic_docker_apps::domain::models::{Algorithm, DynamicLBState};
use dynamic_docker_apps::domain::routing::select_backend;
use serde_json::json;
use tower::util::ServiceExt;

#[tokio::test]
async fn test_full_e2e_flow_registration_routing_deregistration() {
    let state = DynamicLBState::new(Algorithm::RoundRobin);
    let app = create_router(state.clone());

    let payload = json!({
        "ip": "10.28.0.5",
        "port": 8080,
        "sni_name": "container-instance-alpha",
        "health_endpoint": "/health"
    });

    let req = Request::builder()
        .method("POST")
        .uri("/upstreams")
        .header("Content-Type", "application/json")
        .body(Body::from(payload.to_string()))
        .unwrap();

    let resp = app.clone().oneshot(req).await.unwrap();
    assert_eq!(resp.status(), StatusCode::CREATED);

    let (backend, sni) = select_backend(&state, b"").expect("Backend should be registered");
    assert_eq!(backend.addr.to_string(), "10.28.0.5:8080");
    assert_eq!(sni, "container-instance-alpha");

    let dereg_payload = json!({ "ip": "10.28.0.5", "port": 8080 });
    let dereg_req = Request::builder()
        .method("DELETE")
        .uri("/upstreams")
        .header("Content-Type", "application/json")
        .body(Body::from(dereg_payload.to_string()))
        .unwrap();

    let dereg_resp = app.oneshot(dereg_req).await.unwrap();
    assert_eq!(dereg_resp.status(), StatusCode::OK);
}

#[tokio::test]
async fn test_e2e_error_responses_for_invalid_requests() {
    let state = DynamicLBState::new(Algorithm::RoundRobin);
    let app = create_router(state);

    let invalid_payload = json!({ "ip": "not-an-ip", "port": 8080 });
    let req = Request::builder()
        .method("POST")
        .uri("/upstreams")
        .header("Content-Type", "application/json")
        .body(Body::from(invalid_payload.to_string()))
        .unwrap();

    let resp = app.oneshot(req).await.unwrap();
    assert_eq!(resp.status(), StatusCode::BAD_REQUEST);
}

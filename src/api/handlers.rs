use crate::domain::errors::DomainError;
use crate::domain::models::{
    BackendItem, DynamicLBState, UpstreamDeregistrationPayload, UpstreamRegistrationPayload,
};
use crate::domain::routing::{deregister_upstream, register_upstream};
use axum::{extract::State, http::StatusCode, Json};
use serde_json::{json, Value};

pub async fn register_upstream_handler(
    State(state): State<DynamicLBState>,
    Json(payload): Json<UpstreamRegistrationPayload>,
) -> Result<(StatusCode, Json<Value>), DomainError> {
    payload.validate()?;
    let item = BackendItem::new(
        payload.ip.clone(),
        payload.port,
        payload.sni_name,
        payload.health_endpoint,
    );
    register_upstream(&state, item)?;
    Ok((
        StatusCode::CREATED,
        Json(json!({"status": "registered", "ip": payload.ip, "port": payload.port})),
    ))
}

pub async fn deregister_upstream_handler(
    State(state): State<DynamicLBState>,
    Json(payload): Json<UpstreamDeregistrationPayload>,
) -> Result<(StatusCode, Json<Value>), DomainError> {
    payload.validate()?;
    deregister_upstream(&state, &payload.ip, payload.port)?;
    Ok((
        StatusCode::OK,
        Json(json!({"status": "deregistered", "ip": payload.ip})),
    ))
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

#[cfg(test)]
mod tests {
    use super::*;
    use crate::domain::models::Algorithm;
    use crate::domain::routing::select_backend;

    #[tokio::test]
    async fn test_register_handler_success() {
        let state = DynamicLBState::new(Algorithm::RoundRobin);
        let payload = UpstreamRegistrationPayload {
            ip: "10.0.0.1".to_string(),
            port: 8080,
            sni_name: Some("test-sni".to_string()),
            health_endpoint: Some("/health".to_string()),
        };

        let (status, body) = register_upstream_handler(State(state.clone()), Json(payload))
            .await
            .unwrap();
        assert_eq!(status, StatusCode::CREATED);
        assert_eq!(body.0["status"], "registered");

        let selected = select_backend(&state, b"");
        assert!(selected.is_some());
        let (backend, sni) = selected.unwrap();
        assert_eq!(backend.addr.to_string(), "10.0.0.1:8080");
        assert_eq!(sni, "test-sni");
    }

    #[tokio::test]
    async fn test_register_handler_invalid_ip() {
        let state = DynamicLBState::new(Algorithm::RoundRobin);
        let payload = UpstreamRegistrationPayload {
            ip: "not-an-ip".to_string(),
            port: 8080,
            sni_name: None,
            health_endpoint: None,
        };

        let err = register_upstream_handler(State(state), Json(payload))
            .await
            .unwrap_err();
        assert_eq!(err, DomainError::InvalidIpAddress("not-an-ip".to_string()));
    }

    #[tokio::test]
    async fn test_register_handler_duplicate_conflict() {
        let state = DynamicLBState::new(Algorithm::RoundRobin);
        let payload = UpstreamRegistrationPayload {
            ip: "10.0.0.2".to_string(),
            port: 8080,
            sni_name: None,
            health_endpoint: None,
        };

        assert!(register_upstream_handler(State(state.clone()), Json(payload.clone())).await.is_ok());
        let err = register_upstream_handler(State(state), Json(payload)).await.unwrap_err();
        assert_eq!(err, DomainError::BackendAlreadyExists("10.0.0.2:8080".to_string()));
    }

    #[tokio::test]
    async fn test_deregister_handler_success() {
        let state = DynamicLBState::new(Algorithm::RoundRobin);
        let reg_payload = UpstreamRegistrationPayload {
            ip: "10.0.0.3".to_string(),
            port: 8080,
            sni_name: None,
            health_endpoint: None,
        };
        let _ = register_upstream_handler(State(state.clone()), Json(reg_payload)).await;

        let dereg_payload = UpstreamDeregistrationPayload {
            ip: "10.0.0.3".to_string(),
            port: Some(8080),
        };
        let (status, body) = deregister_upstream_handler(State(state.clone()), Json(dereg_payload))
            .await
            .unwrap();
        assert_eq!(status, StatusCode::OK);
        assert_eq!(body.0["status"], "deregistered");
        assert!(select_backend(&state, b"").is_none());
    }

    #[tokio::test]
    async fn test_deregister_handler_not_found() {
        let state = DynamicLBState::new(Algorithm::RoundRobin);
        let dereg_payload = UpstreamDeregistrationPayload {
            ip: "10.0.0.99".to_string(),
            port: Some(8080),
        };
        let err = deregister_upstream_handler(State(state), Json(dereg_payload))
            .await
            .unwrap_err();
        assert_eq!(err, DomainError::BackendNotFound("10.0.0.99:8080".to_string()));
    }

    #[tokio::test]
    async fn test_list_upstreams_handler() {
        let state = DynamicLBState::new(Algorithm::RoundRobin);
        let (status, body) = list_upstreams_handler(State(state.clone())).await;
        assert_eq!(status, StatusCode::OK);
        assert_eq!(body.0, json!([]));

        let reg_payload = UpstreamRegistrationPayload {
            ip: "10.0.0.4".to_string(),
            port: 8080,
            sni_name: Some("sni-4".to_string()),
            health_endpoint: Some("/health".to_string()),
        };
        let _ = register_upstream_handler(State(state.clone()), Json(reg_payload)).await;

        let (status2, body2) = list_upstreams_handler(State(state)).await;
        assert_eq!(status2, StatusCode::OK);
        assert_eq!(body2.0[0]["ip"], "10.0.0.4");
        assert_eq!(body2.0[0]["sni_name"], "sni-4");
    }

    #[tokio::test]
    async fn test_health_check_handler() {
        let (status, body) = health_check_handler().await;
        assert_eq!(status, StatusCode::OK);
        assert_eq!(body.0["status"], "healthy");
    }
}

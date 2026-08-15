use axum::{http::StatusCode, response::IntoResponse, Json};
use serde_json::json;
use thiserror::Error;

#[derive(Error, Debug, PartialEq, Clone)]
pub enum DomainError {
    #[error("Invalid IP address format: '{0}'")]
    InvalidIpAddress(String),

    #[error("Invalid port number: {0}")]
    InvalidPort(u16),

    #[error("Health endpoint must start with '/': '{0}'")]
    InvalidHealthEndpoint(String),

    #[error("Backend not found: '{0}'")]
    BackendNotFound(String),

    #[error("Backend already registered: '{0}'")]
    BackendAlreadyExists(String),

    #[error("Failed to rebuild load balancer: {0}")]
    LoadBalancerRebuildFailed(String),
}

impl IntoResponse for DomainError {
    fn into_response(self) -> axum::response::Response {
        let (status, msg) = match &self {
            Self::InvalidIpAddress(_) | Self::InvalidPort(_) | Self::InvalidHealthEndpoint(_) => {
                (StatusCode::BAD_REQUEST, self.to_string())
            }
            Self::BackendNotFound(_) => (StatusCode::NOT_FOUND, self.to_string()),
            Self::BackendAlreadyExists(_) => (StatusCode::CONFLICT, self.to_string()),
            Self::LoadBalancerRebuildFailed(_) => {
                (StatusCode::INTERNAL_SERVER_ERROR, self.to_string())
            }
        };

        let body = Json(json!({
            "error": msg,
            "code": status.as_u16(),
        }));
        (status, body).into_response()
    }
}

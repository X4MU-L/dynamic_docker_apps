use axum::{http::StatusCode, response::IntoResponse, Json};
use serde_json::json;
use thiserror::Error;

#[derive(Error, Debug, PartialEq, Clone)]
pub enum DomainError {
    #[error("Invalid IP address format: '{0}'")]
    InvalidIpAddress(String),

    #[error("Invalid port number: {0}")]
    InvalidPort(u16),

    #[error("SNI name / Hostname is required and cannot be empty: '{0}'")]
    InvalidSniName(String),

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
            Self::InvalidIpAddress(_)
            | Self::InvalidPort(_)
            | Self::InvalidSniName(_)
            | Self::InvalidHealthEndpoint(_) => (StatusCode::BAD_REQUEST, self.to_string()),
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

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_domain_error_display_formatting() {
        let err = DomainError::InvalidIpAddress("bad-ip".to_string());
        assert_eq!(err.to_string(), "Invalid IP address format: 'bad-ip'");

        let sni_err = DomainError::InvalidSniName("".to_string());
        assert_eq!(
            sni_err.to_string(),
            "SNI name / Hostname is required and cannot be empty: ''"
        );
    }

    #[test]
    fn test_domain_error_into_response_status() {
        let err = DomainError::InvalidSniName("".to_string());
        let resp = err.into_response();
        assert_eq!(resp.status(), StatusCode::BAD_REQUEST);
    }
}

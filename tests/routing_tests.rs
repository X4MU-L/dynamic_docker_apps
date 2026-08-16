use dynamic_docker_apps::domain::errors::DomainError;
use dynamic_docker_apps::domain::models::{
    Algorithm, BackendItem, DynamicLBState, UpstreamRegistrationPayload,
};
use dynamic_docker_apps::domain::routing::{
    deregister_upstream, register_upstream, select_backend,
};

#[test]
fn test_register_and_select_backend_round_robin() {
    let state = DynamicLBState::new(Algorithm::RoundRobin);
    let item1 = BackendItem::new(
        "172.28.0.2".to_string(),
        8080,
        "app-1.edge.local".to_string(),
        None,
    );
    assert!(register_upstream(&state, item1).is_ok());

    let (backend, sni) = select_backend(&state, b"").expect("Backend should be selected");
    assert_eq!(backend.addr.to_string(), "172.28.0.2:8080");
    assert_eq!(sni, "app-1.edge.local");
}

#[test]
fn test_register_duplicate_returns_conflict_error() {
    let state = DynamicLBState::new(Algorithm::RoundRobin);
    let item1 = BackendItem::new(
        "172.28.0.2".to_string(),
        8080,
        "app-1.edge.local".to_string(),
        None,
    );
    assert!(register_upstream(&state, item1.clone()).is_ok());

    let err = register_upstream(&state, item1).unwrap_err();
    assert_eq!(
        err,
        DomainError::BackendAlreadyExists("172.28.0.2:8080".to_string())
    );
}

#[test]
fn test_deregister_non_existent_returns_not_found_error() {
    let state = DynamicLBState::new(Algorithm::RoundRobin);
    let err = deregister_upstream(&state, "172.28.0.99", Some(8080)).unwrap_err();
    assert_eq!(
        err,
        DomainError::BackendNotFound("172.28.0.99:8080".to_string())
    );
}

#[test]
fn test_payload_validation_errors() {
    let invalid_ip = UpstreamRegistrationPayload {
        ip: "invalid-ip-addr".to_string(),
        port: 8080,
        sni_name: "app.edge.local".to_string(),
        health_endpoint: None,
    };
    assert_eq!(
        invalid_ip.validate().unwrap_err(),
        DomainError::InvalidIpAddress("invalid-ip-addr".to_string())
    );

    let invalid_port = UpstreamRegistrationPayload {
        ip: "127.0.0.1".to_string(),
        port: 0,
        sni_name: "app.edge.local".to_string(),
        health_endpoint: None,
    };
    assert_eq!(
        invalid_port.validate().unwrap_err(),
        DomainError::InvalidPort(0)
    );
}

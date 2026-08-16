use super::errors::DomainError;
use super::models::{BackendItem, DynamicLBState, DynamicLb};
use pingora::lb::Backend;
use std::sync::Arc;

pub fn register_upstream(state: &DynamicLBState, item: BackendItem) -> Result<(), DomainError> {
    let current_items = state.items.load();
    let mut updated_items = (**current_items).clone();

    if updated_items
        .iter()
        .any(|b| b.ip == item.ip && b.port == item.port)
    {
        return Err(DomainError::BackendAlreadyExists(item.address()));
    }

    updated_items.push(item);
    rebuild_load_balancer(state, &updated_items);
    state.items.store(Arc::new(updated_items));
    Ok(())
}

pub fn deregister_upstream(
    state: &DynamicLBState,
    ip: &str,
    port: Option<u16>,
) -> Result<(), DomainError> {
    let current_items = state.items.load();
    let mut updated_items = (**current_items).clone();

    let initial_len = updated_items.len();
    updated_items.retain(|b| {
        if b.ip != ip {
            return true;
        }
        if let Some(p) = port {
            b.port != p
        } else {
            false
        }
    });

    if updated_items.len() == initial_len {
        let target = port
            .map(|p| format!("{}:{}", ip, p))
            .unwrap_or_else(|| ip.to_string());
        return Err(DomainError::BackendNotFound(target));
    }

    rebuild_load_balancer(state, &updated_items);
    state.items.store(Arc::new(updated_items));
    Ok(())
}

pub fn rebuild_load_balancer(state: &DynamicLBState, items: &[BackendItem]) {
    let current_items = state.items.load();
    if current_items.as_slice() == items {
        return;
    }

    let backends: Vec<Backend> = items
        .iter()
        .filter_map(|item| item.to_pingora_backend())
        .collect();

    let new_lb = DynamicLb::new(state.algorithm, backends);
    state.lb.store(Arc::new(new_lb));
}

pub fn select_backend(state: &DynamicLBState, key: &[u8]) -> Option<(Backend, String)> {
    let lb = state.lb.load();
    let backend = lb.select(key, 256)?;
    let items = state.items.load();
    let sni = find_sni_name(&items, &backend.addr.to_string());
    Some((backend, sni))
}

pub fn find_sni_name(items: &[BackendItem], addr: &str) -> String {
    items
        .iter()
        .find(|item| item.address() == addr)
        .map(|item| item.sni_name.clone())
        .unwrap_or_else(|| addr.to_string())
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::domain::models::Algorithm;

    #[test]
    fn test_register_and_select_round_robin() {
        let state = DynamicLBState::new(Algorithm::RoundRobin);
        let item = BackendItem::new(
            "10.0.0.1".to_string(),
            8080,
            Some("sni-1".to_string()),
            None,
        );
        assert!(register_upstream(&state, item).is_ok());

        let (backend, sni) = select_backend(&state, b"").expect("Backend selected");
        assert_eq!(backend.addr.to_string(), "10.0.0.1:8080");
        assert_eq!(sni, "sni-1");
    }

    #[test]
    fn test_register_duplicate_returns_error() {
        let state = DynamicLBState::new(Algorithm::RoundRobin);
        let item = BackendItem::new("10.0.0.1".to_string(), 8080, None, None);
        assert!(register_upstream(&state, item.clone()).is_ok());
        let err = register_upstream(&state, item).unwrap_err();
        assert_eq!(
            err,
            DomainError::BackendAlreadyExists("10.0.0.1:8080".to_string())
        );
    }

    #[test]
    fn test_deregister_success_by_ip_and_port() {
        let state = DynamicLBState::new(Algorithm::RoundRobin);
        let item1 = BackendItem::new("10.0.0.1".to_string(), 8080, None, None);
        let item2 = BackendItem::new("10.0.0.1".to_string(), 9090, None, None);
        let _ = register_upstream(&state, item1);
        let _ = register_upstream(&state, item2);

        assert!(deregister_upstream(&state, "10.0.0.1", Some(8080)).is_ok());
        let remaining = state.items.load();
        assert_eq!(remaining.len(), 1);
        assert_eq!(remaining[0].port, 9090);
    }

    #[test]
    fn test_deregister_success_by_ip_only() {
        let state = DynamicLBState::new(Algorithm::RoundRobin);
        let item = BackendItem::new("10.0.0.2".to_string(), 8080, None, None);
        let _ = register_upstream(&state, item);
        assert!(deregister_upstream(&state, "10.0.0.2", None).is_ok());
        assert!(state.items.load().is_empty());
    }

    #[test]
    fn test_deregister_non_existent_returns_error() {
        let state = DynamicLBState::new(Algorithm::RoundRobin);
        let err = deregister_upstream(&state, "10.0.0.99", Some(8080)).unwrap_err();
        assert_eq!(
            err,
            DomainError::BackendNotFound("10.0.0.99:8080".to_string())
        );
    }

    #[test]
    fn test_find_sni_name_matching_and_fallback() {
        let items = vec![BackendItem::new(
            "10.0.0.1".to_string(),
            8080,
            Some("custom-sni".to_string()),
            None,
        )];
        assert_eq!(find_sni_name(&items, "10.0.0.1:8080"), "custom-sni");
        assert_eq!(find_sni_name(&items, "10.0.0.99:8080"), "10.0.0.99:8080");
    }

    #[test]
    fn test_differential_check_peeping_no_rebuild() {
        let state = DynamicLBState::new(Algorithm::RoundRobin);
        let items = vec![BackendItem::new("10.0.0.1".to_string(), 8080, None, None)];
        let _ = register_upstream(&state, items[0].clone());

        let lb_before = Arc::clone(&*state.lb.load());
        rebuild_load_balancer(&state, &items);
        let lb_after = Arc::clone(&*state.lb.load());

        assert!(Arc::ptr_eq(&lb_before, &lb_after));
    }
}

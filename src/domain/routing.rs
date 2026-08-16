use super::errors::DomainError;
use super::models::{BackendItem, BackendStatus, BackendStatusResponse, DynamicLBState, DynamicLb};
use pingora::lb::Backend;
use std::collections::HashMap;
use std::net::SocketAddr;
use std::sync::atomic::Ordering;
use std::sync::Arc;
use std::time::{Duration, Instant};

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

pub fn mark_draining_upstream(
    state: &DynamicLBState,
    ip: Option<&str>,
    port: Option<u16>,
    sni_name: Option<&str>,
    timeout_secs: u64,
) -> Result<BackendItem, DomainError> {
    let current_items = state.items.load();
    let mut updated_items = (**current_items).clone();
    let mut target_item = None;

    for item in updated_items.iter_mut() {
        let matches_ip = ip.map_or(true, |target_ip| item.ip == target_ip);
        let matches_port = port.map_or(true, |target_port| item.port == target_port);
        let matches_sni = sni_name.map_or(true, |target_sni| &*item.sni_name == target_sni);

        if matches_ip && matches_port && matches_sni {
            item.status = BackendStatus::Draining;
            item.drain_deadline = Some(Instant::now() + Duration::from_secs(timeout_secs));
            target_item = Some(item.clone());
            break;
        }
    }

    let item = target_item.ok_or_else(|| {
        let target = sni_name
            .map(|s| s.to_string())
            .or_else(|| ip.map(|i| format!("{}:{}", i, port.unwrap_or(0))))
            .unwrap_or_default();
        DomainError::BackendNotFound(target)
    })?;

    rebuild_load_balancer(state, &updated_items);
    state.items.store(Arc::new(updated_items));
    Ok(item)
}

pub fn get_upstream_status(
    state: &DynamicLBState,
    ip: Option<&str>,
    port: Option<u16>,
    sni: Option<&str>,
) -> Option<BackendStatusResponse> {
    let items = state.items.load();
    items.iter().find_map(|item| {
        let matches_ip = ip.map_or(true, |target_ip| item.ip == target_ip);
        let matches_port = port.map_or(true, |target_port| item.port == target_port);
        let matches_sni = sni.map_or(true, |target_sni| &*item.sni_name == target_sni);

        if matches_ip && matches_port && matches_sni {
            Some(item.to_status_response())
        } else {
            None
        }
    })
}

pub fn purge_drained_upstreams(state: &DynamicLBState) -> usize {
    let current_items = state.items.load();
    let mut updated_items = (**current_items).clone();
    let initial_count = updated_items.len();
    let now = Instant::now();

    // get only the active backends and those that are draining but still have active requests
    // or haven't reached their drain deadline
    updated_items.retain(|item| {
        if item.status != BackendStatus::Draining {
            return true;
        }
        let requests = item.active_requests.load(Ordering::Relaxed);
        let deadline_expired = item.drain_deadline.map_or(false, |d| now >= d);
        requests > 0 && !deadline_expired
    });

    let purged_count = initial_count - updated_items.len();
    if purged_count > 0 {
        // update the load balancer with the remaining  ACTIVE items after purging
        // this will not add items that are still draining or its dealine is yet to expire
        rebuild_load_balancer(state, &updated_items);
        // store the updated items back into the state
        // this may include items that are still draining but have active requests or haven't reached their drain deadline
        state.items.store(Arc::new(updated_items));
    }
    purged_count
}

pub fn rebuild_load_balancer(state: &DynamicLBState, items: &[BackendItem]) {
    let mut by_addr_map: HashMap<SocketAddr, BackendItem> = HashMap::new();
    let mut active_backends = Vec::new();

    for item in items {
        if let Some(addr) = item.socket_addr() {
            // Store the item in the by_addr_map for quick lookup
            // This allows us to retrieve the BackendItem based on its SocketAddr when selecting a backend
            // this may include items that are still draining but have active requests or haven't reached their drain deadline
            by_addr_map.insert(addr, item.clone());
            if item.status == BackendStatus::Active {
                if let Some(backend) = item.to_pingora_backend() {
                    // only add active backends to the load balancer, draining backends will be excluded from selection
                    active_backends.push(backend);
                }
            }
        }
    }

    state.by_addr.store(Arc::new(by_addr_map));
    let new_lb = DynamicLb::new(state.algorithm, active_backends);
    state.lb.store(Arc::new(new_lb));
}

pub fn select_backend(state: &DynamicLBState, key: &[u8]) -> Option<(Backend, Arc<str>)> {
    let lb = state.lb.load();
    let backend = lb.select(key, 256)?;

    if let Some(sock_addr) = backend.addr.as_inet() {
        let map = state.by_addr.load();
        if let Some(item) = map.get(sock_addr) {
            return Some((backend, item.sni_name.clone()));
        }
    }

    let fallback: Arc<str> = backend.addr.to_string().into();
    Some((backend, fallback))
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
            "sni-1.edge.local".to_string(),
            None,
        );
        assert!(register_upstream(&state, item).is_ok());

        let (backend, sni) = select_backend(&state, b"").expect("Backend selected");
        assert_eq!(backend.addr.to_string(), "10.0.0.1:8080");
        assert_eq!(&*sni, "sni-1.edge.local");

        let map = state.by_addr.load();
        let addr: SocketAddr = "10.0.0.1:8080".parse().unwrap();
        assert!(map.contains_key(&addr));
    }

    #[test]
    fn test_mark_draining_excludes_from_lb() {
        let state = DynamicLBState::new(Algorithm::RoundRobin);
        let item = BackendItem::new(
            "10.0.0.1".to_string(),
            8080,
            "sni-1.edge.local".to_string(),
            None,
        );
        let _ = register_upstream(&state, item);
        assert!(select_backend(&state, b"").is_some());

        assert!(mark_draining_upstream(&state, Some("10.0.0.1"), Some(8080), None, 10).is_ok());
        assert!(select_backend(&state, b"").is_none());

        let status = get_upstream_status(&state, Some("10.0.0.1"), Some(8080), None).unwrap();
        assert_eq!(status.status, BackendStatus::Draining);
    }

    #[test]
    fn test_purge_drained_upstreams() {
        let state = DynamicLBState::new(Algorithm::RoundRobin);
        let item = BackendItem::new(
            "10.0.0.1".to_string(),
            8080,
            "sni-1.edge.local".to_string(),
            None,
        );
        let _ = register_upstream(&state, item);
        let _ = mark_draining_upstream(&state, Some("10.0.0.1"), Some(8080), None, 0);

        let purged = purge_drained_upstreams(&state);
        assert_eq!(purged, 1);
        assert!(get_upstream_status(&state, Some("10.0.0.1"), Some(8080), None).is_none());
    }
}

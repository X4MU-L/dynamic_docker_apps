use super::models::{BackendItem, DynamicLBState, DynamicLb};
use pingora::lb::Backend;
use std::sync::Arc;

pub fn register_upstream(state: &DynamicLBState, item: BackendItem) {
    let current_items = state.items.load();
    let mut updated_items = (**current_items).clone();

    if !updated_items.iter().any(|b| b.ip == item.ip && b.port == item.port) {
        updated_items.push(item);
    }

    rebuild_load_balancer(state, &updated_items);
    state.items.store(Arc::new(updated_items));
}

pub fn deregister_upstream(state: &DynamicLBState, ip: &str, port: Option<u16>) {
    let current_items = state.items.load();
    let mut updated_items = (**current_items).clone();

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

    rebuild_load_balancer(state, &updated_items);
    state.items.store(Arc::new(updated_items));
}

pub fn rebuild_load_balancer(state: &DynamicLBState, items: &[BackendItem]) {
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

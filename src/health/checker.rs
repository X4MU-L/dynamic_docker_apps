use crate::domain::models::{BackendItem, DynamicLBState};
use crate::domain::routing::rebuild_load_balancer;
use reqwest::Client;
use std::time::Duration;
use tokio::time::sleep;

pub fn start_health_checker(state: DynamicLBState, interval_secs: u64) {
    tokio::spawn(async move {
        let client = Client::builder()
            .timeout(Duration::from_secs(2))
            .build()
            .unwrap_or_default();
        loop {
            sleep(Duration::from_secs(interval_secs)).await;
            run_health_check_cycle(&state, &client).await;
        }
    });
}

pub async fn run_health_check_cycle(state: &DynamicLBState, client: &Client) {
    let current_items = state.items.load();
    let items = (**current_items).clone();
    let mut healthy_items = Vec::new();

    for item in items {
        if probe_backend_health(client, &item).await {
            healthy_items.push(item);
        } else {
            tracing::warn!("Backend {} failed health probe", item.address());
        }
    }

    rebuild_load_balancer(state, &healthy_items);
}

pub async fn probe_backend_health(client: &Client, item: &BackendItem) -> bool {
    let url = format!("http://{}{}", item.address(), item.health_endpoint);
    match client.get(&url).send().await {
        Ok(resp) => resp.status().is_success(),
        Err(_) => false,
    }
}

use crate::domain::models::{BackendItem, DynamicLBState};
use crate::domain::routing::rebuild_load_balancer;
use async_trait::async_trait;
use pingora::server::ShutdownWatch;
use pingora::services::background::BackgroundService;
use reqwest::Client;
use std::time::Duration;

pub struct HealthCheckService {
    pub state: DynamicLBState,
    pub interval_secs: u64,
}

impl HealthCheckService {
    pub fn new(state: DynamicLBState, interval_secs: u64) -> Self {
        Self {
            state,
            interval_secs,
        }
    }
}

#[async_trait]
impl BackgroundService for HealthCheckService {
    async fn start(&self, mut shutdown: ShutdownWatch) {
        if *shutdown.borrow() {
            return;
        }

        let client = Client::builder()
            .timeout(Duration::from_secs(2))
            .build()
            .unwrap_or_default();

        loop {
            tokio::select! {
                res = shutdown.changed() => {
                    if res.is_err() || *shutdown.borrow() {
                        tracing::info!("Health check service shutdown signal received. Exiting...");
                        break;
                    }
                }
                _ = tokio::time::sleep(Duration::from_secs(self.interval_secs)) => {
                    run_health_check_cycle(&self.state, &client).await;
                }
            }
        }
    }
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

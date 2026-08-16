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
            .pool_max_idle_per_host(0)
            .timeout(Duration::from_secs(3))
            .build()
            .unwrap_or_default();

        loop {
            tokio::select! {
                res = shutdown.changed() => {
                    if res.is_err() || *shutdown.borrow() {
                        tracing::info!("Health check service received shutdown signal. Exiting...");
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
    let mut healthy_items = Vec::new();

    for item in current_items.iter() {
        if probe_backend_health(client, item).await {
            healthy_items.push(item.clone());
        } else {
            tracing::warn!(
                "Backend {} failed health check after retries",
                item.address()
            );
        }
    }

    rebuild_load_balancer(state, &healthy_items);
}

pub async fn probe_backend_health(client: &Client, item: &BackendItem) -> bool {
    let url = format!("http://{}{}", item.address(), item.health_endpoint);
    for attempt in 0..2 {
        match client.get(&url).send().await {
            Ok(resp) if resp.status().is_success() => return true,
            Ok(resp) => {
                tracing::debug!("Probe for {} returned status {}", url, resp.status());
            }
            Err(e) => {
                tracing::debug!("Probe attempt {} error for {}: {}", attempt + 1, url, e);
            }
        }
        if attempt < 1 {
            tokio::time::sleep(Duration::from_millis(500)).await;
        }
    }
    false
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::domain::models::Algorithm;

    #[test]
    fn test_health_check_service_constructor() {
        let state = DynamicLBState::new(Algorithm::RoundRobin);
        let svc = HealthCheckService::new(state, 5);
        assert_eq!(svc.interval_secs, 5);
    }
}

use crate::domain::models::DynamicLBState;
use crate::domain::routing::purge_drained_upstreams;
use async_trait::async_trait;
use pingora::services::background::BackgroundService;
use std::time::Duration;
use tracing::info;

pub struct DrainingService {
    state: DynamicLBState,
    interval: Duration,
}

impl DrainingService {
    pub fn new(state: DynamicLBState, interval: Duration) -> Self {
        Self { state, interval }
    }
}

#[async_trait]
impl BackgroundService for DrainingService {
    async fn start(&self, mut shutdown: pingora::server::ShutdownWatch) {
        let mut timer = tokio::time::interval(self.interval);
        loop {
            tokio::select! {
                _ = timer.tick() => {
                    let purged = purge_drained_upstreams(&self.state);
                    if purged > 0 {
                        info!("DrainingService: Purged {} drained upstream(s)", purged);
                    }
                }

                res = shutdown.changed() => {
                    if res.is_err() || *shutdown.borrow() {
                        tracing::info!("DrainingService received shutdown signal. Exiting...");
                        break;
                    }
                }
            }
        }
    }
}

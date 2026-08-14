use arc_swap::ArcSwap;
use clap::ValueEnum;
use pingora::lb::selection::RoundRobin;
use pingora::lb::{Backend, LoadBalancer};
use serde::{Deserialize, Serialize};
use std::sync::Arc;

#[derive(ValueEnum, Clone, Debug, PartialEq, Serialize, Deserialize, Copy)]
pub enum Algorithm {
    RoundRobin,
    Random,
    Consistent,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub struct BackendItem {
    pub ip: String,
    pub port: u16,
    pub sni_name: String,
    pub health_endpoint: String,
}

impl BackendItem {
    pub fn new(
        ip: String,
        port: u16,
        sni_name: Option<String>,
        health_endpoint: Option<String>,
    ) -> Self {
        let sni = sni_name.unwrap_or_else(|| format!("{}:{}", ip, port));
        Self {
            ip,
            port,
            sni_name: sni,
            health_endpoint: health_endpoint.unwrap_or_else(|| "/health".to_string()),
        }
    }

    pub fn address(&self) -> String {
        format!("{}:{}", self.ip, self.port)
    }

    pub fn to_pingora_backend(&self) -> Option<Backend> {
        Backend::new(&self.address()).ok()
    }
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UpstreamRegistrationPayload {
    pub ip: String,
    pub port: u16,
    pub sni_name: Option<String>,
    pub health_endpoint: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UpstreamDeregistrationPayload {
    pub ip: String,
    pub port: Option<u16>,
}

#[derive(Clone)]
pub enum DynamicLb {
    RoundRobin(Arc<LoadBalancer<RoundRobin>>),
}

#[derive(Clone)]
pub struct DynamicLBState {
    pub algorithm: Algorithm,
    pub items: Arc<ArcSwap<Vec<BackendItem>>>,
    pub lb: Arc<ArcSwap<DynamicLb>>,
}

impl DynamicLBState {
    pub fn new(algorithm: Algorithm) -> Self {
        let backends = vec!["127.0.0.1:8081", "127.0.0.1:8082", "127.0.0.1:8083"];
        let lb = LoadBalancer::try_from_iter(backends).unwrap();
        let initial_lb: DynamicLb = DynamicLb::RoundRobin(Arc::new(lb));
        Self {
            algorithm,
            items: Arc::new(ArcSwap::from_pointee(Vec::new())),
            lb: Arc::new(ArcSwap::new(Arc::new(initial_lb))),
        }
    }
}

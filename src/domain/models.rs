use super::errors::DomainError;
use arc_swap::ArcSwap;
use clap::ValueEnum;
use pingora::lb::selection::{Consistent, Random, RoundRobin};
use pingora::lb::{Backend, LoadBalancer};
use serde::{Deserialize, Serialize};
use std::net::IpAddr;
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

impl UpstreamRegistrationPayload {
    pub fn validate(&self) -> Result<(), DomainError> {
        if self.ip.parse::<IpAddr>().is_err() {
            return Err(DomainError::InvalidIpAddress(self.ip.clone()));
        }
        if self.port == 0 {
            return Err(DomainError::InvalidPort(self.port));
        }
        if let Some(ref ep) = self.health_endpoint {
            if !ep.starts_with('/') {
                return Err(DomainError::InvalidHealthEndpoint(ep.clone()));
            }
        }
        Ok(())
    }
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UpstreamDeregistrationPayload {
    pub ip: String,
    pub port: Option<u16>,
}

impl UpstreamDeregistrationPayload {
    pub fn validate(&self) -> Result<(), DomainError> {
        if self.ip.parse::<IpAddr>().is_err() {
            return Err(DomainError::InvalidIpAddress(self.ip.clone()));
        }
        if let Some(p) = self.port {
            if p == 0 {
                return Err(DomainError::InvalidPort(p));
            }
        }
        Ok(())
    }
}

#[derive(Clone)]
pub enum DynamicLb {
    RoundRobin(Arc<LoadBalancer<RoundRobin>>),
    Random(Arc<LoadBalancer<Random>>),
    Consistent(Arc<LoadBalancer<Consistent>>),
}

impl DynamicLb {
    pub fn new(algo: Algorithm, backends: Vec<Backend>) -> Self {
        match algo {
            Algorithm::RoundRobin => {
                let lb = LoadBalancer::try_from_iter(backends).unwrap();
                DynamicLb::RoundRobin(Arc::new(lb))
            }
            Algorithm::Random => {
                let lb = LoadBalancer::try_from_iter(backends).unwrap();
                DynamicLb::Random(Arc::new(lb))
            }
            Algorithm::Consistent => {
                let lb = LoadBalancer::try_from_iter(backends).unwrap();
                DynamicLb::Consistent(Arc::new(lb))
            }
        }
    }

    pub fn select(&self, key: &[u8], max_retries: usize) -> Option<Backend> {
        match self {
            DynamicLb::RoundRobin(lb) => lb.select(key, max_retries),
            DynamicLb::Random(lb) => lb.select(key, max_retries),
            DynamicLb::Consistent(lb) => lb.select(key, max_retries),
        }
    }
}

#[derive(Clone)]
pub struct DynamicLBState {
    pub algorithm: Algorithm,
    pub items: Arc<ArcSwap<Vec<BackendItem>>>,
    pub lb: Arc<ArcSwap<DynamicLb>>,
}

impl DynamicLBState {
    pub fn new(algorithm: Algorithm) -> Self {
        let initial_lb = DynamicLb::new(algorithm, Vec::new());
        Self {
            algorithm,
            items: Arc::new(ArcSwap::from_pointee(Vec::new())),
            lb: Arc::new(ArcSwap::new(Arc::new(initial_lb))),
        }
    }
}

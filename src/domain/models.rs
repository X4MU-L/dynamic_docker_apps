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
    pub fn new(ip: String, port: u16, sni_name: String, health_endpoint: Option<String>) -> Self {
        Self {
            ip,
            port,
            sni_name: sni_name.to_lowercase(),
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
    pub sni_name: String,
    pub health_endpoint: Option<String>,
}

impl UpstreamRegistrationPayload {
    pub fn validate(&mut self) -> Result<(), DomainError> {
        if self.ip.parse::<IpAddr>().is_err() {
            return Err(DomainError::InvalidIpAddress(self.ip.clone()));
        }
        if self.port == 0 {
            return Err(DomainError::InvalidPort(self.port));
        }
        self.sni_name = self.sni_name.trim().to_lowercase();
        if self.sni_name.is_empty() {
            return Err(DomainError::InvalidSniName(self.sni_name.clone()));
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

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_backend_item_lowercases_sni() {
        let item = BackendItem::new(
            "10.0.0.5".to_string(),
            9000,
            "APP-1.EDGE.LOCAL".to_string(),
            Some("/custom/health".to_string()),
        );
        assert_eq!(item.address(), "10.0.0.5:9000");
        assert_eq!(item.sni_name, "app-1.edge.local");
        assert_eq!(item.health_endpoint, "/custom/health");
    }

    #[test]
    fn test_payload_validation_lowercases_sni() {
        let mut payload = UpstreamRegistrationPayload {
            ip: "192.168.1.1".to_string(),
            port: 80,
            sni_name: "App-1.Edge.Local".to_string(),
            health_endpoint: Some("/health".to_string()),
        };
        assert!(payload.validate().is_ok());
        assert_eq!(payload.sni_name, "app-1.edge.local");
    }

    #[test]
    fn test_payload_validation_empty_sni() {
        let mut payload = UpstreamRegistrationPayload {
            ip: "192.168.1.1".to_string(),
            port: 80,
            sni_name: "".to_string(),
            health_endpoint: Some("/health".to_string()),
        };
        assert_eq!(
            payload.validate().unwrap_err(),
            DomainError::InvalidSniName("".to_string())
        );
    }
}

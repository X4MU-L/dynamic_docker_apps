use crate::domain::models::Algorithm;
use clap::Parser;

#[derive(Parser, Debug, Clone)]
#[command(author, version, about = "Dynamic Pingora Load Balancer", long_about = None)]
pub struct AppConfig {
    #[arg(short, long, value_enum, default_value_t = Algorithm::RoundRobin)]
    pub algorithm: Algorithm,

    #[arg(short, long, default_value = "0.0.0.0:80")]
    pub proxy_addr: String,

    #[arg(long, default_value = "0.0.0.0:8081")]
    pub api_addr: String,

    #[arg(long, default_value_t = 5)]
    pub health_check_interval_secs: u64,
}

impl AppConfig {
    pub fn parse_cli() -> Self {
        Self::parse()
    }
}

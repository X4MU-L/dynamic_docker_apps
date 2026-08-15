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

    pub fn parse_from_args<I, T>(args: I) -> Result<Self, clap::Error>
    where
        I: IntoIterator<Item = T>,
        T: Into<std::ffi::OsString> + Clone,
    {
        Self::try_parse_from(args)
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_app_config_defaults() {
        let args: Vec<&str> = vec!["app"];
        let config = AppConfig::parse_from_args(args).expect("Failed to parse defaults");
        assert_eq!(config.algorithm, Algorithm::RoundRobin);
        assert_eq!(config.proxy_addr, "0.0.0.0:80");
        assert_eq!(config.api_addr, "0.0.0.0:8081");
        assert_eq!(config.health_check_interval_secs, 5);
    }

    #[test]
    fn test_app_config_custom_flags() {
        let args = vec![
            "app",
            "--algorithm",
            "random",
            "--proxy-addr",
            "127.0.0.1:8000",
            "--api-addr",
            "127.0.0.1:8081",
        ];
        let config = AppConfig::parse_from_args(args).expect("Failed to parse custom flags");
        assert_eq!(config.algorithm, Algorithm::Random);
        assert_eq!(config.proxy_addr, "127.0.0.1:8000");
    }
}

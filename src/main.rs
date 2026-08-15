pub mod api;
pub mod domain;
pub mod health;
pub mod proxy;
pub mod utils;

use api::server::ApiServerService;
use domain::models::DynamicLBState;
use health::checker::HealthCheckService;
use pingora::prelude::*;
use pingora::proxy::http_proxy_service;
use pingora::services::background::background_service;
use proxy::lb::DynamicProxy;
use utils::config::AppConfig;

fn main() {
    tracing_subscriber::fmt::init();
    let config = AppConfig::parse_cli();
    tracing::info!(
        "⚡ Running Pingora Load Balancer with [{:?}] algorithm",
        config.algorithm
    );

    let state = DynamicLBState::new(config.algorithm);
    let mut my_server = Server::new(None).expect("Failed to create Pingora server");
    my_server.bootstrap();

    let api_svc = ApiServerService::new(config.api_addr.clone(), state.clone());
    my_server.add_service(background_service("api_server", api_svc));

    let health_svc = HealthCheckService::new(state.clone(), config.health_check_interval_secs);
    my_server.add_service(background_service("health_checker", health_svc));

    run_proxy_service(my_server, &config.proxy_addr, state);
}

fn run_proxy_service(mut my_server: Server, proxy_addr: &str, state: DynamicLBState) {
    let mut proxy_service = http_proxy_service(&my_server.configuration, DynamicProxy::new(state));
    proxy_service.add_tcp(proxy_addr);

    my_server.add_service(proxy_service);
    my_server.run_forever();
}

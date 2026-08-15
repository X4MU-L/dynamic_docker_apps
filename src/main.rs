pub mod api;
pub mod domain;
pub mod health;
pub mod proxy;
pub mod utils;

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
    println!("⚡ Running Pingora Load Balancer with [{:?}] algorithm", config.algorithm);

    let state = DynamicLBState::new(config.algorithm);
    let mut my_server = Server::new(None).expect("Failed to create Pingora server");
    my_server.bootstrap();

    spawn_api_server(&config, state.clone());

    let health_svc = HealthCheckService::new(state.clone(), config.health_check_interval_secs);
    my_server.add_service(background_service("health_checker", health_svc));

    run_proxy_service(my_server, &config.proxy_addr, state);
}

fn spawn_api_server(config: &AppConfig, state: DynamicLBState) {
    let api_addr = config.api_addr.clone();
    tokio::spawn(async move {
        api::server::start_api_server(&api_addr, state).await;
    });
}

fn run_proxy_service(mut my_server: Server, proxy_addr: &str, state: DynamicLBState) {
    let mut proxy_service = http_proxy_service(
        &my_server.configuration,
        DynamicProxy::new(state),
    );
    proxy_service.add_tcp(proxy_addr);

    my_server.add_service(proxy_service);
    my_server.run_forever();
}

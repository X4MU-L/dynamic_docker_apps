pub mod api;
pub mod domain;
pub mod health;
pub mod proxy;
pub mod utils;

use domain::models::DynamicLBState;
use pingora::prelude::*;
use pingora::proxy::http_proxy_service;
use proxy::lb::DynamicProxy;
use utils::config::AppConfig;

fn main() {
    tracing_subscriber::fmt::init();
    let config = AppConfig::parse_cli();
    println!(
        "⚡ Running Pingora Load Balancer with [{:?}] algorithm",
        config.algorithm
    );

    let state = DynamicLBState::new(config.algorithm);
    let rt = tokio::runtime::Builder::new_multi_thread()
        .enable_all()
        .build()
        .expect("Failed to create Tokio runtime");

    let _guard = rt.enter();
    spawn_api_and_health_services(&config, state.clone(), &rt);

    run_pingora_server(config.proxy_addr, state);
}

fn spawn_api_and_health_services(
    config: &AppConfig,
    state: DynamicLBState,
    rt: &tokio::runtime::Runtime,
) {
    let api_state = state.clone();
    let api_addr = config.api_addr.clone();
    rt.spawn(async move {
        api::server::start_api_server(&api_addr, api_state).await;
    });

    health::checker::start_health_checker(state, config.health_check_interval_secs);
}

fn run_pingora_server(proxy_addr: String, state: DynamicLBState) {
    let mut my_server = Server::new(None).expect("Failed to create Pingora server");
    my_server.bootstrap();

    let mut proxy_service = http_proxy_service(&my_server.configuration, DynamicProxy::new(state));
    proxy_service.add_tcp(&proxy_addr);

    my_server.add_service(proxy_service);
    my_server.run_forever();
}

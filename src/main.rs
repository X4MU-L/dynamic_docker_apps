pub mod api;
pub mod domain;
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

    run_pingora_server(config.proxy_addr, state);
}

fn run_pingora_server(proxy_addr: String, state: DynamicLBState) {
    let mut my_server = Server::new(None).expect("Failed to create Pingora server");
    my_server.bootstrap();

    let mut proxy_service = http_proxy_service(&my_server.configuration, DynamicProxy::new(state));
    proxy_service.add_tcp(&proxy_addr);

    my_server.add_service(proxy_service);
    my_server.run_forever();
}

use crate::domain::models::{Algorithm, DynamicLBState};
use crate::domain::routing::select_backend;
use async_trait::async_trait;
use pingora::http::RequestHeader;
use pingora::prelude::*;
use pingora::proxy::{ProxyHttp, Session};
use std::sync::atomic::{AtomicUsize, Ordering};
use std::sync::Arc;

pub struct DynamicProxy {
    pub state: DynamicLBState,
}

#[derive(Default)]
pub struct ProxyCtx {
    pub sni_name: Option<Arc<str>>,
    pub active_req_counter: Option<Arc<AtomicUsize>>,
}

impl DynamicProxy {
    pub fn new(state: DynamicLBState) -> Self {
        Self { state }
    }

    pub fn extract_key<'a>(&self, session: &'a Session) -> &'a [u8] {
        if self.state.algorithm == Algorithm::Consistent {
            session.req_header().uri.path().as_bytes()
        } else {
            b""
        }
    }
}

#[async_trait]
impl ProxyHttp for DynamicProxy {
    type CTX = ProxyCtx;
    fn new_ctx(&self) -> Self::CTX {
        ProxyCtx::default()
    }

    async fn upstream_peer(
        &self,
        session: &mut Session,
        ctx: &mut Self::CTX,
    ) -> Result<Box<HttpPeer>> {
        let key = self.extract_key(session);
        if let Some((backend, sni_name)) = select_backend(&self.state, key) {
            ctx.sni_name = Some(sni_name.clone());
            if let Some(sock_addr) = backend.addr.as_inet() {
                let addr_map = self.state.by_addr.load();
                if let Some(item) = addr_map.get(sock_addr) {
                    item.active_requests.fetch_add(1, Ordering::Relaxed);
                    ctx.active_req_counter = Some(Arc::clone(&item.active_requests));
                }
            }

            let mut peer = Box::new(HttpPeer::new(backend.addr, false, sni_name.to_string()));
            peer.options.connection_timeout = Some(std::time::Duration::from_secs(2));
            return Ok(peer);
        }

        Err(Error::explain(
            ErrorType::HTTPStatus(503),
            "No healthy upstream available in Pingora LoadBalancer",
        ))
    }

    async fn upstream_request_filter(
        &self,
        _session: &mut Session,
        upstream_request: &mut RequestHeader,
        ctx: &mut Self::CTX,
    ) -> Result<()> {
        if let Some(ref sni) = ctx.sni_name {
            upstream_request.insert_header("Host", &**sni)?;
        }
        Ok(())
    }

    async fn logging(&self, _session: &mut Session, _e: Option<&Error>, ctx: &mut Self::CTX) {
        if let Some(counter) = ctx.active_req_counter.take() {
            counter.fetch_sub(1, Ordering::Relaxed);
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::domain::models::{BackendItem, DynamicLBState};
    use crate::domain::routing::register_upstream;

    #[test]
    fn test_proxy_constructor() {
        let state = DynamicLBState::new(Algorithm::RoundRobin);
        let proxy = DynamicProxy::new(state.clone());
        assert_eq!(proxy.state.algorithm, Algorithm::RoundRobin);
    }

    #[test]
    fn test_proxy_selects_peer_with_sni() {
        let state = DynamicLBState::new(Algorithm::RoundRobin);
        let item = BackendItem::new(
            "127.0.0.1".to_string(),
            9000,
            "backend-1.edge.local".to_string(),
            None,
        );
        let _ = register_upstream(&state, item);

        let proxy = DynamicProxy::new(state);
        let selected = select_backend(&proxy.state, b"");
        assert!(selected.is_some());
        let (backend, sni) = selected.unwrap();
        assert_eq!(backend.addr.to_string(), "127.0.0.1:9000");
        assert_eq!(&*sni, "backend-1.edge.local");
    }

    #[test]
    fn test_proxy_empty_pool_returns_none() {
        let state = DynamicLBState::new(Algorithm::RoundRobin);
        let proxy = DynamicProxy::new(state);
        assert!(select_backend(&proxy.state, b"").is_none());
    }
}

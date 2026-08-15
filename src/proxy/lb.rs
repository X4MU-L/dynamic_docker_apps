use crate::domain::models::{Algorithm, DynamicLBState};
use crate::domain::routing::select_backend;
use async_trait::async_trait;
use pingora::prelude::*;
use pingora::proxy::{ProxyHttp, Session};

pub struct DynamicProxy {
    pub state: DynamicLBState,
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
    type CTX = ();
    fn new_ctx(&self) -> () {}

    async fn upstream_peer(
        &self,
        session: &mut Session,
        _ctx: &mut (),
    ) -> Result<Box<HttpPeer>> {
        let key = self.extract_key(session);
        if let Some((backend, sni_name)) = select_backend(&self.state, key) {
            let mut peer = Box::new(HttpPeer::new(
                backend.addr,
                false,
                sni_name,
            ));
            peer.options.connection_timeout = Some(std::time::Duration::from_secs(2));
            return Ok(peer);
        }

        Err(Error::explain(
            ErrorType::HTTPStatus(503),
            "No healthy upstream available in Pingora LoadBalancer",
        ))
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::domain::models::{BackendItem, DynamicLBState};
    use crate::domain::routing::register_upstream;

    #[test]
    fn test_proxy_selects_peer_with_sni() {
        let state = DynamicLBState::new(Algorithm::RoundRobin);
        let item = BackendItem::new("127.0.0.1".to_string(), 9000, Some("backend-1".to_string()), None);
        let _ = register_upstream(&state, item);

        let proxy = DynamicProxy::new(state);
        let selected = select_backend(&proxy.state, b"");
        assert!(selected.is_some());
        let (backend, sni) = selected.unwrap();
        assert_eq!(backend.addr.to_string(), "127.0.0.1:9000");
        assert_eq!(sni, "backend-1");
    }
}

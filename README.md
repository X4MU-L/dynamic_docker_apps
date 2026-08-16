# ⚡ Dynamic Docker Apps — Cloudflare Pingora Edge Load Balancer & Kubernetes-Inspired Control Plane

> **Chef's Kiss Engineering 🤌**  
> A blazing-fast, lock-free, zero-allocation **Edge Proxy and Container Orchestration Engine** built with **Cloudflare Pingora (Rust)** and a **Go Developer CLI (`deployer`)**.

---

## 🌟 Vision & Design Philosophy

`Dynamic Docker Apps` borrows core architectural patterns from **Kubernetes** to deliver a dynamic, self-healing edge proxy and container management system for Docker environments.

Instead of needing full Kubernetes clusters for small-to-medium edge deployments, `Dynamic Docker Apps` brings **K8s-style Ingress, Service Discovery, Connection Draining, Replicas, Init Containers, and Event Watchers** directly to standard Docker bridge networks (`edge-net`).

```
                              ┌─────────────────────────────────────────┐
                              │            Incoming Requests            │
                              └────────────────────┬────────────────────┘
                                                   │ (Port 80)
                                                   ▼
┌────────────────────────────────────────────────────────────────────────────────────────────────────┐
│                                 PINGORA EDGE LOAD BALANCER (Rust)                                  │
│                                                                                                    │
│   ┌───────────────────────────┐    ┌───────────────────────────┐    ┌───────────────────────────┐  │
│   │   Dynamic LB State        │    │  O(1) SocketAddr Lookup   │    │  Zero-Alloc Arc<str> SNI  │  │
│   │  (ArcSwap Lock-Free Pointer)│    │  (by_addr HashMap Index)  │    │  (Atomic Refcount Copy)   │  │
│   └─────────────┬─────────────┘    └─────────────┬─────────────┘    └─────────────┬─────────────┘  │
└─────────────────┼────────────────────────────────┼────────────────────────────────┼────────────────┘
                  │                                │                                │
                  ▼                                ▼                                ▼
  ┌──────────────────────────────┐ ┌──────────────────────────────┐ ┌──────────────────────────────┐
  │     sample-app-1 (172.30.0.10)│ │     sample-app-2 (172.30.0.11)│ │        myapp (172.30.0.4)    │
  │     (sample-app-1.edge.local)│ │     (sample-app-2.edge.local)│ │       (myapp.edge.local)     │
  └──────────────────────────────┘ └──────────────────────────────┘ └──────────────────────────────┘
```

---

## ☸️ Kubernetes-Inspired Concepts & Mechanics

| Kubernetes Concept | `Dynamic Docker Apps` Equivalent | How It Works |
| :--- | :--- | :--- |
| **Ingress Controller** | `pingora-lb` (Rust Proxy) | Single entrypoint on Port 80 routing HTTP traffic by Host/SNI header (`<app>.edge.local`) to dynamic container IPs. |
| **Endpoints & Endpointslices** | Control API & `by_addr` Index | Upstreams are registered dynamically via `POST /upstreams` and tracked in lock-free memory indices. |
| **Connection Draining (`preStop`)** | Graceful Draining (`POST /upstreams/drain`) | Immediately stops sending *new* requests to an upstream, waits for `active_requests == 0` or deadline, then evicts. |
| **Replicas (`spec.replicas`)** | Multi-Replica Deployments (`-r N`) | `./bin/cli deploy -c ./app -n app -r 3` provisions multiple isolated container instances (`app-1`, `app-2`, `app-3`). |
| **Init Containers** | `pingora-discover` Container | One-shot startup container (`depends_on: pingora-lb: condition: service_healthy`) that probes running backends, auto-registers them, and exits (`Exited (0)`). |
| **Controller Loop / Watcher** | Real-time Event Watcher (`./bin/cli watch`) | Subscribes to Docker daemon event stream (`die`, `destroy`) to automatically evict dead container IPs without human intervention. |
| **Readiness Probes** | Background Health Checker | Periodically probes `/health`, `/healthz`, `/api/health`, `/api/healthz`. Unhealthy endpoints are temporarily evicted from load balancer peer selection. |

---

## 🚀 High-Performance Rust Engineering Highlights

Every line of code in the Pingora Load Balancer is engineered for **ultra-low latency and high-throughput concurrency**:

1. **Lock-Free State Updates (`ArcSwap`)**:
   - Upstream pool updates (registration, deregistration, draining) use `ArcSwap` atomic pointer swaps. Routing threads never block or lock when pool membership changes.
2. **Zero-Allocation \(O(1)\) `SocketAddr` Lookup**:
   - Replaced linear array scans and heap string formatting (`.to_string()`) with a lock-free `HashMap<SocketAddr, BackendItem>` index (`by_addr`).
   - Pingora's native stack `SocketAddr` (`backend.addr.as_inet()`) is queried directly without any heap memory allocation.
3. **Atomic Reference Counted SNI (`Arc<str>`)**:
   - Upstream SNI hostnames are stored as `Arc<str>`. Resolving SNI per request performs an atomic pointer copy (0 nanosecond allocation cost).
4. **Relaxed Atomic Memory Ordering (`Ordering::Relaxed`)**:
   - In-flight request counters (`active_requests`) use `Ordering::Relaxed` to avoid hardware CPU memory-bus fencing (`mfence` / `lock xadd`) under high traffic.
5. **Strict 40-Line Quality Guard**:
   - Every function across the Rust engine and Go CLI is strictly **<= 40 lines of code**, enforcing modularity and testability.

---

## 🛠️ Quick Start Guide

### Option 1: Running with Docker Compose (Recommended)

Start the Pingora Edge Load Balancer, sample backends, and one-shot auto-discovery container:

```bash
docker compose up -d --build
```

#### What Happens on Startup:
1. `pingora-lb` starts listening on Port 80 (HTTP Proxy) and Port 8081 (Control API).
2. Docker Compose waits for `pingora-lb` health check to report `Healthy`.
3. `pingora-discover` init container launches, auto-discovers all running backends on `edge-net`, registers them with `pingora-lb`, and **exits with code 0 (`Exited (0)`)**.

---

### Option 2: Running Bare Commands (Manual Execution)

If you prefer running components directly:

#### 1. Start the Pingora LB Container
```bash
docker compose up -d --build pingora-lb
```

#### 2. Run Auto-Discovery via Go CLI
```bash
./bin/cli discover --api-url http://localhost:8081
```

#### 3. Deploy an App with Replicas
```bash
./bin/cli deploy -c ./sample_app -n myapp -r 2
```

#### 4. List Active Upstreams
```bash
./bin/cli list
```

#### 5. Gracefully Draining & Stop an Upstream
```bash
./bin/cli deregister -n myapp-1 -s -t 10
```

---

## 🖥️ Developer CLI Deep-Dive (`./bin/cli` / `deployer`)

The Go CLI (`deployer`) provides a full suite of subcommands:

```text
Usage:
  deployer <command> [flags]

Commands:
  deploy      Build/pull and run container replicas on edge-net and register with Pingora
  deregister  Evict an upstream from Pingora by container name or IP address
  discover    Scan running containers, probe health endpoints, and register active backends
  list        List all active Pingora upstreams
  watch       Listen for Docker container death events and auto-evict endpoints
```

### 1. `deploy` Subcommand
Builds a local Dockerfile context or pulls a remote image, runs $N$ replicas on `edge-net`, probes health, and registers upstreams with Pingora.

```bash
# Build local context with 3 replicas:
./bin/cli deploy -c ./sample_app -n myapp -r 3 -p 8080 -d edge.local

# Deploy from private/public Docker registry:
./bin/cli deploy -i myregistry.com/my-app:v1.2 -u admin --password secret -n prod-app -r 2
```

### 2. `deregister` Subcommand
Initiates connection-aware graceful draining or force eviction for an upstream.

```bash
# Gracefully drain for 15 seconds and stop container:
./bin/cli deregister -n myapp-1 -s -t 15

# Immediate force eviction by IP address:
./bin/cli deregister --ip 172.30.0.4 -p 8080 -t 0
```

### 3. `discover` Subcommand
Scans running containers on `edge-net`, probes candidate health paths (`/health`, `/healthz`, `/api/health`, `/api/healthz`), and registers healthy backends.

```bash
./bin/cli discover --api-url http://localhost:8081
```

### 4. `list` Subcommand
Lists all currently active upstreams, their IP address, port, SNI hostname, status, active requests, and remaining drain duration.

```bash
./bin/cli list
```

### 5. `watch` Subcommand
Runs a long-running event listener watching Docker daemon `die` and `destroy` container events to auto-evict dead containers in real time.

```bash
./bin/cli watch --network edge-net
```

---

## 📡 Pingora Control API Specification (`http://localhost:8081`)

### 1. `POST /upstreams` — Register Upstream
```json
// Request Body
{
  "ip": "172.30.0.10",
  "port": 8080,
  "sni_name": "sample-app-1.edge.local",
  "health_endpoint": "/health"
}
```

### 2. `POST /upstreams/drain` — Mark Draining
```json
// Request Body
{
  "ip": "172.30.0.10",
  "port": 8080,
  "drain_timeout_secs": 15
}
```

### 3. `GET /upstreams/status` — Upstream Status Query
```bash
GET /upstreams/status?ip=172.30.0.10&port=8080
```

### 4. `DELETE /upstreams` — Deregister Upstream
```json
// Request Body
{
  "ip": "172.30.0.10",
  "port": 8080
}
```

### 5. `GET /upstreams` — List All Upstreams
Returns a JSON array of all registered upstreams and their live metrics.

---

## 🧪 Testing Suite

### Run Go CLI Unit Tests
```bash
go test ./... -v
```

### Run Rust Load Balancer Integration & Unit Tests
```bash
cargo test
```

---

## 📜 License

MIT License. Designed with ❤️ using Cloudflare Pingora & Go.

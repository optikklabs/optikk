# Optikk — Kubernetes deployment

Kustomize manifests to run the full Optikk stack on a **local kind cluster**. The root
`kustomization.yaml` is a single flat layer — every platform component in the `optikk`
namespace (default storageClass, Traefik on NodePorts, placeholder secrets). `optikk up`
(or `kubectl apply -k .` from this directory) deploys everything. There are no overlays and
no per-tenant manifests: tenants are rows in MySQL created at signup, and `ingest` resolves
them per-request from the `x-api-key` header.

## Components

| Component | Kind | Image | Role |
|---|---|---|---|
| ClickHouse | StatefulSet | `clickhouse/clickhouse-server:24.8` | OLAP store (spans/logs/metrics) |
| MariaDB | StatefulSet | `mariadb:11.4` | Relational store (users, dashboards, monitors) |
| mq | StatefulSet | `ghcr.io/ramantayal12/mq:latest` | Kafka-compatible broker (AutoMQ-style tiered storage) |
| ingest | Deployment + HPA | `ghcr.io/optikklabs/ingest:latest` | OTLP intake → mq → ClickHouse/MariaDB |
| query | Deployment + HPA | `ghcr.io/optikklabs/query:latest` | Read/API server |
| traefik | Deployment + HPA | `traefik:v3.3` | Edge proxy / only public Service |

## Network model

Everything runs in namespace **`optikk`** and resolves peers by Service DNS
(`clickhouse:9000`, `mariadb:3306`, `mq:9092`, `ingest:18317`/`18318`, `query:19090`).
**Only Traefik is public** — it fronts the two external surfaces, each with a
`rateLimit` middleware and TLS termination:

```text
                 ┌────────────────────── Traefik (public) ───────────────────────┐
agents  ──OTLP──▶ :4317 grpc / :4318 http ──▶ Header(x-api-key) ─────────────┐
                                                                             │
clients ──HTTP──▶ :9080 / :9443 ────────────▶ query ──┐                      │
                 └────────────────────────────────────┼──────────────────────┼───┘
                                                      ▼                      ▼
                                     ClickHouse / MariaDB     ingest ──▶ mq ──▶ ClickHouse/MariaDB
```

`query` and `ingest` Services are ClusterIP (internal only). Traefik forwards the
`x-api-key` header through to `ingest`, which serves OTLP on `18317` (gRPC) / `18318`
(HTTP) and resolves the tenant from that header.

## Multi-tenant ingestion

`query` and `ingest` are single, shared Deployments — there are no per-tenant manifests.
`ingest` resolves the tenant per-request from the `x-api-key` header against the `tenant`
table in MySQL. `query` authenticates human users via JWT (`dtid`/`tids` claims +
`TenantMiddleware`), which needs no per-tenant routing at all since every query is already
scoped by `team_id` inside the query itself.

Onboarding a tenant is therefore just `POST /v1/auth/signup` (which inserts the `tenant`
row and mints its `api_key`); no manifest changes are required to start accepting its
traffic. Deactivating a tenant (`tenant.active = 0`) stops it at the `ingest` auth check.

## Layout

```
kustomization.yaml # root: single flat layer over base/ (apply -k .)
base/              # one definition of every component (+ HPAs, Traefik CRDs/RBAC/routes)
kind/              # kind cluster config (host port mappings)
```

Both services read config via `OPTIKK_*` env vars (viper `AutomaticEnv`), so hosts/ports are
set in the manifests with no baked config files. Schema is **self-migrating on boot** —
ingest applies the ClickHouse DDL, query applies the MySQL DDL; no init job is required.

---

## Run locally (kind on Podman)

**Prerequisites:** `kubectl`, `kind`, a running Podman machine. kind needs a **rootful**
Podman machine on macOS:

```bash
podman machine stop && podman machine set --rootful && podman machine start   # if not already rootful
```

> Heads-up: toggling rootful restarts the Podman VM (stops running containers). If you'd
> rather not, use `minikube start --driver=podman` and apply the same overlay.

### 1. Create the cluster

```bash
cd deploy
KIND_EXPERIMENTAL_PROVIDER=podman kind create cluster --name optikk --config kind/kind-config.yaml
```

Podman defaults new containers to `--pids-limit=2048`, and the kind node is itself just one
Podman container hosting the whole nested cluster (kubelet + containerd + CoreDNS + every pod
share that one budget). ClickHouse alone needs >512 threads on boot, so once a few pods are
running the shared budget can run out and ClickHouse aborts with `Couldn't get 512 threads from
global thread pool`. Lift the limit on the node container right after creating it:

```bash
podman update --pids-limit=-1 optikk-control-plane
```

### 2. metrics-server (for HPAs)

```bash
kubectl apply -f https://github.com/kubernetes-sigs/metrics-server/releases/latest/download/components.yaml
kubectl -n kube-system patch deployment metrics-server --type=json \
  -p='[{"op":"add","path":"/spec/template/spec/containers/0/args/-","value":"--kubelet-insecure-tls"}]'
```

### 3. App images

The `ingest`/`query` packages on ghcr are private until made public. Either make them public
(see cloud step 2), or build and load them locally:

```bash
podman build -t ghcr.io/optikklabs/ingest:latest ../ingest
podman build -t ghcr.io/optikklabs/query:latest  ../query
kind load docker-image ghcr.io/optikklabs/ingest:latest ghcr.io/optikklabs/query:latest --name optikk
```

### 4. Deploy

```bash
kubectl apply -k .
kubectl -n optikk rollout status statefulset/clickhouse statefulset/mariadb statefulset/mq
kubectl -n optikk wait --for=condition=available --timeout=180s deploy/ingest deploy/query deploy/traefik
kubectl -n optikk get pods
```

### 5. Verify

```bash
# Query API through Traefik (kind maps host 18040 -> Traefik web)
curl -fsS http://localhost:18040/health

# Send an OTLP/HTTP trace through Traefik -> ingest -> mq -> ClickHouse.
# x-api-key must match a signed-up tenant's key so ingest resolves the tenant.
curl -fsS -X POST http://localhost:4318/v1/traces \
  -H 'Content-Type: application/json' -H 'x-api-key: c3448fae' --data @example-trace.json

# Confirm it landed
kubectl -n optikk exec statefulset/clickhouse -- \
  clickhouse-client --password "$CH_PASSWORD" --query "SELECT count() FROM optikk.spans"
```

### Teardown

```bash
kubectl delete -k .
kind delete cluster --name optikk
```

---

## Sizing & scaling

**Local (kind) — minimum to run everything:**

| Component | CPU req | Mem req | PVC |
|---|---|---|---|
| ClickHouse | 500m | 1Gi (limit 2Gi) | 5Gi |
| MariaDB | 250m | 512Mi | 2Gi |
| mq | 250m | 512Mi | 5Gi |
| ingest / query | 100m | 128Mi | — |
| traefik | 100m | 128Mi | — |
| **Total** | **~1.5 vCPU** | **~3 Gi** | **~12Gi** |

Fits a 5 vCPU / 8 GiB Podman machine with one tenant. Bump the machine
(`podman machine set --cpus … --memory …`) if tight, or as tenant count grows.

**How to scale:**

- **Stateless (ingest/query/traefik)** — automatic via HPA (CPU 70%,
  1→5 replicas). Change the ceiling in `base/<svc>/hpa.yaml` (`maxReplicas`) or give more
  headroom by raising `resources`.
- **ClickHouse vertically** — edit its `resources` in `base/clickhouse/statefulset.yaml`,
  then `kubectl -n optikk rollout restart statefulset/clickhouse`. **Grow its disk** (if the
  StorageClass supports online expansion):
  ```bash
  kubectl -n optikk patch pvc data-clickhouse-0 \
    -p '{"spec":{"resources":{"requests":{"storage":"200Gi"}}}}'
  ```
- **MariaDB / mq** — same vertical pattern (resources + PVC expansion).
- **True horizontal ClickHouse** (sharding/replication) needs a Keeper ensemble + macros +
  Distributed tables — larger change, out of scope here.

## Visibility (what's running)

No in-cluster dashboard is deployed (keeps the footprint minimal). Use:

- Any cluster: `kubectl get pods -n optikk -w`, or [`k9s`](https://k9scli.io/).

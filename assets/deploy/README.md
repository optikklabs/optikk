# Optikk — Kubernetes deployment

Kustomize manifests to run the full Optikk stack on **kind locally** and on **GKE**, from a
single base with two overlays. The only differences between environments are the storage
class, the Traefik Service type, and where `mq` / ClickHouse store data (local disk vs GCS).

## Components

| Component | Kind | Image | Role |
|---|---|---|---|
| ClickHouse | StatefulSet | `clickhouse/clickhouse-server:24.8` | OLAP store (spans/logs/metrics) |
| MariaDB | StatefulSet | `mariadb:11.4` | Relational store (users, dashboards, monitors) |
| mq | StatefulSet | `ghcr.io/ramantayal12/mq:latest` | Kafka-compatible broker (AutoMQ-style tiered storage) |
| ingest | Deployment + HPA | `ghcr.io/optikklabs/ingest:latest` | OTLP intake → mq → ClickHouse/MariaDB |
| query | Deployment + HPA | `ghcr.io/optikklabs/query:latest` | Read/API server |
| otel-collector | Deployment + HPA, **one per tenant** | `otel/opentelemetry-collector-contrib:0.155.0` | Telemetry pipeline → ingest |
| traefik | Deployment + HPA | `traefik:v3.3` | Edge proxy / only public Service |

## Network model

Everything runs in namespace **`optikk`** and resolves peers by Service DNS
(`clickhouse:9000`, `mariadb:3306`, `mq:9092`, `ingest:4318`, `query:19090`,
`otel-collector-<tenant>:4317/4318`). **Only Traefik is public** — it fronts the two
external surfaces, each with a `rateLimit` middleware and TLS termination:

```
                 ┌────────────────────── Traefik (public) ───────────────────────┐
agents  ──OTLP──▶ :4317 grpc / :4318 http ──▶ Header(x-api-key) ──▶ otel-collector-<tenant> ──┐
                                               (one dedicated collector             │
                                                per tenant, selected by             │
                                                the caller's own key —              │
                                                see "Multi-tenant otel-collector"    │
                                                below)                              │
clients ──HTTP──▶ :80 / :443 ───────────────▶ query ──┐                            │
                 └───────────────────────────────────┼────────────────────────────┘
                                                        ▼                           ▼
                                    ClickHouse / MariaDB     ingest ──▶ mq ──▶ ClickHouse/MariaDB
```

Each otel-collector has no L7 rate limiter (only `memory_limiter` backpressure), so external
OTLP is deliberately routed through Traefik. `query` and every `otel-collector-<tenant>`
Service are ClusterIP (internal only).

## Multi-tenant otel-collector

There is no shared/default otel-collector — `deploy/base/otel-collector/` is a
template, instantiated once per tenant under `deploy/tenants/<id>/` with that
tenant's real API key baked in as `OPTIKK_API_KEY`. Traefik routes each request to
the right tenant's collector by matching the `x-api-key` the caller sends
(`Header(`x-api-key`, ...)` on the `otlp-http`/`otlp-grpc` entrypoints) — clients keep
sending to the same public host/ports regardless of which tenant they are. A request
whose key doesn't match any tenant gets a 404 before it reaches any collector.

This exists because `spanmetrics`/`servicegraph`/`exceptions`/`count` are
**connectors** — they synthesize new telemetry inside the collector with no inbound
request behind it, so there is no per-request header for a shared collector to
forward for that data. Giving each tenant a dedicated collector sidesteps this
entirely: since a collector only ever sees one tenant's traffic, its existing static
`headers_setter` (`value: ${env:OPTIKK_API_KEY}`) is correct for every pipeline,
including the connector-derived ones.

`query` and `ingest` are **not** part of this pattern — they stay single, shared
Deployments. `ingest` resolves the tenant per-request from `tenant.api_key` (same
mechanism, just at the next hop). `query` authenticates human users via JWT
(`dtid`/`tids` claims + `TenantMiddleware`), which needs no per-tenant routing at all
since every query is already scoped by `team_id` inside the query itself.

### Onboarding a tenant

After `POST /v1/auth/signup` (pre-auth) returns a new `{tenant, api_key}` —
or simply run `optikk onboard --local`, which does all of this for you:

1. `cp -r deploy/tenants/_template deploy/tenants/<id>`
2. Replace every `<id>` and `REPLACE_WITH_<TENANT_KEY>` placeholder in the new
   directory's `kustomization.yaml` and `ingressroute.yaml` with the real id and key.
3. Apply the real key out-of-band — **do not commit it**:
   ```bash
   kubectl create secret generic otel-collector-<id>-secret -n optikk \
     --from-literal=api-key='<real key>' --dry-run=client -o yaml | kubectl apply -f -
   ```
4. Add `../../tenants/<id>` to the target overlay's `resources:`
   (`overlays/local/kustomization.yaml` or `overlays/gcp/kustomization.yaml`).
5. `kubectl apply -k deploy/overlays/<env>`

This is a **manual** step, not automated — `tenant` in MySQL and the tenant
directories/overlay list here are two separate sources of truth. Deactivating a tenant
(`tenant.active = 0`) does not by itself stop its collector.

**Offboarding**: remove the tenant's line from the overlay's `resources:`,
`kubectl delete -k deploy/tenants/<id> -n optikk`, delete `deploy/tenants/<id>`.

## Layout

```
base/            # one definition of every component (+ HPAs, Traefik CRDs/RBAC/routes)
overlays/local/  # kind: default storageClass, Traefik NodePort, mq on local disk
overlays/gcp/    # GKE: premium-rwo SSD, Traefik LoadBalancer, mq+ClickHouse on GCS
kind/            # kind cluster config (host port mappings)
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
kubectl apply -k overlays/local
kubectl -n optikk rollout status statefulset/clickhouse statefulset/mariadb statefulset/mq
kubectl -n optikk wait --for=condition=available --timeout=180s deploy/ingest deploy/query deploy/otel-collector-local deploy/traefik
kubectl -n optikk get pods
```

### 5. Verify

```bash
# Query API through Traefik (kind maps host 8080 -> Traefik web)
curl -fsS http://localhost:8080/health

# Send an OTLP/HTTP trace through Traefik -> otel-collector-local -> ingest -> mq -> ClickHouse.
# x-api-key must match the "local" tenant's key (deploy/tenants/local) so Traefik
# routes it to a collector at all — see "Multi-tenant otel-collector" above.
curl -fsS -X POST http://localhost:4318/v1/traces \
  -H 'Content-Type: application/json' -H 'x-api-key: c3448fae' --data @example-trace.json

# Confirm it landed
kubectl -n optikk exec statefulset/clickhouse -- \
  clickhouse-client --password "$CH_PASSWORD" --query "SELECT count() FROM optikk.spans"
```

### Teardown

```bash
kubectl delete -k overlays/local
kind delete cluster --name optikk
```

---

## Deploy on GCP (GKE + GCS)

### 1. Cluster

```bash
gcloud container clusters create optikk \
  --region=us-central1 --num-nodes=1 \
  --enable-autoscaling --min-nodes=3 --max-nodes=6 \
  --machine-type=e2-standard-4
gcloud container clusters get-credentials optikk --region=us-central1
```

### 2. Make the app images pullable

Set the `optikklabs/ingest` and `optikklabs/query` GitHub Packages to **public**
(Package → Settings → Change visibility), so no pull secret is needed.

### 3. GCS buckets + interop (HMAC) keys

`mq` and ClickHouse's cold tier use GCS via its **S3-compatible interop API**, which needs
**HMAC** keys.

```bash
gcloud storage buckets create gs://optikk-mq-data       --location=US
gcloud storage buckets create gs://optikk-ch-cold        --location=US
# HMAC keys for a service account with objectAdmin on both buckets:
gcloud storage hmac create SA_EMAIL          # note the accessId + secret it prints
```

Then put the bucket names into `overlays/gcp/patches/mq-gcs.yaml`
(`KAFKA_OBJECT_BUCKET`) and `overlays/gcp/patches/clickhouse-gcs.yaml`
(`REPLACE_CH_COLD_BUCKET`).

> For production, prefer VPC **Private Google Access** to GCS to avoid egress costs.

### 4. Namespace + secrets

```bash
kubectl create namespace optikk

# App credentials (override the in-repo placeholders with real values)
kubectl create secret generic clickhouse-secret -n optikk --from-literal=password='<ch-pass>'
kubectl create secret generic mariadb-secret    -n optikk --from-literal=root-password='<my-pass>'
kubectl create secret generic query-secret       -n optikk --from-literal=jwt-secret='<32+ chars>'

# otel-collector has no default secret to create here — it's onboarded per tenant,
# see "Multi-tenant otel-collector" > "Onboarding a tenant" above.

# GCS HMAC keys (see secrets-example/ for the shape)
kubectl create secret generic mq-gcs-secret -n optikk \
  --from-literal=access-key='<HMAC_ACCESS_ID>' --from-literal=secret-key='<HMAC_SECRET>'
kubectl create secret generic ch-gcs-secret -n optikk \
  --from-literal=access-key='<HMAC_ACCESS_ID>' --from-literal=secret-key='<HMAC_SECRET>'
```

### 5. Deploy + reach it

```bash
kubectl apply -k overlays/gcp
kubectl -n optikk get pods
kubectl -n optikk get svc traefik -w   # wait for EXTERNAL-IP
# query API  -> http://<EXTERNAL-IP>/        OTLP -> <EXTERNAL-IP>:4317 / :4318
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
| otel-collector | 100m | 256Mi | — |
| traefik | 100m | 128Mi | — |
| **Total** | **~1.5 vCPU** | **~3 Gi** | **~12Gi** |

`otel-collector` is per tenant — the row above is the cost of **one** tenant's
collector; total collector footprint scales linearly with the number of onboarded
tenants (`deploy/tenants/*`), not with total request volume alone. Fits a 5 vCPU / 8
GiB Podman machine with one tenant. Bump the machine (`podman machine set --cpus … --memory …`)
if tight, or as tenant count grows.

**Cloud (GKE) — recommended start:** 3→6 × `e2-standard-4` (autoscaling). Stateful requests
in `overlays/gcp` patches: ClickHouse 2 vCPU/8Gi (+100Gi SSD +GCS cold), MariaDB 1 vCPU/4Gi
(20Gi), mq 1 vCPU/2Gi (data in GCS). Stateless services start small and scale via HPA.

**How to scale:**

- **Stateless (ingest/query/traefik/otel-collector)** — automatic via HPA (CPU 70%,
  1→5 replicas). Change the ceiling in `base/<svc>/hpa.yaml` (`maxReplicas`) or give more
  headroom by raising `resources`. For `otel-collector`, `base/otel-collector/hpa.yaml`
  is the shared template every tenant is instantiated from — edit it to change the
  default for all tenants, or add a per-tenant `patches:` entry in
  `deploy/tenants/<id>/kustomization.yaml` to size one tenant differently (e.g. a
  low-volume tenant that doesn't need 5 replicas of headroom).
- **ClickHouse vertically** — edit the `resources` in `overlays/gcp/patches/clickhouse-gcs.yaml`,
  then `kubectl -n optikk rollout restart statefulset/clickhouse`. **Grow its disk** (premium-rwo
  supports online expansion):
  ```bash
  kubectl -n optikk patch pvc data-clickhouse-0 \
    -p '{"spec":{"resources":{"requests":{"storage":"200Gi"}}}}'
  ```
  **Node capacity:** raise the node-pool machine type or `--max-nodes`.
- **MariaDB / mq** — same vertical pattern (resources + PVC expansion). mq offloads to GCS so
  its PVC stays small.
- **True horizontal ClickHouse** (sharding/replication) needs a Keeper ensemble + macros +
  Distributed tables — larger change, out of scope here.

## Visibility (what's running)

No in-cluster dashboard is deployed (keeps the footprint minimal). Use:

- Local / any cluster: `kubectl get pods -n optikk -w`, or [`k9s`](https://k9scli.io/).
- GKE: **Cloud Console → Kubernetes Engine → Workloads** (free, shows every pod).

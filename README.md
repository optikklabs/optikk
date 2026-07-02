# optikk

A single CLI to provision the **entire Optikk stack from prebuilt container images** and
operate it — on a **local kind cluster** or on **Google Cloud (GKE)**. It turns a manual,
order-sensitive runbook into a handful of health-gated commands.

- **Module:** `github.com/optikklabs/optikk` · **Go:** 1.26 · **CLI:** Cobra
- The Kubernetes manifests are **embedded** in the binary (`assets/deploy`, via `go:embed`) —
  the CLI **renders** them (kustomize Go API) and **server-side-applies** them (client-go), so
  it is fully self-contained with no external manifest tree to clone.
- Environment is a **`--target local|gcp` flag**, not a subcommand. The top-level verbs
  (`up`, `down`, `status`, `verify`, `tenant`, `admin`, `team`) provision infra from scratch.
- **Native Go SDKs** throughout (kind lib, client-go SSA, kustomize, GKE + GCS SDKs). The only
  shell-outs are two host-level Podman steps that have no Go SDK, and `podman save`.

---

## Contents

- [What `up` deploys](#what-up-deploys)
- [Container images used](#container-images-used)
- [Architecture](#architecture)
- [Data flow: how a trace lands](#data-flow-how-a-trace-lands)
- [`up` provisioning flow](#up-provisioning-flow)
- [Install / build](#install--build)
- [Quick start (local)](#quick-start-local)
- [Command reference](#command-reference)
- [Configuration](#configuration)
- [Resource sizing](#resource-sizing)
- [Project layout](#project-layout)
- [Extending the CLI](#extending-the-cli)
- [Troubleshooting](#troubleshooting)
- [Status](#status)

---

## What `up` deploys

`up` brings up the **whole stack** in the `optikk` namespace. No component is optional in v1.

| Component | Role |
|---|---|
| **traefik** | Ingress. Routes `/api`, `/swagger`, `/health` to query; `/` to web; `x-api-key` header to the matching tenant otel-collector; `:4318` OTLP to ingest. |
| **web** | Dashboard UI (nginx). |
| **query** | Read/query API + auth (JWT). Seeds the super-admin on boot. |
| **ingest** | Validates the tenant `x-api-key` against MariaDB, writes spans toward ClickHouse via mq. |
| **otel-collector** | **One per tenant.** Receives OTLP, forwards to ingest. Created by `tenant onboard`. |
| **mq** | Message queue / buffering between ingest and ClickHouse. |
| **clickhouse** | Span store (`optikk.spans`). |
| **mariadb** | Tenant/team + identity store. |
| **metrics-server** | Installed by `up --target local` (with `--kubelet-insecure-tls`) so HPA/`kubectl top` work on kind. |

---

## Container images used

The CLI **never builds images inside the cluster** — it applies manifests and Kubernetes pulls
the images. **All images are public**, so the default path needs no registry auth.

### Application images (`ghcr`, public)

| Image | Component |
|---|---|
| `ghcr.io/optikklabs/ingest:latest` | ingest |
| `ghcr.io/optikklabs/query:latest` | query |
| `ghcr.io/optikklabs/web:latest` | web |
| `ghcr.io/ramantayal12/mq:latest` | mq |

These ghcr packages are public — Kubernetes pulls them directly, no credentials required.

**Optional `--load-local-images` (local target):** for air-gapped/offline runs or when testing a
locally-built image, pass this flag to `up --target local`. The CLI runs `podman save -m` on the
four images already present on the host and imports the archive into the kind node via the **kind
Go library** (`nodeutils.LoadImageArchive`) — which also sidesteps the containerd-config-version
mismatch of the external `kind load` CLI. Not needed for a normal run.

### Upstream images (public)

| Image | Component |
|---|---|
| `clickhouse/clickhouse-server:26.6` | clickhouse |
| `mariadb:11.4` | mariadb |
| `otel/opentelemetry-collector-contrib:0.155.0` | otel-collector (per tenant) |
| `traefik:v3.3` | traefik |
| `busybox:1.36` | init containers (ingest/query) |

Pulled directly from their public registries; never loaded locally.

---

## Architecture

```mermaid
flowchart LR
    browser([Browser]) -->|/ , /api| traefik
    otlp([OTLP client]) -->|x-api-key| traefik
    otlp -->|:4318 OTLP| traefik

    subgraph ns["optikk namespace"]
        traefik["traefik<br/>(ingress)"]
        web["web<br/>(dashboard UI)"]
        query["query<br/>(API + auth)"]
        collector["otel-collector-&lt;tenant&gt;<br/>(one per tenant)"]
        ingest["ingest"]
        mq["mq<br/>(queue)"]
        clickhouse[("clickhouse<br/>optikk.spans")]
        mariadb[("mariadb<br/>teams + identity")]

        traefik -->|/| web
        traefik -->|/api, /health| query
        traefik -->|x-api-key route| collector
        collector --> ingest
        traefik -->|:4318| ingest
        ingest -->|validate key| mariadb
        ingest --> mq
        mq --> clickhouse
        query -->|read spans| clickhouse
        query -->|seed admin, teams| mariadb
    end
```

**Per-target ingress / storage differences:**

| | Target = local (kind on Podman) | Target = gcp (GKE) |
|---|---|---|
| Query/UI ingress | host `:8080` → Traefik web | Traefik Service → LoadBalancer external IP |
| OTLP ingress | host `:4318` → ingest | LoadBalancer IP `:4318` |
| mq + ClickHouse cold tier | local PVCs | GCS buckets (HMAC keys) |

---

## Data flow: how a trace lands

`verify` exercises this exact path end-to-end.

```mermaid
sequenceDiagram
    participant C as client
    participant T as traefik
    participant O as otel-collector-&lt;tenant&gt;
    participant I as ingest
    participant M as mariadb
    participant Q as mq
    participant CH as clickhouse

    C->>T: OTLP POST /v1/traces (x-api-key: KEY)
    T->>O: route by x-api-key
    O->>I: forward spans
    I->>M: validate KEY against teams
    M-->>I: tenant ok
    I->>Q: enqueue spans
    Q->>CH: write to optikk.spans
    Note over C,CH: verify then polls SELECT count() FROM optikk.spans (async, ~30s)
```

Two gotchas the CLI accounts for:

1. **A fresh cluster has no tenant whose key matches the collector.** Teams are created via the
   API (`team create`), not seeded. So the first-run order is: `up → admin login → team create →
   tenant onboard --key <key> → verify --api-key <key>`.
2. **ingest caches a failed key lookup for ~5 minutes** (negative cache). If you seed/create a
   team *after* a key was already tried, `kubectl -n optikk rollout restart deploy/ingest` clears it.

---

## `up` provisioning flow

### `up --target local`

```
precheck podman machine (rootful? running? ≥5 vCPU / ≥8 GiB / ≥40 GiB disk)
  └─ short? fail with the exact `podman machine set ...` command  (or run it with --manage-podman)
create kind cluster "optikk"  (deploy/kind/kind-config.yaml)   [reused if it already exists]
lift pids-limit on the node container
[--load-local-images] podman save 4 app images  ->  kind LoadImageArchive
install metrics-server (+ --kubelet-insecure-tls)
render deploy/overlays/local (kustomize)  ->  server-side apply (CRDs before CRs, retry)
wait for all Deployments + StatefulSets to roll out
ready -> query API at http://localhost:8080
```

### `up --target gcp`

```
validate --project/--region/--mq-bucket/--ch-bucket/--hmac-sa
create GKE cluster "optikk"  (autoscaling pool, container/apiv1)
create GCS buckets (mq + ClickHouse cold tier)   [409 already-exists = ok]
mint HMAC key for --hmac-sa
create namespace + GCS secrets (mq-gcs-secret, ch-gcs-secret)
render deploy/overlays/gcp with bucket names substituted  ->  server-side apply
wait for rollouts, then wait for the Traefik LoadBalancer external IP
ready -> query API at http://<external-ip>
```

`down` reverses it: local deletes the kind cluster (or just the stack with `--keep-cluster`);
gcp deletes the GKE cluster and the buckets (unless `--keep-buckets`).

---

## Install

The binary is **self-contained** — the Kubernetes manifests are embedded, so `optikk` runs
from any directory with no external files. Local target additionally needs **Podman** (rootful
machine, ≥8 GiB RAM); GCP target needs **Application Default Credentials**
(`gcloud auth application-default login`).

```bash
# Go (any platform, Go 1.26+)
go install github.com/optikklabs/optikk@latest

# Raw binary (macOS/Linux, amd64/arm64) — from the GitHub Release
curl -L https://github.com/optikklabs/optikk/releases/latest/download/optikk_$(uname -s)_$(uname -m).tar.gz | tar xz
./optikk --help
```

### Run via `go install`

`go install` drops the `optikk` binary in `$(go env GOPATH)/bin`. Add that to your `PATH`, then run it directly:

```bash
go install github.com/optikklabs/optikk@latest
export PATH="$(go env GOPATH)/bin:$PATH"   # add to your shell profile to persist

optikk --help      # verify install
optikk up          # provision the local (kind) stack
```

Build from source: `go build -o optikk .`. Developing against a live manifest tree instead of
the embedded copy? Point at it with `--deploy-dir PATH`.

---

## Quick start (local)

```bash
# 1. Provision cluster + full stack (public images pulled by Kubernetes)
optikk up                                # add --load-local-images only for offline/local builds

# 2. Health + trace roundtrip against the default tenant key
optikk verify

# 3. Seed + cache the platform super-admin
optikk admin setup
optikk admin login                       # defaults: admin@optikk.dev / Password123!

# 4. Create a team -> get its api_key (this is the tenant key)
optikk team create demo                  # prints team_id, slug, api_key K
optikk team member add u@x.com --team <id> --password 'Secret123!'

# 5. Onboard a per-tenant collector and verify with its key
optikk tenant onboard demo --key K
optikk verify --api-key K

# 6. Inspect / tear down
optikk status
optikk down
```

---

## Command reference

Persistent flags (all commands): `--target local|gcp` (default `local`) · `--config PATH` ·
`--deploy-dir PATH` · `--verbose/-v`.

### `optikk up`
Provision infra from scratch + deploy the full stack for `--target`.

- local: `[--manage-podman] [--load-local-images] [--timeout 10m]`
- gcp: `--project ID --region R [--nodes 1] [--min-nodes 3] [--max-nodes 6]`
  `[--machine-type e2-standard-4] [--mq-bucket NAME] [--ch-bucket NAME] [--hmac-sa EMAIL] [--timeout 10m]`

### `optikk down`
Tear down the stack + the cluster it created.

- local: `[--keep-cluster]` · gcp: `--project ID --region R [--keep-buckets]`

### `optikk status`
List `optikk`-namespace pods + readiness for `--target`.

### `optikk verify`
`/health` 200 → POST one OTLP trace with `x-api-key` → assert the ClickHouse span count rose
(polls up to 30s; ingestion is async). `[--api-key c3448fae] [--trace-file <deploy>/example-trace.json]`

### `optikk tenant onboard <slug> --key KEY` / `optikk tenant offboard <slug>`
Materialize `deploy/tenants/_template` for `<slug>`, create `otel-collector-<slug>-secret`,
render + apply (or delete) the per-tenant otel-collector.

### `optikk admin setup` / `optikk admin login`
- `setup [--email E] [--password P]` — patch query's admin secret and restart so query reseeds
  the super-admin (**create-if-absent**: an existing admin's password is unchanged).
- `login [--email E] [--password P]` — `POST /api/v1/auth/login`, cache the JWT at
  `~/.optikk/token.json` for the `team` commands.

### `optikk team create <name>` / `optikk team member add <email>`
- `create <name> [--org O] [--slug S]` — admin-gated; prints `team_id`, `slug`, `api_key`
  (the `api_key` is what `tenant onboard --key` consumes).
- `member add <email> --team ID --password P [--name N] [--role R]` — admin-gated; creates a
  user assigned to the team (there is no dedicated member endpoint — this maps to create-user).

### `optikk config show` · `optikk version`
Print the merged config (flags + `optikk.yaml`/`~/.optikk` + gcloud fallback) / the version.

---

## Configuration

Precedence: **flags → `optikk.yaml` (cwd) or `~/.optikk/config` → `OPTIKK_*` env → gcloud active
config** (project/region fallback for gcp). Example `optikk.yaml`:

```yaml
target: gcp
gcp:
  project: my-gcp-project
  region: us-central1
  machine_type: e2-standard-4
  mq_bucket: optikk-mq-cold
  ch_bucket: optikk-ch-cold
  hmac_service_account: optikk-storage@my-gcp-project.iam.gserviceaccount.com
admin:
  email: admin@optikk.dev
  password: Password123!
```

State the CLI writes: the admin session cache at `~/.optikk/token.json` (per API base).

---

## Resource sizing

Pod resource **requests** live in `deploy/` (base + `overlays/gcp` patches) and are applied
unchanged. The CLI owns **cluster/VM** sizing and validates the host floor before `up`.

### Local — kind on Podman
Cluster total for 1 tenant ≈ **1.5 vCPU / ~3 GiB / ~12 GiB disk**. Podman machine floor the CLI
enforces: **≥5 vCPU, ≥8 GiB RAM (hard floor — less and ClickHouse OOMs), ≥40 GiB disk**, rootful
and running. Short? `up` fails with the exact `podman machine set --cpus/--memory/--disk-size`
command (or fixes it with `--manage-podman`).

### Cloud — GKE
Node pool `e2-standard-4` (4 vCPU / 16 GiB), autoscaling **3 → 6** nodes (override via
`--machine-type/--nodes/--min-nodes/--max-nodes`). Stateful requests from `overlays/gcp`:
ClickHouse 2 vCPU / 8 GiB + 100 Gi SSD + GCS cold tier · MariaDB 1 vCPU / 4 GiB + 20 Gi ·
mq 1 vCPU / 2 GiB (data in GCS). Stateless workloads scale via HPA.

---

## Project layout

```
pro/optikk/
  main.go                 cobra Execute()
  assets/                 embedded deploy/ kustomize tree (go:embed) + Materialize()
  cmd/                    one file per command; root.go wires target + persistent flags
    up down status verify tenant admin team config version   (+ gcpflags, root)
  internal/
    config/               viper: flags -> optikk.yaml/~/.optikk -> gcloud fallback
    deploypath/           resolve manifests (embedded assets/deploy, or --deploy-dir override)
    hostexec/             podman machine precheck (+ opt-in manage), pids-limit
    localcluster/         kind create/delete/load via sigs.k8s.io/kind/pkg/cluster
    k8sapply/             kustomize render -> client-go SSA (CRD-ordered) + rollout waits,
                          metrics-server, namespace/secret helpers
    gcp/                  GKE (container/apiv1) + GCS buckets/HMAC (cloud.google.com/go/storage)
    provision/            Local + GCP "up/down" orchestrators
    target/               resolve live conn (REST config + API/OTLP base) per target
    verify/               /health + OTLP roundtrip + ClickHouse count assert (pod exec)
    status/               pod readiness table
    tenant/               onboard/offboard per-tenant otel-collector
    apiclient/            query API client: login (JWT cache) + CreateTeam/CreateUser
    adminseed/            patch query admin secret + restart to reseed
```

---

## Extending the CLI

- **New command:** add `cmd/<name>.go` exposing `func newXCmd(app *App) *cobra.Command` and add
  one line to the `root.AddCommand(...)` list. Shared state (config, deploy dir) hangs off the
  injected `*App` — no globals, no edits to existing commands.
- **New target** (e.g. EKS): implement the `up`/`down` orchestrator in `internal/provision` and
  resolve its connection in `internal/target`. `k8sapply`, `verify`, and rollout-wait are shared,
  so a new target reuses the apply/verify path unchanged.

---

## Troubleshooting

| Symptom | Cause / fix |
|---|---|
| `up` refuses at precheck | Podman machine below floor or not rootful/running — run the printed `podman machine set ...` (or use `--manage-podman`). |
| Offline / `unknown containerd config version` on manual `kind load` | Images are public so a normal `up` pulls them; for air-gapped runs use `--load-local-images` (loads via the kind library, and the 4 app images must exist on the host — `podman images`). |
| `verify` span count stays 0 | No team matches the tenant key on a fresh cluster — `team create` then `tenant onboard --key`. If a key was tried too early, `kubectl -n optikk rollout restart deploy/ingest` clears the 5-min negative cache. |
| `login` 405 | The query API is under `/api` (Traefik `PathPrefix(/api)`); the client already targets `<base>/api`. |
| `team`/`admin` commands say no session | Run `optikk admin login` first (caches JWT at `~/.optikk/token.json`). |

---

## Status

| Milestone | State |
|---|---|
| M0 scaffold + git repo + config | ✅ verified |
| M1 `up`/`down --target local` | ✅ verified (full destroy + from-scratch rebuild) |
| M2 `status` + `verify` | ✅ verified (0→1 span roundtrip) |
| M4 `tenant onboard/offboard` | ✅ verified (keyed trace lands, cleanup works) |
| M5 `admin` + `team` | ✅ verified (login, team create, member add, reseed) |
| M3 `up`/`down --target gcp` | ⚠️ code complete, compile/vet clean — **live GKE run not yet performed** (billable) |

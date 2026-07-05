# optikk

One CLI to **provision** the full Optikk observability stack on a local kind cluster **and
query** it (Datadog Pup-style). Manifests are `go:embed`-ed in the ~8 MB binary; it shells out to
`podman`, `kind`, `kubectl` for provisioning and hits the query API directly for data commands.

- **Module:** `github.com/optikklabs/optikk` · **Go 1.26** · **Cobra**
- Env is a flag (`--target local`, the default), not a subcommand.
- Three command families: **ops** (`init`/`down`/`status`/`verify`/`tenant`/`admin`/`team`) provision
  infra; **account** (`signup`/`login`/`onboard`/`keys`) self-serve onboarding against a running API;
  **data** (`traces`/`logs`/`metrics`/`services`/`infra`/`llm`/`saturation`/`dashboards`/`monitors`)
  read the query API — no `kubectl` needed, TTY-aware (tables for humans, JSON for pipes).

## Install

```bash
go install github.com/optikklabs/optikk@latest      # Go 1.26+
export PATH="$(go env GOPATH)/bin:$PATH"
# or: brew install optikklabs/tap/optikk  |  curl a release tarball  |  make build
```
Provisioning needs `podman` (rootful, ≥5 vCPU / ≥8 GiB RAM / ≥40 GiB disk — under 8 GiB ClickHouse
OOMs), `kind`, `kubectl` on PATH. `optikk doctor` checks them; `init` fails fast with install hints.

## Quick start (self-serve onboarding)

Against a running API (local or hosted via `--api-url`), go from zero to your first trace
without `kubectl`:

```bash
optikk onboard                             # signup (or reuse session) -> wait for collector -> print OTLP snippet -> wait for first trace
# or step by step:
optikk signup                              # create account + team, print ingest api_key, cache JWT
optikk login                               # browser device login instead of a password (RFC 8628)
optikk demo send                           # push synthetic traces (no service needed) -> see data in seconds
# then point your OpenTelemetry SDK at the printed endpoint + x-api-key, and:
optikk verify --remote                     # poll onboarding status (collector ready? first trace landed?) — no cluster access
optikk keys rotate                         # issue a fresh ingest key   |   optikk keys revoke --yes  to disable ingest
```

## Quick start (local)

```bash
optikk doctor                              # verify podman/kind/kubectl
optikk init                                # provision kind cluster + full stack -> API at :8080
optikk verify                              # /health + OTLP roundtrip + ClickHouse count assert
optikk admin setup && optikk admin login   # seed + cache super-admin (admin@optikk.dev / Password123!)
optikk team create demo                    # prints team_id, slug, api_key K  (K == tenant key)
optikk tenant onboard demo --key K         # per-tenant otel-collector
optikk verify --api-key K                  # keyed trace lands
optikk status                              # pod readiness   |   optikk down  to tear down
```
First-run order matters: a fresh cluster has **no** team matching the collector key, and ingest
**negative-caches a failed key for ~5 min** — so create the team *before* first verifying its key.
If a key was tried too early: `kubectl -n optikk rollout restart deploy/ingest`.

## Global flags

Ops: `--target local` (default) · `--config PATH` · `--deploy-dir PATH` · `-v/--verbose`.
Data: `--api-url`/`OPTIKK_API_URL` (default `http://localhost:8080`) · `--team-id`/`OPTIKK_TEAM_ID`
(`X-Team-Id` header) · `-o/--output` `table|json|yaml` · `--agent`/`FORCE_AGENT_MODE=1` (JSON, no prompts).
Most data commands take `--from 1h --to now` (accepts `15m`/`7d`/ISO8601/epoch-ms) and `-q/--query`
(Datadog-style DSL, e.g. `service:api status:error has_error:true`).

## Ops commands

| Command | Does |
|---|---|
| `doctor` | Check `podman`/`kind`/`kubectl` are installed. Aliases: `check`, `preflight`. |
| `init` | Provision cluster + deploy full stack. `[--timeout 10m]`. Aliases: `start`, `deploy`, `up`. |
| `down` | Tear down stack + cluster. `[--keep-cluster]`. Aliases: `stop`, `destroy`. |
| `status` | List `optikk`-namespace pods + readiness. Aliases: `ps`, `pods`. |
| `verify` | `/health` → OTLP trace with `x-api-key` → assert ClickHouse span count rose (polls ~30s). `[--api-key c3448fae] [--trace-file F]`. `--remote` verifies via the query API only (onboarding status; no cluster access). |
| `tenant onboard <slug> --key K` / `offboard <slug>` | Render + apply/delete the per-tenant otel-collector. |
| `admin setup` / `admin login` | `setup` reseeds super-admin (create-if-absent); `login` caches JWT at `~/.optikk/token.json`. `[--email E] [--password P]`. |
| `team create <name>` / `team member add <email>` | `create` (admin-gated) prints `team_id`/`slug`/`api_key`; `member add --team ID --password P` maps to create-user. |
| `config show` · `config get-contexts` · `config set-context <name> [--api-url U] [--team-id N]` · `config use-context <name>` | Print merged config; list/add/switch named contexts (kubectl/gh-style) for multiple stacks. Aliases: `cfg`, `contexts`. |
| `completion` · `version` | Gen bash/zsh/fish/powershell completion; print version/commit/date. |

## Account & onboarding commands

Self-serve against a running API (no `kubectl`). All accept `--api-url`/`OPTIKK_API_URL`.

| Command | Does |
|---|---|
| `signup` | `POST /api/v1/auth/signup`: create account + team, print the ingest `api_key` + OTLP snippet, cache the JWT at `~/.optikk/token.json`. `[--email E] [--password P] [--name N] [--org O]` (prompted if omitted). |
| `login` | Browser device-authorization login (RFC 8628): prints a code, opens the approval page, polls until you confirm — no password paste. |
| `onboard` | One shot: signup (or reuse cached session) → wait for your collector → print the `OTEL_EXPORTER_OTLP_*` snippet → wait until your first trace lands. `[--first-data-timeout 10m]` + the signup flags. |
| `demo send` | Push synthetic OTLP traces (fresh timestamps, unique IDs) with your ingest key — no cluster access, no instrumented service. `[--api-key K] [--count N]`. Then `optikk traces search`. |
| `keys rotate` / `keys revoke --yes` | `rotate` issues a fresh ingest key (old key works ~5m until the ingest cache expires); `revoke` disables ingest for the team. Alias: `key`. |

`signup`/`onboard` derive the OTLP endpoint from the API URL (hosted: `api.<domain>` → `ingest.<domain>:4318`); set `OTEL_EXPORTER_OTLP_ENDPOINT` to override the printed value for a non-standard deployment.

## Data commands

All read the query API. Row-list verbs render tables; analytical/graph/tree verbs emit raw JSON.

| Group | Subcommands |
|---|---|
| `auth` | `login [--email E] [--password P]` · `status` · `logout` |
| `traces` | `search -q Q` · `get <id>` · `trend` · `critical-path <id>` · `error-path <id>` · `service-map <id>` · `errors <id>` · `related <id> --service S --operation O` |
| `logs` | `search -q Q` · `facets -q Q` · `summary -q Q` · `trend -q Q` |
| `metrics` | `list` · `query --metric M --aggregation A` · `tags <metric>` |
| `services` | `list` (fleet RED) · `topology` (dependency graph) · `summary <service>` · `top-endpoints` · `errors` (error groups) |
| `infra` | `hosts` · `nodes` · `pods` · `cpu` (per-instance) · `memory` (per-instance) |
| `llm` | `apps` · `cost --group-by model` · `timeseries --metric spend\|latency\|tokens_by_vendor` · `traces [--limit N]` · `trace <id>` |
| `saturation` | `db-systems` · `db-latency` · `db-slow-queries` · `db-ops` · `kafka-topology` · `kafka-throughput` · `kafka-groups` |
| `dashboards` | `list` · `get <id>` · `create` · `update <id>` · `delete <id>` · `export <id> -o F` · `import -f F` · `url <id>` |
| `monitors` | `list [--status triggered]` · `get <id>` · `create -f F` · `update <id> -f F` · `delete <id>` · `mute <id> --duration 1h` · `unmute <id>` · `ack <id>` · `test <id>` |
| `agent` | `schema` — JSON command-tree for AI discoverability. |

## Architecture

```
Browser ─/ , /api─┐                        ┌─ web (dashboard UI)
OTLP client ──────┤  traefik (ingress) ────┼─ query (API + JWT auth) ─┬─ clickhouse (optikk.spans)
   x-api-key ─────┘        │                └─ otel-collector-<tenant> │
   :4318 OTLP ────────────>│                       │                  └─ mariadb (teams + identity)
                           └─ ingest <── collector ─┘ ── validate key (mariadb) ── mq ── clickhouse
```
`init` deploys the whole stack (no optional components): traefik, web, query, ingest, one
otel-collector per tenant, mq, clickhouse, mariadb, metrics-server (kind, `--kubelet-insecure-tls`).
App images are public `ghcr.io/optikklabs/{ingest,query,web}` + `ghcr.io/ramantayal12/mq`; upstream
images: `clickhouse:26.6`, `mariadb:11.4`, `otel-collector-contrib:0.155.0`, `traefik:v3.3`.
The app images are multi-arch (`linux/amd64` + `linux/arm64`); kind pulls them straight from ghcr and
selects the node's architecture, so Apple Silicon runs arm64 natively.

## Layout & extending

```
main.go            cobra Execute()
assets/            embedded deploy/ kustomize tree (go:embed)
cmd/               one file per command; root.go wires flags + AddCommand
internal/
  config deploypath prereq hostexec localcluster kubectl k8sapply provision target
  verify status tenant apiclient adminseed          (ops)
  queryclient/     typed query-API client (one file per domain)
  clitime dsl output                                (data: range parsing, DSL, table/JSON writer)
```
**New command:** add `cmd/<name>.go` with `func newXCmd(app *App) *cobra.Command` and one line in
`root.AddCommand`. **New data domain:** add `internal/queryclient/<domain>.go` (typed methods over
`do`) + `cmd/<domain>.go`. Shared state hangs off the injected `*App` — no globals.

## Troubleshooting

| Symptom | Fix |
|---|---|
| `init` refuses at precheck | Podman below floor / not rootful — run the printed `podman machine set …`. |
| `missing required tools` | Install printed podman/kind/kubectl (error has exact brew cmd + docs). |
| `verify` span count stays 0 | No team matches the key — `team create` then `tenant onboard --key`; if tried early, `rollout restart deploy/ingest`. |
| data cmd 405 / no session | API is under `/api` (client already targets `<base>/api`); run `optikk auth login` (or `admin login`) first. |

## Status

`init`/`down`, `status`/`verify`, `tenant onboard/offboard`, `admin`+`team`, and the full data-command
suite are verified against a live local stack. GCP/cloud targets are out of scope (local-only).

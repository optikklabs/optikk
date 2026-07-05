# optikk

One CLI to **provision** the full Optikk observability stack on a local kind cluster **and
query** it. Manifests are `go:embed`-ed in the ~8 MB binary; it shells out to `podman`, `kind`,
`kubectl` for provisioning and hits the query API directly for data commands.

- **Module:** `github.com/optikklabs/optikk` · **Go 1.26** · **Cobra**
- Env is a flag (`--target local`, the default), not a subcommand.
- Three command families: **ops** (`init`/`down`/`status`/`verify`/`tenant`/`admin`) provision infra;
  **account** (`signup`/`login`/`onboard`/`keys`) self-serve onboarding against a running API;
  **data** (`traces`/`logs`/`metrics`/`services`/`infra`/`llm`/`saturation`/`dashboards`/`monitors`)
  read the query API — no `kubectl` needed, TTY-aware (tables for humans, JSON for pipes).
- **Local is the default** (the demo above). The account + data commands also run against the
  team-hosted Optikk under **`optikk cloud …`** — same commands, pointed at `api.optikk.in`
  (OTLP at `ingest.optikk.in`). See [Managed (`optikk cloud`)](#managed-optikk-cloud).

## Install

```bash
go install github.com/optikklabs/optikk@latest      # Go 1.26+
export PATH="$(go env GOPATH)/bin:$PATH"
# or grab a release binary from GitHub Releases, or `make build`
```

Provisioning needs `podman` (rootful, ≥5 vCPU / ≥8 GiB RAM / ≥40 GiB disk — under 8 GiB ClickHouse
OOMs), `kind`, and `kubectl` on PATH. `optikk doctor` checks them; `init` fails fast with install hints.

## Quick start

```bash
optikk doctor                              # verify podman/kind/kubectl
optikk init                                # provision kind cluster + full stack
optikk signup                              # create account + tenant, print ingest api_key, cache JWT
optikk verify                              # /health + OTLP roundtrip + ClickHouse count assert
optikk status                              # pod readiness   |   optikk down  to tear down
```

Then open the web UI at **http://localhost:18040** and log in with the account you just made.
Point any OpenTelemetry SDK at the endpoint + key that `signup` prints:

```bash
export OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318
export OTEL_EXPORTER_OTLP_HEADERS=x-api-key=<your ingest key>
optikk demo send                           # or push synthetic traces without a service
```

No account yet on a fresh cluster? `optikk admin setup` seeds the default super-admin
(`admin@optikk.dev` / `Password123!`); `optikk auth login` caches its session at `~/.optikk/config.json`.

## Managed (`optikk cloud`)

The same account + data commands run against the **team-hosted** Optikk — no local stack to
provision. Everything under `optikk cloud` defaults to `https://api.optikk.in` (OTLP at
`ingest.optikk.in:4318`); `--api-url` overrides it for staging.

```bash
optikk cloud signup                        # create a hosted account + tenant, print ingest key
optikk cloud demo send                     # push synthetic telemetry to ingest.optikk.in
optikk cloud traces search -q "service:api"
```

Provisioning commands (`init`/`down`/`status`/`verify`/`tenant`/`admin`) are **local-only** and
are not part of the `cloud` subtree — you can't provision the hosted stack from the CLI.

## Global flags

Ops: `--target local` (default) · `--config PATH` · `--deploy-dir PATH` · `-v/--verbose`.
Data: `--api-url`/`OPTIKK_API_URL` (default `http://localhost:18040`) · `--tenant-id`/`OPTIKK_TENANT_ID`
(`X-Tenant-Id` header) · `-o/--output` `table|json|yaml` · `--agent`/`FORCE_AGENT_MODE=1` (JSON, no prompts).
Most data commands take `--from 1h --to now` (accepts `15m`/`7d`/ISO8601/epoch-ms) and `-q/--query`
(Datadog-style DSL, e.g. `service:api status:error has_error:true`).

## Ops commands

| Command | Does |
|---|---|
| `doctor` | Check `podman`/`kind`/`kubectl` are installed. Aliases: `check`, `preflight`. |
| `init` | Provision cluster + deploy full stack. `[--timeout 10m]`. Aliases: `start`, `deploy`, `up`. |
| `down` | Tear down stack + cluster. `[--keep-cluster]`. Aliases: `stop`, `destroy`. |
| `status` | List `optikk`-namespace pods + readiness. Aliases: `ps`, `pods`. |
| `verify` | `/health` → OTLP trace with `x-api-key` → assert ClickHouse span count rose (polls ~30s). `[--api-key K] [--trace-file F]`. `--remote` verifies via the query API only (no cluster access). |
| `tenant onboard <id> --key K` / `offboard <id>` / `member add <email> --tenant ID` | Render + apply/delete a per-tenant otel-collector; map a user into a tenant. |
| `admin setup` | Reseed the query super-admin (create-if-absent). `[--email E] [--password P]`. Alias: `seed`. |
| `config show` · `config get-contexts` · `config set-context <name> [--api-url U] [--tenant-id N]` · `config use-context <name>` | Print merged config; list/add/switch named contexts (kubectl-style) for multiple stacks. Aliases: `cfg`, `contexts`. |
| `completion` · `version` | Gen bash/zsh/fish/powershell completion; print version/commit/date. |

## Account & onboarding commands

Self-serve against a running API (no `kubectl`). All accept `--api-url`/`OPTIKK_API_URL`.

| Command | Does |
|---|---|
| `signup` | `POST /api/v1/auth/signup`: create account + tenant, print the ingest `api_key` + OTLP snippet, cache the JWT at `~/.optikk/config.json`. `[--email E] [--password P] [--name N] [--org O]` (prompted if omitted). |
| `login` | Browser device-authorization login (RFC 8628): prints a code, opens the approval page, polls until you confirm — no password paste. |
| `auth login` / `auth status` / `auth logout` | Password login, show current session, or clear it (`config.json`). |
| `onboard` | One shot: signup (or reuse cached session) → wait for your collector → print the `OTEL_EXPORTER_OTLP_*` snippet → wait until your first trace lands. `[--first-data-timeout 10m]` + the signup flags. |
| `demo send` | Push synthetic OTLP traces (fresh timestamps, unique IDs) with your ingest key — no service needed. `[--api-key K] [--count N]`. |
| `keys rotate` / `keys revoke --yes` | `rotate` issues a fresh ingest key (old key works ~5m until the ingest cache expires); `revoke` disables ingest for the tenant. Alias: `key`. |

`signup`/`onboard` derive the OTLP endpoint from the API URL; set `OTEL_EXPORTER_OTLP_ENDPOINT` to
override the printed value for a non-standard deployment.

## Data commands

All read the query API. Row-list verbs render tables; analytical/graph/tree verbs emit raw JSON.

| Group | Subcommands |
|---|---|
| `traces` | `search -q Q` · `get <id>` · `trend` · `critical-path <id>` · `error-path <id>` · `service-map <id>` · `errors <id>` · `related <id> --service S --operation O` |
| `logs` | `search -q Q` · `facets -q Q` · `summary -q Q` · `trend -q Q` |
| `metrics` | `list` · `query --metric M --aggregation A` · `tags <metric>` |
| `services` | `list` (fleet RED) · `topology` · `summary <service>` · `top-endpoints` · `errors` |
| `infra` | `hosts` · `nodes` · `pods` · `cpu` · `memory` (per-instance) |
| `llm` | `apps` · `cost --group-by model` · `timeseries --metric spend\|latency\|tokens_by_vendor` · `traces` · `trace <id>` |
| `saturation` | `db-systems` · `db-latency` · `db-slow-queries` · `db-ops` · `kafka-topology` · `kafka-throughput` · `kafka-groups` |
| `dashboards` | `list` · `get <id>` · `create` · `update <id>` · `delete <id>` · `export <id> -o F` · `import -f F` · `url <id>` |
| `monitors` | `list [--status triggered]` · `get <id>` · `create -f F` · `update <id> -f F` · `delete <id>` · `mute <id> --duration 1h` · `unmute <id>` · `ack <id>` · `test <id>` |
| `agent` | `schema` — JSON command-tree for AI discoverability. |


**New command:** add `cmd/<name>.go` with `func newXCmd(app *App) *cobra.Command` and one line in
`root.AddCommand`. **New data domain:** add `internal/queryclient/<domain>.go` + `cmd/<domain>.go`.
Shared state hangs off the injected `*App` — no globals.

## Troubleshooting

| Symptom | Fix |
|---|---|
| `init` refuses at precheck | Podman below floor / not rootful — run the printed `podman machine set …`. |
| `missing required tools` | Install printed podman/kind/kubectl (error has the exact command). |
| `verify` span count stays 0 | No tenant matches the key — `signup` (or `tenant onboard --key`); if a key was tried too early, ingest negative-caches it ~5 min: `kubectl -n optikk rollout restart deploy/ingest`. |
| data cmd 405 / no session | API is under `/api` (the client targets `<base>/api`); run `optikk auth login` (or `optikk login`) first. |
| `onboard` reuses a dead session | `optikk auth logout` clears the cached token in `~/.optikk/config.json`, then `signup`/`login` again. |

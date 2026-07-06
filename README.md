# optikk

The `optikk` CLI is a pure query and API-client tool for the Optikk observability stack. It hits the query API directly for data commands and handles onboarding/authentication.

- **Module:** `github.com/optikklabs/optikk` · **Go 1.26** · **Cobra**
- Two command families:
  - **account** (`signup`/`login`/`onboard`/`keys`/`config`) self-serve onboarding against a running API and manage contexts.
  - **data** (`traces`/`logs`/`metrics`/`services`/`infra`/`llm`/`saturation`/`dashboards`/`monitors`) read the query API — no `kubectl` needed, TTY-aware (tables for humans, JSON for pipes).

## Install

```bash
go install github.com/optikklabs/optikk@latest      # Go 1.26+
export PATH="$(go env GOPATH)/bin:$PATH"
# or grab a release binary from GitHub Releases, or `make build`
```

## Quick start

```bash
optikk config init                         # configure your API endpoint
optikk signup                              # create account + tenant, print ingest api_key, cache JWT
```

Then open the web UI at your configured API endpoint and log in with the account you just made.
Point any OpenTelemetry SDK at the endpoint + key that `signup` prints:

```bash
export OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318
export OTEL_EXPORTER_OTLP_HEADERS=x-api-key=<your ingest key>
```

## Global flags

Data commands: `--api-url`/`OPTIKK_API_URL` (default `http://localhost:18040`) · `--tenant-id`/`OPTIKK_TENANT_ID`
(`X-Tenant-Id` header) · `-o/--output` `table|json|yaml` · `--agent`/`FORCE_AGENT_MODE=1` (JSON, no prompts).
Most data commands take `--from 1h --to now` (accepts `15m`/`7d`/ISO8601/epoch-ms) and `-q/--query`
(Datadog-style DSL, e.g. `service:api status:error has_error:true`).

## Config commands

| Command | Does |
|---|---|
| `config init` | Initialize a new context interactively. |
| `config show` · `config get-contexts` · `config set-context <name> [--api-url U] [--tenant-id N]` · `config use-context <name>` | Print merged config; list/add/switch named contexts for multiple environments. Aliases: `cfg`, `contexts`. |
| `completion` · `version` | Gen bash/zsh/fish/powershell completion; print version/commit/date. |

## Account & onboarding commands

Self-serve against a running API. All accept `--api-url`/`OPTIKK_API_URL`.

| Command | Does |
|---|---|
| `signup` | `POST /api/v1/auth/signup`: create account + tenant, print the ingest `api_key` + OTLP snippet, cache the JWT at `~/.optikk/config.json`. `[--email E] [--password P] [--name N] [--org O]` (prompted if omitted). |
| `login` | Browser device-authorization login (RFC 8628): prints a code, opens the approval page, polls until you confirm — no password paste. |
| `auth login` / `auth status` / `auth logout` | Password login, show current session, or clear it (`config.json`). |
| `onboard` | Prints your API key and the `OTEL_EXPORTER_OTLP_*` snippet. |
| `keys rotate` / `keys revoke --yes` | `rotate` issues a fresh ingest key (old key works ~5m until the ingest cache expires); `revoke` disables ingest for the tenant. Alias: `key`. |
| `users add` | Add a user to a tenant. Alias: `user`, `member`. |

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

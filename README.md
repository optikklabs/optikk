# optikk

The `optikk` CLI is a query and API-client tool for the Optikk observability stack. It hits the query API directly for data commands and handles onboarding/authentication.

## Install

**Homebrew (macOS/Linux):**
```bash
brew tap optikklabs/tap
brew trust optikklabs/tap
brew install optikk
```

**Go (cross-platform):**
```bash
go install github.com/optikklabs/optikk@latest
```
*(Or grab a release binary from GitHub Releases)*

## Quick Start

```bash
optikk config init
optikk signup
```
Log into the web UI using the created account. Point your OpenTelemetry SDK at the endpoint and use the printed `api_key` for `OTEL_EXPORTER_OTLP_HEADERS=x-api-key=<key>`.

## Global Flags

- `--api-url` / `OPTIKK_API_URL` (default: `http://localhost:18040`)
- `--tenant-id` / `OPTIKK_TENANT_ID`
- `-o/--output` (format: `table|json|yaml`)
- `--from` / `--to` (e.g., `15m`, `1h`, `7d`, `now`)
- `-q/--query` (e.g., `service:api status:error`)

## Core Commands

### Account & Config
- `optikk config init` - Initialize a new context interactively.
- `optikk config show` - Print merged config.
- `optikk signup` - Create account & tenant, print ingest api_key.
- `optikk login` - Browser device-authorization login.
- `optikk auth login|status|logout` - Password login / check session / logout.
- `optikk keys rotate|revoke` - Rotate or revoke your ingest API key.

### Data & Query
- **Traces:** `optikk traces search -q <query>` · `get <id>` · `trend` · `critical-path` · `service-map`
- **Logs:** `optikk logs search -q <query>` · `facets` · `summary` · `trend`
- **Metrics:** `optikk metrics list` · `query` · `tags`
- **Services:** `optikk services list` · `topology` · `summary` · `top-endpoints` · `errors`
- **Infrastructure:** `optikk infra hosts|nodes|pods|cpu|memory`
- **LLM/AI:** `optikk llm apps|cost|timeseries|traces`
- **Saturation:** `optikk saturation db-systems|db-latency|kafka-topology|kafka-throughput`
- **Dashboards:** `optikk dashboards list` · `get` · `create` · `update` · `delete`
- **Monitors:** `optikk monitors list` · `get` · `create` · `update` · `delete` · `mute` · `unmute` · `ack`

# optikk

The `optikk` CLI is a query and API-client tool for the Optikk observability stack. It hits the query API directly for data commands and handles onboarding/authentication.

## Install

macOS and Linux (amd64/arm64), installs to `/usr/local/bin`:
```bash
curl -fsSL https://optikk.in/install.sh | sh
```

The script ([install.sh](install.sh)) downloads the latest [GitHub Release](https://github.com/optikklabs/optikk/releases) tarball for your platform, verifies its checksum, and — when `cosign` is installed — verifies the release signature. Use `OPTIKK_VERSION=v<x.y.z>` to pin a version and `OPTIKK_INSTALL_DIR` to change the destination — or grab a release tarball manually.

Once installed, `optikk update` keeps itself current and always verifies the release signature, using the public key compiled into the binary.

## Quick Start

```bash
optikk signup
optikk status
```
Log into the web UI (`optikk open`) using the created account. Point your OpenTelemetry SDK at the endpoint and use the printed `api_key` for `OTEL_EXPORTER_OTLP_HEADERS=x-api-key=<key>`.

Optikk is a hosted service: the CLI talks to `https://api.optikk.in` by default, and **only over HTTPS** — a plaintext `--api-url` is rejected rather than silently downgraded.

## Global Flags

- `--api-url` / `OPTIKK_API_URL` (default: `https://api.optikk.in`, https only)
- `--tenant-id` / `OPTIKK_TENANT_ID`
- `-o/--output` (format: `table|json|yaml`)
- `--from` / `--to` (e.g., `15m`, `1h`, `7d`, `now`)
- `-q/--query` (e.g., `service:api status:error`)

## Core Commands

### Account & Config
- `optikk config init` - Initialize a new context interactively.
- `optikk config show` - Print merged config.
- `optikk config set <key> <value>` / `unset <key>` - Change one field (`api_url`, `tenant_id`) on the active context.
- `optikk config current-context` / `use-context` / `get-contexts` / `delete-context` - Manage contexts.
- `optikk signup` - Create account & tenant, print ingest api_key.
- `optikk login` - Browser device-authorization login.
- `optikk auth login|status|logout` - Password login / check session / logout.
- `optikk whoami` - Show the account, tenant, and API of the current session.
- `optikk keys rotate|revoke` - Rotate or revoke your ingest API key.

### Getting Around
- `optikk status` - Is the API reachable, am I signed in, is there an update?
- `optikk update` - Update to the latest signed release (`--check` to only report; exits non-zero when an update exists).
- `optikk open [page]` - Open the web app (`traces`, `logs`, `dashboards`, …).
- `optikk docs` / `support` / `feedback` - Open documentation, support, or file an issue.

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

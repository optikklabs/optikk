# optikk

The `optikk` CLI is a query and API-client tool for the Optikk observability stack. It hits the query API directly for data commands and handles onboarding/authentication.

## Install

macOS and Linux (amd64/arm64), installs to `/usr/local/bin`:
```bash
curl -fsSL https://optikk.in/install.sh | sh
```

The script ([install.sh](install.sh)) downloads the latest [GitHub Release](https://github.com/optikklabs/optikk/releases) tarball for your platform over HTTPS and verifies its checksum. Use `OPTIKK_VERSION=v<x.y.z>` to pin a version and `OPTIKK_INSTALL_DIR` to change the destination — or grab a release tarball manually.

Once installed, `optikk update` keeps itself current.

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
- `optikk signup` / `onboard` - Create account & tenant, print ingest api_key + OTLP env pair.
- `optikk verify` - Confirm telemetry is arriving (exits non-zero when the window is empty).
- `optikk users add|list` - Invite teammates (no `--password` = set-password email) and list members.
- `optikk login` - Browser device-authorization login.
- `optikk auth login|status|logout` - Password login / check session / logout.
- `optikk whoami` - Show the account, tenant, and API of the current session.
- `optikk keys rotate|revoke` - Rotate or revoke your ingest API key.

### Getting Around
- `optikk status` - Is the API reachable, am I signed in, is there an update?
- `optikk update` - Update to the latest release (`--check` to only report; exits non-zero when an update exists).
- `optikk open [page]` - Open the web app (`traces`, `logs`, `dashboards`, …).
- `optikk docs` / `support` / `feedback` - Open documentation, support, or file an issue.

### Data & Query
- **Traces:** `optikk traces search -q <query>` · `get <id>` · `trend` · `critical-path` · `service-map`
- **Logs:** `optikk logs search -q <query>` · `trace <trace-id>` · `facets` · `summary` · `trend`
- **Errors:** `optikk errors list` · `get <group>` · `traces <group>` · `timeseries <group>` · `latest <group>`
- **Metrics:** `optikk metrics list` · `query` · `tags`
- **Services:** `optikk services list` · `topology` · `summary` · `top-endpoints` · `errors`
- **Infrastructure:** `optikk infra hosts|nodes|pods|cpu|memory`
- **LLM/AI:** `optikk llm apps|cost|timeseries|traces`
- **Saturation:** `optikk saturation db-systems|db-latency|kafka-topology|kafka-throughput`
- **Dashboards:** `optikk dashboards list` · `get` · `create` · `update` · `delete`
- **Monitors:** `optikk monitors list` · `get` · `create` · `update` · `delete` · `mute` · `unmute` · `ack`

## AI Agents

The CLI is built to be driven end-to-end by AI coding agents (Claude Code, Codex, …):

- `optikk agent setup` installs the agent guide into your project as a Claude Code skill (`.claude/skills/optikk/SKILL.md`); `--agents-md` maintains a marked section in `AGENTS.md`; `--print` dumps it. The same guide lives at [AGENTS.md](AGENTS.md) and the discovery entry point is [llms.txt](llms.txt).
- `--agent` (or `OPTIKK_AGENT=1`): JSON on stdout, a one-line JSON error envelope on stderr, no prompts.
- Exit codes: `0` ok · `1` error · `2` usage · `3` auth · `4` network unreachable · `5` API error.
- `optikk agent schema` emits the full command tree, exit codes, and example playbooks as JSON.

**Breaking changes vs. earlier builds:** `--agent` no longer auto-confirms `dashboards/monitors delete` or `update` — pass `--yes`; and `signup`/`onboard` no longer prompt on a non-TTY stdin (pipe-fed answers) — pass all flags instead.

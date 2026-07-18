# CODE_INDEX.md — optikk CLI

Index of the `optikk` CLI (`github.com/optikklabs/optikk`, Go 1.26, Cobra). The `optikk` CLI is a pure query and API-client tool. Update this file after any structural change.

## Entry point

- `main.go` — `os.Exit(cmd.Execute())`. All error rendering and exit-code mapping lives in `cmd/execute.go`: prose + hint for humans, a one-line JSON envelope on stderr in agent mode, exit codes from the `internal/clierr` taxonomy (1 internal, 2 usage, 3 auth, 4 network, 5 api). `cmd.SilentExitError` still exits silently with its code (used by `update --check`).

## Agent mode

`--agent` (or `OPTIKK_AGENT=1`) makes every command machine-consumable: JSON documents on stdout (lifecycle commands render typed result structs through `writeResult` in `data_helpers.go`), the stderr error envelope above, and no prompts — non-interactive invocations fail with a usage error naming the missing flags instead of blocking. Agent mode never auto-confirms destructive commands; `--yes` is always required for `dashboards/monitors delete`, `keys revoke`, and `update`.

Agent-facing assets: `optikk agent schema` (command tree + exit codes + example playbooks), `optikk agent setup` (writes `.claude/skills/optikk/SKILL.md`, `--agents-md` maintains a marked AGENTS.md block, `--print` dumps the guide), repo-root `AGENTS.md` (generated — `make gen`; a test fails if stale) and `llms.txt` (served via optikk.in redirect, like install.sh). Single source: `internal/agentdocs`.

## `cmd/` — one file per command, wired onto a shared `*App` in `root.go`

`App` (in `root.go`) carries resolved config + agent-mode; commands depend on it, not globals.

**Root / wiring**
- `root.go` — `NewRootCmd`, persistent flags, `App.load` (config+env+flags → `App.Cfg`). Resolves the API base **once** per invocation; commands read it via `App.API()`, which returns the reason it is unusable rather than failing in `load` (so `config` can repair a bad context).
- `config.go` — context management (`optikk config init/show/get-contexts/current-context/set-context/use-context/set/unset/delete-context`).

**Account / onboarding** — `signup.go` (also owns the shared non-interactive guard: `allowPrompt`/`stdinIsTerminal`/`signupInput.missingFlags`), `login.go` (device flow; agent mode emits NDJSON approval handoff instead of opening a browser), `auth.go` (password login / status / logout), `onboard.go` (full signup flags incl. `--accept-terms`; no server call to re-read a key — cached sessions are pointed at `keys rotate`), `keys.go`, `users.go` (`add` = invite when `--password` omitted, `list`), `verify.go` (ingestion-summary "is data flowing" check; exit 1 + hint when the window is empty).

> `login`'s device flow spans two hosts: the API serves `/auth/device/code|token|approve` as POST JSON, but the **approval page is a web app route** (`AppURL/device?user_code=`) — see `deviceVerifyURL`. Anything a browser opens comes from `endpoint.AppURL`; only API calls use the resolved API base.

**Self-service / status** — `update.go` (verified self-update), `status.go` (API reachability + session + update check), `whoami.go` (session identity; `printSession` is shared with `auth status`), `open.go` (web app), `links.go` (docs/support/feedback, declared as data).

**Data (read query API, TTY-aware)** — `traces.go`, `logs.go` (incl. `logs trace <id>`), `errors.go` (error groups: list/get/traces/timeseries/latest), `metrics.go`, `services.go`, `infra.go`, `llm.go`, `saturation.go`, `dashboards.go`, `monitors.go`, `schema.go` (`agent schema` v1.1) + `agent_setup.go` (`agent setup`), plus `data_helpers.go` (client/output/range resolution, `writeResult`, `writeNDJSON`).

**Tests** — `signup_test.go`.

## `internal/` — packages (single responsibility)

- `config` — load/merge config file + `OPTIKK_*` env + flags.
- `endpoint` — the hosted service URLs (`APIURL`/`AppURL`/`SiteURL`/`DocsURL`) and `Resolve` (flag/env → context → default). **HTTPS-only: plaintext URLs are rejected, not downgraded.** `DocsURL` is a separate host — `optikk.in/docs` does not exist.
- `httpx` — the one place transport security is stated: TLS 1.2+, system roots, verification never disabled. Used by every outbound call.
- `selfupdate` — resolves, verifies, and installs releases. `release.go` (GitHub API + semver), `verify.go` (checksum over archive; fails closed), `install.go` (extract + atomic same-dir rename). **Trust model:** TLS authenticates the download (assets are GitHub's); the checksum only catches corruption, since the manifest shares the archive's origin. Releases are deliberately unsigned — see the `release.go` package doc for the reasoning and for where signing would slot in.
- `browser` — opens URLs (best-effort, non-fatal).
- `apiclient` — auth/signup/device/onboarding client + token store (`~/.optikk/config.json` contexts; `SaveToken`/`LoadToken`/`ClearToken`, plus `SetContextValue`/`UnsetContextValue`/`DeleteContext`/`CurrentContextName`). `Ping` hits the query service's root `/health` (outside the `/api` surface). `SignupRequest` carries `accepted_terms` (server requires it); `signup.go` confirms Terms/Privacy consent interactively or via `--accept-terms`.
- `queryclient` — typed query-API client, one file per domain (traces/logs/errors/metrics/services/infra/llm/saturation/dashboards/monitors/ingestion + `client.go`).
- `clierr` — leaf package: the error taxonomy (Kind → exit code), the agent-mode JSON envelope (`RenderJSON`), and the shared classifiers `FromAPI` (401/403 → Auth with re-login hint, else API with the server's `error.code`) and `Unreachable` (transport failure → Network with a config-check hint). Both API clients classify at the source, so every command inherits the taxonomy.
- `agentdocs` — the single source for agent guidance: `guide.md.tmpl` renders the skill file, the AGENTS.md section (`UpsertAgentsSection`, marker-idempotent), and the checked-in repo `AGENTS.md` (`gen/`, run via `make gen`); `Examples()`/`ExitCodes()` also feed `agent schema` so docs and schema cannot drift.
- `output` — table/json/yaml writer; `clitime` — `--from/--to` range parsing; `dsl` — query DSL.

## Releases

Tagging `v*` runs `.github/workflows/release.yml` → goreleaser. Nothing else triggers CI: pushing to `main` runs no workflow. Releases are unsigned; `optikk update` and `install.sh` both rely on HTTPS for authenticity and the checksum for integrity. Asset names are duplicated across `.goreleaser.yaml`, `install.sh`, and `selfupdate/release.go` — change them together.

`install.sh` has no staging: `optikk.in/install.sh` redirects to `raw.githubusercontent.com/.../main/install.sh`, so a push to `main` ships it to users immediately. The same applies to `llms.txt` and `AGENTS.md` once the optikk.in redirects for `/llms.txt` and `/agents.md` land (marketing repo).

## Other

- `Makefile` (`make build`, `make gen`), `scripts/` (`gen-release-key.sh` — the key ceremony), `install.sh`, `llms.txt` (agent discovery entry point), `AGENTS.md` (generated agent guide — edit `internal/agentdocs/guide.md.tmpl`, then `make gen`), `NOTICE`, `README.md`.

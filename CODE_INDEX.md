# CODE_INDEX.md — optikk CLI

Index of the `optikk` CLI (`github.com/optikklabs/optikk`, Go 1.26, Cobra). The `optikk` CLI is a pure query and API-client tool. Update this file after any structural change.

## Entry point

- `main.go` — calls `cmd.NewRootCmd().Execute()`. Handles `cmd.SilentExitError`, which sets an exit code without printing (used by `update --check`).

## `cmd/` — one file per command, wired onto a shared `*App` in `root.go`

`App` (in `root.go`) carries resolved config + agent-mode; commands depend on it, not globals.

**Root / wiring**
- `root.go` — `NewRootCmd`, persistent flags, `App.load` (config+env+flags → `App.Cfg`). Resolves the API base **once** per invocation; commands read it via `App.API()`, which returns the reason it is unusable rather than failing in `load` (so `config` can repair a bad context).
- `config.go` — context management (`optikk config init/show/get-contexts/current-context/set-context/use-context/set/unset/delete-context`).

**Account / onboarding** — `signup.go`, `login.go` (device flow), `auth.go` (password login / status / logout), `onboard.go`, `keys.go`, `users.go`, `demo.go`.

> `login`'s device flow spans two hosts: the API serves `/auth/device/code|token|approve` as POST JSON, but the **approval page is a web app route** (`AppURL/device?user_code=`) — see `deviceVerifyURL`. Anything a browser opens comes from `endpoint.AppURL`; only API calls use the resolved API base.

**Self-service / status** — `update.go` (verified self-update), `status.go` (API reachability + session + update check), `whoami.go` (session identity; `printSession` is shared with `auth status`), `open.go` (web app), `links.go` (docs/support/feedback, declared as data).

**Data (read query API, TTY-aware)** — `traces.go`, `logs.go`, `metrics.go`, `services.go`, `infra.go`, `llm.go`, `saturation.go`, `dashboards.go`, `monitors.go`, `schema.go` (`agent`), plus `data_helpers.go` (client/output/range resolution).

**Tests** — `signup_test.go`.

## `internal/` — packages (single responsibility)

- `config` — load/merge config file + `OPTIKK_*` env + flags.
- `endpoint` — the hosted service URLs (`APIURL`/`AppURL`/`SiteURL`/`DocsURL`) and `Resolve` (flag/env → context → default). **HTTPS-only: plaintext URLs are rejected, not downgraded.** `DocsURL` is a separate host — `optikk.in/docs` does not exist.
- `httpx` — the one place transport security is stated: TLS 1.2+, system roots, verification never disabled. Used by every outbound call.
- `selfupdate` — resolves, verifies, and installs releases. `release.go` (GitHub API + semver), `verify.go` (checksum over archive; fails closed), `install.go` (extract + atomic same-dir rename). **Trust model:** TLS authenticates the download (assets are GitHub's); the checksum only catches corruption, since the manifest shares the archive's origin. Releases are deliberately unsigned — see the `release.go` package doc for the reasoning and for where signing would slot in.
- `browser` — opens URLs (best-effort, non-fatal).
- `apiclient` — auth/signup/device/onboarding client + token store (`~/.optikk/config.json` contexts; `SaveToken`/`LoadToken`/`ClearToken`, plus `SetContextValue`/`UnsetContextValue`/`DeleteContext`/`CurrentContextName`). `Ping` hits the query service's root `/health` (outside the `/api` surface). `SignupRequest` carries `accepted_terms` (server requires it); `signup.go` confirms Terms/Privacy consent interactively or via `--accept-terms`.
- `queryclient` — typed query-API client, one file per domain (traces/logs/metrics/services/infra/llm/saturation/dashboards/monitors + `client.go`).
- `output` — table/json/yaml writer; `clitime` — `--from/--to` range parsing; `dsl` — query DSL.

## Releases

Tagging `v*` runs `.github/workflows/release.yml` → goreleaser. Nothing else triggers CI: pushing to `main` runs no workflow. Releases are unsigned; `optikk update` and `install.sh` both rely on HTTPS for authenticity and the checksum for integrity. Asset names are duplicated across `.goreleaser.yaml`, `install.sh`, and `selfupdate/release.go` — change them together.

`install.sh` has no staging: `optikk.in/install.sh` redirects to `raw.githubusercontent.com/.../main/install.sh`, so a push to `main` ships it to users immediately.

## Other

- `Makefile` (`make build`), `scripts/` (`gen-release-key.sh` — the key ceremony), `install.sh`, `NOTICE`, `README.md`.

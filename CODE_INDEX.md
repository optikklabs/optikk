# CODE_INDEX.md — optikk CLI

Index of the `optikk` CLI (`github.com/optikklabs/optikk`, Go 1.26, Cobra). The `optikk` CLI is a pure query and API-client tool. Update this file after any structural change.

## Entry point

- `main.go` — calls `cmd.NewRootCmd().Execute()`.

## `cmd/` — one file per command, wired onto a shared `*App` in `root.go`

`App` (in `root.go`) carries resolved config + agent-mode; commands depend on it, not globals.

**Root / wiring**
- `root.go` — `NewRootCmd`, persistent flags, `App.load` (config+env+flags → `App.Cfg`).
- `config.go` — context management (`optikk config init/show/get-contexts/set-context/use-context`).

**Account / onboarding** — `signup.go`, `login.go` (device flow), `auth.go` (password login / status / logout), `onboard.go`, `keys.go`, `users.go`, `demo.go`.

**Data (read query API, TTY-aware)** — `traces.go`, `logs.go`, `metrics.go`, `services.go`, `infra.go`, `llm.go`, `saturation.go`, `dashboards.go`, `monitors.go`, `schema.go` (`agent`), plus `data_helpers.go` (client/output/range resolution).

**Tests** — `signup_test.go`.

## `internal/` — packages (single responsibility)

- `config` — load/merge config file + `OPTIKK_*` env + flags.
- `conn` — resolve the query API base URL without kubectl.
- `apiclient` — auth/signup/device/onboarding client + token store (`~/.optikk/config.json` contexts; `SaveToken`/`LoadToken`/`ClearToken`). `SignupRequest` carries `accepted_terms` (server requires it); `signup.go` confirms Terms/Privacy consent interactively or via `--accept-terms`.
- `queryclient` — typed query-API client, one file per domain (traces/logs/metrics/services/infra/llm/saturation/dashboards/monitors + `client.go`).
- `output` — table/json/yaml writer; `clitime` — `--from/--to` range parsing; `dsl` — query DSL.

## Other

- `Makefile` (`make build`), `scripts/`, `NOTICE`, `README.md`.

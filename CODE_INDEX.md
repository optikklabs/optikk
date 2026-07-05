# CODE_INDEX.md — optikk CLI

Index of the `optikk` CLI (`github.com/optikklabs/optikk`, Go 1.26, Cobra). One binary
that **provisions** the full Optikk stack on a local kind cluster and **queries** it; the
`deploy/` kustomize tree is `go:embed`-ed. Update this file after any structural change.

## Entry point

- `main.go` — calls `cmd.NewRootCmd().Execute()`.

## `cmd/` — one file per command, wired onto a shared `*App` in `root.go`

`App` (in `root.go`) carries resolved config + agent-mode; commands depend on it, not globals.
Commands are grouped by annotation: `optikk/no-config`, `optikk/skip-deploy` (account+data,
need only an API URL/token), `optikk/managed` (the `cloud` subtree, defaults to the hosted API).

**Root / wiring**
- `root.go` — `NewRootCmd`, persistent flags, `App.load` (config+env+flags → `App.Cfg`),
  the `skipsConfig`/`skipsDeploy`/`isManaged` annotation walkers.
- `cloud.go` — `optikk cloud` parent: attaches the account+data builders against the hosted
  API (`conn.ManagedAPIURL`). No ops commands. `load` flips the default base when `isManaged`.

**Ops (local cluster; provisioning)** — `up.go` (`init`), `down.go`, `status.go`, `verify.go`,
`doctor.go`, `tenant.go`, `member.go`, `admin.go`, `config.go`, `completion.go`, `version.go`.

**Account / onboarding** — `signup.go`, `login.go` (device flow), `auth.go` (password login /
status / logout), `onboard.go`, `keys.go`, `demo.go`.

**Data (read query API, TTY-aware)** — `traces.go`, `logs.go`, `metrics.go`, `services.go`,
`infra.go`, `llm.go`, `saturation.go`, `dashboards.go`, `monitors.go`, `schema.go` (`agent`),
plus `data_helpers.go` (client/output/range resolution).

**Tests** — `cloud_test.go`, `signup_test.go`, `onboard_test.go`.

## `internal/` — packages (single responsibility)

- `config` — load/merge config file + `OPTIKK_*` env + flags.
- `conn` — resolve the query API base URL (`DefaultAPIURL` localhost:18040, `ManagedAPIURL`
  api.optikk.in) without kubectl.
- `apiclient` — auth/signup/device/onboarding client + token store (kubectl-style
  `~/.optikk/config.json` contexts; `SaveToken`/`LoadToken`/`ClearToken`).
- `queryclient` — typed query-API client, one file per domain (traces/logs/metrics/services/
  infra/llm/saturation/dashboards/monitors + `client.go`).
- `output` — table/json/yaml writer; `clitime` — `--from/--to` range parsing; `dsl` — query DSL.
- Provisioning: `deploypath`, `prereq`, `hostexec`, `localcluster`, `kubectl`, `k8sapply`,
  `provision`, `target`, `verify`, `status`, `adminseed`.

## `assets/deploy/` — embedded kustomize tree (local-only)

A single flat kustomization — no overlays, no patches, no per-tenant manifests. Tenants are
MySQL rows created at signup; `ingest` resolves them per-request from the `x-api-key` header.

- `kustomization.yaml` — root: lists every `base/` resource directly (base/ has no
  kustomization of its own). `optikk up` applies this dir.
- `base/` — raw manifests for every component (clickhouse, mariadb, mq, ingest, query, web,
  traefik) + HPAs; Traefik Service is `NodePort` inline. Traefik publishes `4317`/`4318`
  (OTLP gRPC/HTTP) and forwards `x-api-key` to `ingest` (`18317`/`18318`).
- `kind/` — kind cluster config (host port mappings).

## Other

- `Makefile` (`make build`), `scripts/`, `NOTICE`, `README.md`.

# AGENTS.md

Go client library plus `lkctl` CLI for the Lakekeeper Iceberg catalog.

## Layout

- `pkg/apis/management/v1/` — generated SDK; regenerate, never hand-edit (see Generation)
- `pkg/client/` — facade over the SDK (auth, retries, optional bootstrap)
- `pkg/core/` — auth, context, and error primitives shared across packages
- `pkg/lakekeeper/` — umbrella re-exporter; the single import path most consumers use (`lakekeeper.New`, `lakekeeper.NewS3Profile`, ...)
- `pkg/storage/`, `pkg/permissions/` — hand-written helpers / builders
- `pkg/common/`, `pkg/testutil/`, `pkg/version/` — small shared helpers (defaults, test client, build-time version)
- `cmd/` — Cobra-based CLI; entry at `cmd/main.go`, commands tree under `cmd/lkctl/commands/`
- `api/openapi/` — spec + preprocessor + postprocessor + generator config
- `e2e/`, `integration/` — test suites, run through `make` (see Commands)
- `examples/` — runnable docker-compose stack for SDK users (separate from `e2e/compose/`)
- `tests/` — fixtures for the compose stack (Keycloak realm); not Go test code

Read the matching doc before changing its area:

- `docs/ARCHITECTURE.md` — request lifecycle, bootstrap, project scoping
- `docs/PACKAGES.md` — per-package contracts and dependency graph
- `docs/AUTHENTICATION.md` / `docs/AUTHORIZATION.md` — auth flows; assignment model
- `docs/CLI.md` — `lkctl` command tree, flags, env vars
- `docs/GENERATION.md` — SDK generation pipeline

## Commands

Every workflow has a `make` target; use it rather than the raw tool, because the targets provision `.env`, the compose stack, and `CONTAINER_ENGINE` (some hosts run podman only).

- `make build` — builds `dist/lkctl`; runs `mod`, `fmt`, `vet`, `test` first, so it rewrites files.
- `make test` — unit tests for `./pkg/...` only, with coverage.
- `make validate` — `vet` + `lint`. `make fmt` applies golangci-lint fixes (gofumpt, goimports). golangci-lint runs via `go run`; nothing to install.
- `make test-integration` — starts Lakekeeper + Keycloak + MinIO + OpenFGA via compose and runs the `integration`-tagged tests.
- `make test-e2e-compose` — runs the `e2e_cli` suite against the host-built `lkctl` on the compose stack. Run it before calling any `e2e/` change done.
- `make test-e2e-kind` — runs the auth-only subset of `e2e_cli` against a kind cluster; other tests skip via `requireBackend`.
- `make clean` — tears down the compose stack (`down --volumes`), removes `bin/`, `coverage.txt`, `.env`.

`KEEP_STACK=1` leaves the compose stack running after integration/e2e tests, for iterative debugging; tear down with `make clean`.

In Makefiles and scripts, call the container runtime through `$(CONTAINER_ENGINE)` / `$CONTAINER_ENGINE`.

## Code style

Import aliases are enforced by `importas`; use exactly these:

- `github.com/sirupsen/logrus` → `log`
- `github.com/lakekeeper/go-lakekeeper/pkg/apis/management/v1` → `managementv1`

## Generation

`make generate` regenerates `pkg/apis/management/v1/` from `api/openapi/management-open-api.yaml` (preprocess → `openapi-generator-cli` → postprocess). To change generated code, change the spec, the pre/postprocessor, or the generator config, then regenerate.

`api/openapi/*-config.yaml` lists only deviations from generator defaults; set an option there only when its value differs from the default.

## Commits and branches

- Conventional Commits, parsed by release-please for versioning and `CHANGELOG.md`. Use only `feat`, `fix`, `docs`, `chore`: the only types release-please is configured for.
- Branches: `type/name`, using the commit types above (e.g. `fix/role-list-pagination`).
- `CHANGELOG.md` and `.release-please-manifest.json` belong to release-please; leave them untouched.

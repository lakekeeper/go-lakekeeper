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
- `api/openapi/` — spec + postprocessor + generator config
- `e2e/`, `integration/` — test suites, run through `make` (see Commands)
- `examples/` — runnable docker-compose stack for SDK users (separate from `e2e/compose/`)
- `tests/` — fixtures for the compose stack (Keycloak realm); not Go test code
- `.agents/skills/` — agent skills (see Skills)

Before changing an area, read its doc:

- `docs/ARCHITECTURE.md` — request lifecycle, bootstrap, project scoping
- `docs/PACKAGES.md` — per-package contracts and dependency graph
- `docs/AUTHENTICATION.md` / `docs/AUTHORIZATION.md` — auth flows; assignment model
- `docs/CLI.md` — `lkctl` command tree, flags, env vars
- `docs/GENERATION.md` — SDK generation pipeline

## Commands

Every workflow has a `make` target; use it rather than the raw tool, because the targets provision `.env`, the compose stack, and `CONTAINER_ENGINE` (some hosts run podman only).

- `make build` — builds `dist/lkctl`; runs `vet` and `test` first. It does not rewrite files; run `make mod` / `make fmt` explicitly.
- `make test` — unit tests for `./...` with coverage; integration/e2e tests are excluded by their build tags.
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

`forbidigo` rejects any reference from `cmd/` to the generated services (`c.WarehouseAPI`, `*managementv1.XAPIService`, `managementv1.NewAPIClient`); `lkctl` calls the `pkg/client` façades only. A command that needs an operation without a façade adds the façade method first.

## Generation

`make generate` regenerates `pkg/apis/management/v1/` from `api/openapi/management-open-api.yaml` (`openapi-generator-cli` → postprocess). To change generated code, change the spec, the postprocessor, or the generator config, then regenerate.

`api/openapi/*-config.yaml` lists only deviations from generator defaults; set an option there only when its value differs from the default.

## Skills

Skills live in `.agents/skills/<name>/SKILL.md`, in a format any agent can read: YAML front matter (`name`, `description`) and Markdown instructions. When a task matches a skill's `description`, read its `SKILL.md` and follow it.

- `verify` — run `make validate` and `make test` before calling a task done or opening a PR.
- `api-coverage` — compare the OpenAPI spec with the generated SDK and the `pkg/client` façades, and draft a gap report.

`.claude/skills` is a symlink to `.agents/skills`, because Claude Code only discovers skills under `.claude/skills/`. Add and edit skills in `.agents/skills/` only; never replace the symlink with a copy.

## Hard rules

- Never edit `pkg/apis/management/v1/` by hand; regenerate it (see Generation).
- Never edit `CHANGELOG.md` or `.release-please-manifest.json`; release-please maintains them.
- Never push to `main` and never force-push; work on a branch and open a pull request.

## Commits and branches

[`CONTRIBUTING.md`](CONTRIBUTING.md#commits-and-branches) is the source of truth for commit messages, PR titles, and branch names. Read it before committing or opening a pull request.

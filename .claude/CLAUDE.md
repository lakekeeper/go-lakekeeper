# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project

Go client library plus `lkctl` CLI for the Lakekeeper Iceberg catalog. Single-module project; requires Go 1.25+.

Keep this `.claude/` tree small. Most colleagues are new to Claude Code; prefer notes here and on-demand skills over hooks or `settings.json` automation.

## Architecture

- `pkg/apis/management/v1/` — generated SDK (see Generation below)
- `pkg/client/` — facade over the SDK (auth, retries, optional bootstrap)
- `pkg/core/` — auth, context, and error primitives shared across packages
- `pkg/lakekeeper/` — umbrella re-exporter; the single import path most consumers use (`lakekeeper.New`, `lakekeeper.NewS3Profile`, ...)
- `pkg/storage/`, `pkg/permissions/` — hand-written helpers / builders
- `pkg/common/`, `pkg/testutil/`, `pkg/version/` — small shared helpers (defaults, test client, build-time version)
- `cmd/` — Cobra-based CLI; entry at `cmd/main.go`, commands tree under `cmd/lkctl/commands/`
- `api/openapi/` — spec + preprocessor + generator config
- `e2e/`, `integration/` — test suites (see `test-*` make targets)
- `examples/` — runnable docker-compose stack for SDK users (separate from `e2e/compose/`)
- `tests/` — fixtures for the compose stack (Keycloak realm); not Go test code

See `docs/ARCHITECTURE.md` for the design and `docs/PACKAGES.md` for per-package contracts. Other docs in `docs/`: `CLI.md`, `AUTHENTICATION.md`, `AUTHORIZATION.md`, `GENERATION.md`.

## Commands

Everything goes through `make`:

- `make build` — builds `dist/lkctl`; runs `mod`, `fmt`, `vet`, `test` first.
- `make test` — unit tests, `./pkg/...` only, with coverage.
- `make test-integration` — spins up Lakekeeper + Keycloak + MinIO + OpenFGA via docker-compose and runs tests tagged `integration`. Do **not** call `go test -tags integration` directly; use the make target so the stack and `.env` are provisioned. Set `KEEP_STACK=1` to leave the compose stack running after the test exits (useful for iterative debugging; tear down with `make clean`).
- `make test-e2e-compose` — spins up the compose stack and runs the `e2e_cli` suite against the host-built `lkctl`. Do **not** call `./e2e/compose/run.sh` or `go test -tags e2e_cli` directly; the make target wires `CONTAINER_ENGINE` and `.env` so podman-only hosts work.
- `make test-e2e-kind` — runs the `e2e_cli` suite against a kind cluster (allowlist of four auth-only tests; lifecycle tests skip via `requireBackend`).
- `make test-e2e` — runs `test-e2e-compose` then `test-e2e-kind` sequentially.
- `make fmt` — `golangci-lint run --fix ./...` (runs gofumpt + goimports via golangci-lint v2).
- `make lint` — `golangci-lint run ./...`.
- `make validate` — `vet` + `lint`.
- `make snapshot` — goreleaser snapshot build.
- `make clean` — tears down compose stack (`down --volumes`), removes `bin/`, `coverage.txt`, `.env`.

golangci-lint is invoked via `go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint ...` inside the Makefile — no separate install step needed.

## Code style

- Formatters: `gofumpt` + `goimports` (configured in `.golangci.yml`).
- Import aliases are linter-enforced via `importas` (see `.golangci.yml`):
  - `github.com/sirupsen/logrus` → `log`
  - `github.com/lakekeeper/go-lakekeeper/pkg/apis/management/v1` → `managementv1`

  Do not introduce alternative aliases.

## Generation

`make generate` regenerates `pkg/apis/management/v1/` from `api/openapi/management-open-api.yaml` via a preprocessor + `openapi-generator-cli`. See [docs/GENERATION.md](../docs/GENERATION.md) for the full pipeline.

**`api/openapi/*-config.yaml` convention:** only set options whose value differs from the generator default. Don't pin defaults explicitly — these files document deviations, not the full surface.

## Discipline

These skills are loaded in this environment and should be **actively invoked**
when their domain comes up — not just consulted by their description line.

- **`dev-discipline:tdd-bdd`** — Red-Green-Refactor governs implementation.
  The implementer (sub-agent or main agent in execution mode) must load this
  skill at the start of work and cycle one behavior at a time. Plans should
  name the behaviors needing tests and identify TDD as the discipline; they
  should **not** script cycle mechanics (per-step Red/Green/Refactor
  sub-bullets) — that's an execution concern, not an architecture one.
- **`go-dev:go-idioms`** — settled Go-idiom questions (e.g. *accept interfaces,
  return structs*, error wrapping style, package layout) are not user choices.
  State the canonical answer directly; do not surface them for selection.
- **`lakekeeper-knowledge:lakekeeper-concepts`** — apply when reasoning about
  Server / Project / Warehouse / Namespace / Role entities, the Management API
  surface, or relationships to Postgres / Vault / OpenFGA / external IdPs.
  Prefer the skill's vocabulary over guessing from the code.

## Commits and branches

- **Conventional Commits are required.** `release-please` parses messages to compute versions and generate `CHANGELOG.md`. Only these types are configured: `feat`, `fix`, `docs`, `chore`. Do not use `refactor`, `test`, `ci`, `style`, `perf`, or `build` — release-please will ignore or mishandle them.
- Branches use `user/type/name`.
- Do **not** hand-edit `CHANGELOG.md` or `.release-please-manifest.json` — release-please manages them.

## Personal vs team rules

Anything in this `CLAUDE.md` is team-shared and committed. Personal/session-level learnings live in `~/.claude/projects/-<repo-slug>/memory/` — not checked in, not propagated to teammates. If a personal rule turns out to be broadly applicable, propose adding it here.

# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project

Go client library plus `lkctl` CLI for the Lakekeeper Iceberg catalog. Single-module project; requires Go 1.24+.

## Commands

Everything goes through `make`:

- `make build` — builds `dist/lkctl`; runs `mod`, `fmt`, `vet`, `test` first.
- `make test` — unit tests, `./pkg/...` only, with coverage.
- `make test-integration` — spins up Lakekeeper + Keycloak + MinIO + OpenFGA via docker-compose and runs tests tagged `integration`. Do **not** call `go test -tags integration` directly; use the make target so the stack and `.env` are provisioned.
- `make fmt` — `golangci-lint run --fix ./...` (runs gofumpt + goimports via golangci-lint v2).
- `make lint` — `golangci-lint run ./...`.
- `make validate` — `vet` + `lint`.
- `make snapshot` — goreleaser snapshot build.
- `make clean` — tears down compose stack (`down --volumes`), removes `bin/`, `coverage.txt`, `.env`.

golangci-lint is invoked via `go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint ...` inside the Makefile — no separate install step needed.

## Code style

- Formatters: `gofumpt` + `goimports` (configured in `.golangci.yml`).
- Import aliases are linter-enforced via `importas`:
  - `github.com/sirupsen/logrus` → `log`
  - management / permission / storage API packages → `managementv1` / `permissionv1` / `storagev1`

  Do not introduce alternative aliases.

## Commits and branches

- **Conventional Commits are required.** `release-please` parses messages to compute versions and generate `CHANGELOG.md` (`feat:`, `fix:`, `chore:`, `docs:`, etc.).
- Branches use `user/type/name`.
- Do **not** hand-edit `CHANGELOG.md` or `.release-please-manifest.json` — release-please manages them.

## Integration-test environment

`make test-integration` creates `.env` (if missing) with test credentials, then runs `./run-tests.sh`, which brings up `docker-compose.yml` and waits via `./scripts/await-healthy.sh`. Requires a running Docker daemon plus `docker compose` (or `docker-compose`).

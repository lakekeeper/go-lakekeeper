---
name: integration-up
description: Bring up the integration-test stack (Lakekeeper, Keycloak, MinIO, OpenFGA via docker-compose) and run integration tests. Use only when the user explicitly asks for integration testing — this starts real services.
disable-model-invocation: true
---

From the repo root:

1. Confirm the Docker daemon is running: `docker info`. If this fails, ask the user to start Docker before continuing.
2. Run `make test-integration`. This will:
   - Generate `.env` if missing (pre-filled test credentials for the local Keycloak).
   - Launch the compose stack, wait for health via `./scripts/await-healthy.sh`, and run `go test -v -tags integration ./integration/...`.
3. When the user is done, remind them that `make clean` stops the stack (`docker compose down --volumes`) and removes `.env` and `coverage.txt`.

Do **not** call `go test -tags integration` directly — the stack must be up and `.env` must exist. Use the make target.

Optional: `LAKEKEEPER_VERSION=<tag> make test-integration` runs against a specific Lakekeeper image (defaults to `latest-main`).

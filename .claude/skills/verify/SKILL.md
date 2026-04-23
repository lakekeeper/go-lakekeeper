---
name: verify
description: Run full local validation — go vet, golangci-lint, and unit tests. Use before marking a task complete or before opening a PR.
---

From the repo root:

1. `make validate` — runs `go vet` then `golangci-lint run ./...`.
2. `make test` — runs unit tests under `./pkg/...` with coverage.

Report any failures with exact output. Do not attempt to fix lint/test failures without confirming with the user first, except for trivially obvious cases (e.g., an unused import in code you just touched).

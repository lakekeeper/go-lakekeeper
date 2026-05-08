#!/usr/bin/env bash
#
# E2E test runner for the lkctl CLI against the docker-compose stack.
# Mirrors run-tests.sh but drives the e2e/cli suite (build tag e2e_cli) and
# builds the lkctl binary once before launching go test.
#
# Honours KEEP_STACK=1 to skip teardown for post-mortem debugging.

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$REPO_ROOT"

CONTAINER_ENGINE="${CONTAINER_ENGINE:-docker}"
CONTAINER_COMPOSE_ENGINE="${CONTAINER_COMPOSE_ENGINE:-docker compose}"
export CONTAINER_ENGINE CONTAINER_COMPOSE_ENGINE

# Probe the resolved engine (not a literal `docker`) so podman-only hosts
# don't get rejected here. CONTAINER_COMPOSE_ENGINE may legitimately contain
# a space (e.g. `docker compose`), so we let `up -d` surface its own error.
for cmd in "$CONTAINER_ENGINE" go; do
    command -v "$cmd" >/dev/null 2>&1 || {
        echo "missing required tool: $cmd (install before running e2e/compose)" >&2
        exit 1
    }
done

# Provision .env using the same defaults as `make test-integration`. We let
# stderr through so a real failure (missing .env.example, broken make rule)
# surfaces here rather than as a cryptic auth error inside go test.
make .env >/dev/null
set -a
# shellcheck disable=SC1091
source .env
set +a

teardown() {
    if [ "${KEEP_STACK:-0}" = "1" ]; then
        echo "KEEP_STACK=1: leaving compose stack running"
        return
    fi
    $CONTAINER_COMPOSE_ENGINE down --volumes
}
trap teardown EXIT

$CONTAINER_COMPOSE_ENGINE down --volumes
LAKEKEEPER_VERSION=${LAKEKEEPER_VERSION:-latest-main} $CONTAINER_COMPOSE_ENGINE up -d

./scripts/await-healthy.sh

# Build lkctl once and pass it to the suite via LKCTL_BIN; saves N goroutine
# build invocations on slow machines.
LKCTL_BIN="$REPO_ROOT/dist/lkctl-e2e"
mkdir -p "$(dirname "$LKCTL_BIN")"
go build -o "$LKCTL_BIN" ./cmd
export LKCTL_BIN

export LKCTL_E2E_BACKEND=compose

go test -v -tags e2e_cli ./e2e/cli/...

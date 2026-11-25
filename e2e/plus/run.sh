#!/usr/bin/env bash
#
# Runs the Lakekeeper+ suites against the compose stack with Lakekeeper+ in
# place of Lakekeeper, once per authorization backend variant:
#
#   cedar    compose profile plus,         E2E backend compose-plus
#   openfga  compose profile plus-openfga, E2E backend compose-plus-openfga
#
# Each variant runs the lkctl E2E tests tagged e2e_plus (plus the read-only
# tests shared with the Lakekeeper suite, picked by E2E_RUN) and the SDK
# tests tagged integration_plus.
#
# Environment:
#   LAKEKEEPER_PLUS_VARIANTS  variants to run, default "cedar openfga"
#   LAKEKEEPER_PLUS_VERSION   image tag prefix, default "v0.15.1"
#   LAKEKEEPER__LICENSE__KEY  Lakekeeper+ license; passed to the server when
#                             set. Optional for local test runs; CI sets it.
#   LICENSE_EXPIRY_NOTICE     if set, a file path: after the first variant is
#                             healthy, the license expiry check writes an
#                             issue body there, or leaves it empty when no
#                             notice is due (CI only)
#   KEEP_STACK=1              leave the last stack running
#
# The license is never printed: the server info dump in
# scripts/await-healthy.sh drops all license-status fields except type and
# validity, and the expiry check reads server info from a pipe.

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$REPO_ROOT"

CONTAINER_ENGINE="${CONTAINER_ENGINE:-docker}"
CONTAINER_COMPOSE_ENGINE="${CONTAINER_COMPOSE_ENGINE:-docker compose}"
export CONTAINER_ENGINE CONTAINER_COMPOSE_ENGINE

for cmd in "$CONTAINER_ENGINE" go jq; do
    command -v "$cmd" >/dev/null 2>&1 || {
        echo "missing required tool: $cmd (install before running e2e/plus)" >&2
        exit 1
    }
done

VARIANTS="${LAKEKEEPER_PLUS_VARIANTS:-cedar openfga}"
LAKEKEEPER_PLUS_VERSION="${LAKEKEEPER_PLUS_VERSION:-v0.15.1}"
export LAKEKEEPER_PLUS_VERSION

# Read-only Lakekeeper E2E tests that also run against Lakekeeper+, and the
# Lakekeeper+ tests (prefix TestPlus, build tag e2e_plus).
E2E_RUN='^(TestServerInfoJSON|TestServerInfoText|TestPlus.*)$'

if [ -n "${LAKEKEEPER__LICENSE__KEY:-}" ]; then
    echo "Lakekeeper+ license: set in the environment, passed to the server"
else
    echo "Lakekeeper+ license: not set (optional for local test runs)"
fi

make .env >/dev/null
set -a
# shellcheck disable=SC1091
source .env
set +a

compose() {
    # Both Lakekeeper+ profiles, so that down also removes their containers.
    $CONTAINER_COMPOSE_ENGINE --profile plus --profile plus-openfga "$@"
}

teardown() {
    if [ "${KEEP_STACK:-0}" = "1" ]; then
        echo "KEEP_STACK=1: leaving compose stack running"
        return
    fi
    compose down --volumes
}
trap teardown EXIT

LKCTL_BIN="$REPO_ROOT/dist/lkctl-e2e"
mkdir -p "$(dirname "$LKCTL_BIN")"
go build -o "$LKCTL_BIN" ./cmd
export LKCTL_BIN

# check_license_expiry writes the expiry notice (or nothing) to
# $LICENSE_EXPIRY_NOTICE. It never fails the run: only the tests decide it.
check_license_expiry() {
    : > "$LICENSE_EXPIRY_NOTICE"
    if ! "$LKCTL_BIN" --base-url "$LAKEKEEPER_BASE_URL" --auth-mode oauth2 \
        --token-url "$LAKEKEEPER_TOKEN_URL" --client-id "$LAKEKEEPER_CLIENT_ID" \
        --client-secret "$LAKEKEEPER_CLIENT_SECRET" --scopes "$LAKEKEEPER_SCOPE" \
        server info --output json 2>/dev/null \
        | go run ./e2e/plus/licenseexpiry > "$LICENSE_EXPIRY_NOTICE"; then
        echo "license expiry check failed; no notice written" >&2
        : > "$LICENSE_EXPIRY_NOTICE"
    fi
}

expiry_checked=0
for variant in $VARIANTS; do
    case "$variant" in
        cedar)   service=lakekeeper-plus;         backend=compose-plus ;;
        openfga) service=lakekeeper-plus-openfga; backend=compose-plus-openfga ;;
        *) echo "unknown Lakekeeper+ variant: $variant" >&2; exit 1 ;;
    esac
    echo "=== Lakekeeper+ ($variant authorization backend)"

    compose down --volumes
    # Naming the service starts it and its dependencies only, not the
    # Lakekeeper service of the default stack.
    compose up -d "$service"

    ./scripts/await-healthy.sh

    if [ -n "${LICENSE_EXPIRY_NOTICE:-}" ] && [ "$expiry_checked" = 0 ]; then
        check_license_expiry
        expiry_checked=1
    fi

    LKCTL_E2E_BACKEND="$backend" \
        go test -count=1 -v -tags e2e_cli,e2e_plus ./e2e/cli/... -run "$E2E_RUN"

    LAKEKEEPER_PLUS_AUTHZ_BACKEND="$variant" \
        go test -count=1 -v -tags integration_plus ./integration/plus/...
done

#!/usr/bin/env bash
#
# Supported-version E2E runner (make test-e2e-versions). Runs the
# e2e_versions-tagged lkctl tests (prefix TestVersions) against Lakekeeper
# pinned to two minors derived from the Lakekeeper spec release in
# api/openapi/spec-releases.yaml:
#
#   previous minor   the minor before the spec release, the oldest one lkctl
#                    supports (backend compose-previous-minor)
#   older minor      the minor before that, outside the supported range
#                    (backend compose-older-minor)
#
# Each runs on its own fresh compose stack, using the floating minor image tag
# (for example v0.14), which tracks that minor's latest patch release.
#
# Honours KEEP_STACK=1 to leave the last stack running.

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$REPO_ROOT"

CONTAINER_ENGINE="${CONTAINER_ENGINE:-docker}"
CONTAINER_COMPOSE_ENGINE="${CONTAINER_COMPOSE_ENGINE:-docker compose}"
export CONTAINER_ENGINE CONTAINER_COMPOSE_ENGINE

for cmd in "$CONTAINER_ENGINE" go; do
    command -v "$cmd" >/dev/null 2>&1 || {
        echo "missing required tool: $cmd (install before running e2e/versions)" >&2
        exit 1
    }
done

release="$(sed -n 's/^lakekeeper: *//p' api/openapi/spec-releases.yaml)"
if [[ ! "$release" =~ ^v([0-9]+)\.([0-9]+)\.[0-9]+$ ]]; then
    echo "api/openapi/spec-releases.yaml: lakekeeper release \"$release\" is not vX.Y.Z" >&2
    exit 1
fi
major="${BASH_REMATCH[1]}"
minor="${BASH_REMATCH[2]}"
if [ "$minor" -lt 2 ]; then
    echo "spec release $release: the previous two minors are not on the same major; pin them by hand" >&2
    exit 1
fi
PREVIOUS_MINOR="v$major.$((minor - 1))"
OLDER_MINOR="v$major.$((minor - 2))"
echo "Lakekeeper spec release $release: previous minor $PREVIOUS_MINOR, older minor $OLDER_MINOR"

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

LKCTL_BIN="$REPO_ROOT/dist/lkctl-e2e"
mkdir -p "$(dirname "$LKCTL_BIN")"
go build -o "$LKCTL_BIN" ./cmd
export LKCTL_BIN

for pair in "$PREVIOUS_MINOR:compose-previous-minor" "$OLDER_MINOR:compose-older-minor"; do
    version="${pair%%:*}"
    backend="${pair#*:}"
    echo "=== Lakekeeper $version ($backend)"

    $CONTAINER_COMPOSE_ENGINE down --volumes
    LAKEKEEPER_VERSION="$version" $CONTAINER_COMPOSE_ENGINE up -d

    ./scripts/await-healthy.sh

    LKCTL_E2E_BACKEND="$backend" \
        go test -count=1 -v -tags e2e_cli,e2e_versions ./e2e/cli/... -run '^TestVersions'
done

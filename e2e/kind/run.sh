#!/usr/bin/env bash
#
# E2E test runner under kind. Provides a real Kubernetes API server so
# `lkctl --auth-mode k8s` can be exercised against an honest projected SA
# token (audience=lakekeeper) — that's the *only* delta vs the compose
# harness, which is why this runner only drives the kind-relevant subset
# of tests.
#
# Honours KEEP_STACK=1 to skip teardown for post-mortem debugging.

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$REPO_ROOT"

# Resolve the container engine before probing so a podman-only host isn't
# rejected for a missing `docker` binary. kind auto-detects podman when no
# `docker` binary is on PATH, so we don't force KIND_EXPERIMENTAL_PROVIDER=podman
# here. If kind ever stops detecting podman on a supported host, reintroduce
# `export KIND_EXPERIMENTAL_PROVIDER=podman` below with a note recording the
# kind version and the failure mode.
CONTAINER_ENGINE="${CONTAINER_ENGINE:-docker}"
export CONTAINER_ENGINE

# Probe required tools up front so colleagues new to the kind harness get a
# friendly hint instead of `kind: command not found` mid-run.
for cmd in kind kubectl helm "$CONTAINER_ENGINE" go; do
    command -v "$cmd" >/dev/null 2>&1 || {
        echo "missing required tool: $cmd (install before running e2e/kind)" >&2
        exit 1
    }
done

CLUSTER_NAME="${LKCTL_E2E_CLUSTER:-lkctl-e2e}"
NAMESPACE="${LKCTL_E2E_NAMESPACE:-default}"
POD="${LKCTL_E2E_POD:-lkctl-runner}"
LKCTL_IMAGE="${LKCTL_E2E_IMAGE:-localhost/lkctl-e2e:dev}"
LAKEKEEPER_CHART_VERSION="${LAKEKEEPER_CHART_VERSION:-0.10.1}"

TMP_TAR=""
teardown() {
    [ -n "$TMP_TAR" ] && rm -f "$TMP_TAR"
    if [ "${KEEP_STACK:-0}" = "1" ]; then
        echo "KEEP_STACK=1: leaving kind cluster '$CLUSTER_NAME' running"
        return
    fi
    kind delete cluster --name "$CLUSTER_NAME" >/dev/null 2>&1 || true
}
trap teardown EXIT

# 1. Provision the cluster.
if ! kind get clusters | grep -qx "$CLUSTER_NAME"; then
    kind create cluster --name "$CLUSTER_NAME" --config "$REPO_ROOT/e2e/kind/cluster.yaml"
fi
kubectl cluster-info --context "kind-$CLUSTER_NAME"

# 2. Build lkctl for linux (the runner image is alpine), then load into cluster.
# Match the Docker daemon's architecture so the binary runs in-pod regardless
# of the host's GOOS/GOARCH (macOS-arm64 dev, linux-amd64 CI).
KIND_GOARCH="$($CONTAINER_ENGINE info -f '{{.Architecture}}' 2>/dev/null \
    | sed -e 's/x86_64/amd64/' -e 's/aarch64/arm64/' \
    || echo amd64)"
mkdir -p "$REPO_ROOT/dist"
GOOS=linux GOARCH="$KIND_GOARCH" CGO_ENABLED=0 \
    go build -o "$REPO_ROOT/dist/lkctl" ./cmd

$CONTAINER_ENGINE build -t "$LKCTL_IMAGE" -f "$REPO_ROOT/e2e/kind/Dockerfile.lkctl" "$REPO_ROOT"

# Use save+load instead of `kind load docker-image`: podman can't resolve the
# bare reference passed to that subcommand even when the image is in its store,
# so we hand kind a tarball it can ingest unambiguously. Both engines support
# `save -o` and `kind load image-archive` identically — no engine branching.
# TMP_TAR is declared above the teardown trap so cleanup also removes the tar.
TMP_TAR="$(mktemp -t lkctl-e2e-XXXXXX.tar)"
"$CONTAINER_ENGINE" save -o "$TMP_TAR" "$LKCTL_IMAGE"
kind load image-archive "$TMP_TAR" --name "$CLUSTER_NAME"

# 3. Install Lakekeeper + dependencies via Helm. Idempotent so KEEP_STACK=1
# reruns don't trip "release already exists". No Keycloak: the kind harness
# drives Kubernetes SA auth only; cluster.yaml pins the
# service-account-issuer/jwks-uri so Lakekeeper's additionalIssuers can
# validate projected SA tokens against the in-cluster OIDC endpoint.
helm upgrade --install lakekeeper lakekeeper \
    --repo https://lakekeeper.github.io/lakekeeper-charts/ \
    --version "$LAKEKEEPER_CHART_VERSION" \
    --namespace "$NAMESPACE" \
    --create-namespace \
    -f "$REPO_ROOT/e2e/kind/values.yaml" \
    --wait \
    --timeout 10m

# 4. Stand up the runner pod with a projected SA token.
kubectl apply -n "$NAMESPACE" -f "$REPO_ROOT/e2e/kind/pod-lkctl.yaml"
kubectl wait --for=condition=Ready pod/"$POD" -n "$NAMESPACE" --timeout=2m

# 5. Bootstrap the server through the in-pod CLI before tests start. The
# pod's projected SA token at the default path covers auth, so no
# --base-url/--k8s-token-path overrides are needed.
kubectl exec -n "$NAMESPACE" "$POD" -- \
    lkctl --auth-mode k8s server bootstrap --accept-terms-of-use || true

# 6. Run the kind-relevant subset of the suite. The active backend is
#    selected by LKCTL_E2E_BACKEND; the harness uses kubectl exec for every
#    invocation, so go test runs on the host. LAKEKEEPER_BASE_URL is the
#    in-cluster URL — host-side tests pass it through to lkctl, which then
#    resolves it inside the pod.
export LKCTL_E2E_BACKEND=kind
export LKCTL_E2E_NAMESPACE="$NAMESPACE"
export LKCTL_E2E_POD="$POD"
export LAKEKEEPER_BASE_URL="http://lakekeeper.${NAMESPACE}.svc.cluster.local:8181"

# Explicit allow-list — TestWhoamiOAuth2 / TestWhoamiAccessToken are
# compose-only (no Keycloak in this harness) and must not match by prefix.
go test -v -tags e2e_cli ./e2e/cli/... \
    -run 'TestWhoamiK8sServiceAccount|TestServerInfo|TestServerBootstrapRejectsRebootstrap|TestBadCredentialsExitNonZero'

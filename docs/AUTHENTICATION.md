# Authentication

This guide walks from "which auth flow do I want?" to a working setup on
both `lkctl` and the Go SDK. For the underlying reference material:

- SDK: see [PACKAGES.md `pkg/core` Authentication](PACKAGES.md#authentication)
  for the `AuthSource` interface, all three implementations, and the OAuth2
  sequence diagram.
- CLI: see [CLI.md Global flags & environment variables](CLI.md#global-flags--environment-variables)
  for the full flag table and per-mode invocations.

## Overview

`AuthSource` ([`pkg/core/auth.go`](../pkg/core/auth.go)) is the single
abstraction both surfaces consume. `lkctl` selects an implementation from
`--auth-mode`; the SDK accepts one directly via
`client.NewWithAuthSource`. Three implementations ship in `pkg/core`:
`OAuthTokenSource`, `AccessTokenAuthSource`, `K8sServiceAccountAuthSource`.

## Choosing a flow

| Scenario | Mode |
|---|---|
| Production service with an OIDC IdP (Keycloak, Dex, …) | `oauth2` |
| Short-lived script, CI job, manual testing | `token` |
| Workload running inside a Kubernetes pod | `k8s` |
| Custom token logic (refresh token, device flow, …) | implement `AuthSource` directly (SDK only) |

The remaining sections show each flow on the CLI and the SDK side by side.

## OAuth2 client credentials

The default mode. Tokens are obtained from an OIDC token endpoint and
re-fetched transparently when expired by `golang.org/x/oauth2`.

**CLI** — see [CLI.md auth-mode examples](CLI.md#auth-mode-examples) for
the full flag table:

```sh
export LAKEKEEPER_BASE_URL=http://localhost:8181
# LAKEKEEPER_AUTH_MODE=oauth2 is the default — no need to set it
export LAKEKEEPER_TOKEN_URL=http://localhost:30080/realms/iceberg/protocol/openid-connect/token
export LAKEKEEPER_CLIENT_ID=<your-client-id>
export LAKEKEEPER_CLIENT_SECRET=<your-client-secret>
export LAKEKEEPER_SCOPE=lakekeeper

lkctl whoami        # smoke test
```

**SDK** — see [PACKAGES.md `OAuthTokenSource`](PACKAGES.md#oauthtokensource--oauth-20-client-credentials)
for the sequence diagram and full reference:

```go
import (
    "golang.org/x/oauth2/clientcredentials"

    "github.com/lakekeeper/go-lakekeeper/pkg/client"
    "github.com/lakekeeper/go-lakekeeper/pkg/core"
)

cfg := &clientcredentials.Config{
    ClientID:     "<your-client-id>",
    ClientSecret: "<your-client-secret>",
    TokenURL:     "http://localhost:30080/realms/iceberg/protocol/openid-connect/token",
    Scopes:       []string{"lakekeeper"},
}

as := &core.OAuthTokenSource{TokenSource: cfg.TokenSource(ctx)}
c, err := client.NewWithAuthSource(ctx, "http://localhost:8181", as)
```

## Static access token

Use when a bearer token is obtained out-of-band (CI secret, manual
impersonation, scripted tests). Renewal is the caller's problem — once
the token expires, requests return 401. For long-running processes, use
OAuth2 instead.

**CLI** — see [CLI.md static-access-token example](CLI.md#auth-mode-examples):

```sh
export LAKEKEEPER_BASE_URL=http://localhost:8181
export LAKEKEEPER_AUTH_MODE=token
export LAKEKEEPER_ACCESS_TOKEN=<your-bearer-token>

lkctl whoami        # smoke test
```

**SDK** — see [PACKAGES.md `AccessTokenAuthSource`](PACKAGES.md#accesstokenauthsource--static-bearer-token):

```go
as := &core.AccessTokenAuthSource{Token: "eyJhbGci..."}
c, err := client.NewWithAuthSource(ctx, baseURL, as)

// Equivalent shorthand:
c, err := client.New(ctx, baseURL, "eyJhbGci...")
```

## Kubernetes service account

For workloads running inside a pod whose projected service-account token
is accepted by the Lakekeeper server. The default token path
(`/var/run/secrets/kubernetes.io/serviceaccount/token`, see
[`pkg/core/auth.go:16`](../pkg/core/auth.go)) covers the standard
projected-volume mount; override only for non-default mounts (e.g.
audience-scoped tokens). The token is read **once at construction** — the
pod must restart to pick up rotation.

**CLI** — see [CLI.md k8s example](CLI.md#auth-mode-examples):

```sh
export LAKEKEEPER_BASE_URL=http://lakekeeper.lakekeeper.svc:8181
export LAKEKEEPER_AUTH_MODE=k8s
# --k8s-token-path / LAKEKEEPER_K8S_TOKEN_PATH only needed for non-default mounts

lkctl whoami        # smoke test
```

**SDK** — see [PACKAGES.md `K8sServiceAccountAuthSource`](PACKAGES.md#k8sserviceaccountauthsource--kubernetes-service-account):

```go
// Default token path
as := &core.K8sServiceAccountAuthSource{}

// Or with a custom path:
path := "/var/run/secrets/lakekeeper/token"
as := &core.K8sServiceAccountAuthSource{ServiceAccountTokenPath: &path}

c, err := client.NewWithAuthSource(ctx, baseURL, as)
```

## Bootstrap

Bootstrap is auth-adjacent: until a Lakekeeper server is bootstrapped,
the only operation it accepts is `POST /management/v1/bootstrap`. The
first authenticated principal that calls bootstrap becomes the
server's operator. Both `lkctl` and the SDK can perform this once during
setup — pick whichever surface owns your provisioning flow.

**CLI** — explicit, or auto-bootstrap via the global flag (see
[CLI.md `lkctl server`](CLI.md#lkctl-server)):

```sh
# Explicit one-shot
lkctl server bootstrap --accept-terms-of-use --as-operator

# Or have any command bootstrap-on-first-run
lkctl project list --bootstrap   # equivalent to LAKEKEEPER_BOOTSTRAP=true
```

**SDK** — `client.WithInitialBootstrap` (see
[PACKAGES.md client options](PACKAGES.md#options)):

```go
import (
    managementv1 "github.com/lakekeeper/go-lakekeeper/pkg/apis/management/v1"
    "github.com/lakekeeper/go-lakekeeper/pkg/client"
    "github.com/lakekeeper/go-lakekeeper/pkg/core"
)

c, err := client.NewWithAuthSource(ctx, baseURL, as,
    client.WithInitialBootstrap(
        true,                                       // acceptTermsOfUse
        true,                                       // isOperator
        core.Ptr(managementv1.USERTYPE_APPLICATION), // userType (optional)
    ),
)
```

`acceptTermsOfUse=false` makes the option a no-op. If the server is
already bootstrapped, the option is also a no-op — it's safe to leave
on across restarts.

## Environment variables reference

The CLI honours these variables (names in
[`pkg/common/env.go`](../pkg/common/env.go), defaults in
[`pkg/common/defaults.go`](../pkg/common/defaults.go); K8s token-path
default in [`pkg/core/auth.go`](../pkg/core/auth.go)). The SDK does
**not** read the environment — pass values explicitly to the
constructors, or reuse the same `pkg/common` constants if you want
symmetric behaviour in your own binaries.

| Var | Default | Used by | Notes |
|---|---|---|---|
| `LAKEKEEPER_BASE_URL` | `http://localhost:8181` | CLI | SDK takes baseURL as a `New*` argument |
| `LAKEKEEPER_AUTH_MODE` | `oauth2` | CLI | SDK selects mode by `AuthSource` type |
| `LAKEKEEPER_TOKEN_URL` | _(none)_ | CLI | OAuth2 only |
| `LAKEKEEPER_CLIENT_ID` | _(none)_ | CLI | OAuth2 only |
| `LAKEKEEPER_CLIENT_SECRET` | _(none)_ | CLI | OAuth2 only |
| `LAKEKEEPER_SCOPE` | `lakekeeper` | CLI | OAuth2 only; space-separated |
| `LAKEKEEPER_ACCESS_TOKEN` | _(none)_ | CLI | Token mode only |
| `LAKEKEEPER_K8S_TOKEN_PATH` | `/var/run/secrets/kubernetes.io/serviceaccount/token` | CLI | K8s mode only |
| `LAKEKEEPER_BOOTSTRAP` | `false` | CLI | Auto-bootstrap on first run |

## See also

- [AUTHORIZATION.md](AUTHORIZATION.md) — once authenticated, how grants,
  roles, and access checks work
- [PACKAGES.md `pkg/core`](PACKAGES.md#pkgcore) — `AuthSource` interface
  and the three shipped implementations
- [CLI.md](CLI.md) — full `lkctl` command reference

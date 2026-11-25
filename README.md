# Lakekeeper Go Client

[![Go Report Card](https://goreportcard.com/badge/github.com/lakekeeper/go-lakekeeper)](https://goreportcard.com/report/github.com/lakekeeper/go-lakekeeper)
[![Go Reference](https://pkg.go.dev/badge/github.com/lakekeeper/go-lakekeeper.svg)](https://pkg.go.dev/github.com/lakekeeper/go-lakekeeper)
[![ci](https://github.com/lakekeeper/go-lakekeeper/actions/workflows/ci.yml/badge.svg)](https://github.com/lakekeeper/go-lakekeeper/actions/workflows/ci.yml)
[![nightly](https://github.com/lakekeeper/go-lakekeeper/actions/workflows/nightly.yml/badge.svg)](https://github.com/lakekeeper/go-lakekeeper/actions/workflows/nightly.yml)

Go SDK and `lkctl` CLI for the [Lakekeeper](https://docs.lakekeeper.io)
Iceberg catalog. The Management API client is generated from Lakekeeper's
OpenAPI spec; everything else — auth, retries, the CLI, ergonomic builders
for storage profiles and permissions — is hand-written.

```mermaid
graph LR
    APP["Your code / lkctl CLI"]
    CLIENT["pkg/client.Client"]
    GEN["pkg/apis/management/v1<br/>(generated)"]
    CAT["Iceberg REST Catalog<br/>(via apache/iceberg-go)"]

    APP --> CLIENT
    CLIENT -.->|embeds| GEN
    CLIENT --> CAT
```

## Documentation

- [Architecture](docs/ARCHITECTURE.md) — component overview, request lifecycle, bootstrap flow
- [Package Reference](docs/PACKAGES.md) — every `pkg/` package, including `pkg/core` auth (`OAuthClientCredentialsAuthSource`, `AccessTokenAuthSource`, `K8sServiceAccountAuthSource`)
- [Authentication](docs/AUTHENTICATION.md) — choose a flow (OAuth2, static token, k8s) with CLI ↔ SDK side-by-side examples
- [Authorization](docs/AUTHORIZATION.md) — the permission model and end-to-end grant/revoke workflows
- [Generated client](docs/GENERATION.md) — what's generated, how to regenerate, why a postprocessor is involved
- [CLI Reference](docs/CLI.md) — `lkctl` command tree, environment variables, examples

## Compatibility

The Management API client is generated from the Lakekeeper and Lakekeeper+
OpenAPI specs. Each client release supports, per edition, the minor of the
spec it was generated from and the minor before it
(`client.SupportedVersions`); Lakekeeper minors can make breaking changes to
the Management API, so the range is by minor. A server outside the range is
unsupported: the SDK still sends every call, and `lkctl` warns once per run
and carries on; see [docs/CLI.md](docs/CLI.md#supported-server-versions).

<!-- BEGIN compatibility-matrix -->
| Client version | Lakekeeper            | Lakekeeper+           |
| -------------- | --------------------- | --------------------- |
| `0.2.x`        | `0.14.x` and `0.15.x` | `0.14.x` and `0.15.x` |
| `0.1.x`        | `v0.12.0` minimum     | not supported         |
<!-- END compatibility-matrix -->

The range comes from the spec releases in
[`api/openapi/spec-releases.yaml`](api/openapi/spec-releases.yaml). CI runs
the integration and end-to-end suites against the Lakekeeper spec release
(the `lakekeeper_version` matrix of
[`.github/workflows/ci.yml`](.github/workflows/ci.yml)) and Lakekeeper
`latest-main`, and, on `main`, the Lakekeeper+ suites against the Lakekeeper+
spec release; `make test-e2e-versions` checks `lkctl` locally against the oldest
supported Lakekeeper minor and the minor before it.

## Quick start — CLI

Pre-built binaries on the [Releases page](https://github.com/lakekeeper/go-lakekeeper/releases/latest),
or via Docker:

```sh
docker run --rm quay.io/lakekeeper/lkctl version
```

Authenticate via flags or environment variables (`.env` files are loaded
automatically):

```sh
export LAKEKEEPER_BASE_URL=http://localhost:8181
export LAKEKEEPER_TOKEN_URL=http://localhost:30080/realms/iceberg/protocol/openid-connect/token
export LAKEKEEPER_CLIENT_ID=<your-client-id>
export LAKEKEEPER_CLIENT_SECRET=<your-client-secret>
export LAKEKEEPER_SCOPE=lakekeeper

lkctl server bootstrap --accept-terms-of-use --as-operator
PROJECT_ID=$(lkctl project create new-project -o json | jq -r '."project-id"')
lkctl role create "new-role" --project $PROJECT_ID --description "A new role"
```

See [docs/CLI.md](docs/CLI.md) for the full command reference.

## Quick start — Go SDK

```sh
go get github.com/lakekeeper/go-lakekeeper
```

Requires Go 1.26+.

```go
package main

import (
    "context"
    "log"

    "golang.org/x/oauth2/clientcredentials"

    managementv1 "github.com/lakekeeper/go-lakekeeper/pkg/apis/management/v1"
    "github.com/lakekeeper/go-lakekeeper/pkg/client"
    "github.com/lakekeeper/go-lakekeeper/pkg/core"
)

func main() {
    ctx := context.Background()

    cfg := &clientcredentials.Config{
        ClientID:     "<your-client-id>",
        ClientSecret: "<your-client-secret>",
        TokenURL:     "http://localhost:30080/realms/iceberg/protocol/openid-connect/token",
        Scopes:       []string{"lakekeeper"},
    }

    as := &core.OAuthClientCredentialsAuthSource{TokenSource: cfg.TokenSource(ctx)}
    c, err := client.NewWithAuthSource(ctx, "http://localhost:8181", as,
        client.WithInitialBootstrap(true, true, core.Ptr(managementv1.USERTYPE_APPLICATION)),
    )
    if err != nil {
        log.Fatalf("create client: %v", err)
    }

    info, err := c.Server.Info(ctx)
    if err != nil {
        log.Fatalf("server info: %v", err)
    }
    log.Printf("connected to lakekeeper %s", info.GetVersion())
}
```

Resource façades on `*client.Client` (`c.Server`, `c.Projects`,
`c.Warehouses`, `c.Roles`, `c.Users`, `c.Permissions`) cover the common
operations in one call each. The generated services stay exposed as fields
(`c.ServerAPI`, `c.ProjectAPI`, `c.WarehouseAPI`, `c.TasksAPI`, etc.) for
everything else. See
[docs/PACKAGES.md](docs/PACKAGES.md) for the full surface and
[docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) for how requests flow
end-to-end.

### Catalog access

```go
catalog, err := c.CatalogV1(ctx, projectID, warehouseName)
if err != nil {
    log.Fatalf("get catalog: %v", err)
}
// catalog is *apache/iceberg-go/catalog/rest.Catalog
```

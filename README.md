# Lakekeeper Go Client

[![Go Report Card](https://goreportcard.com/badge/github.com/lakekeeper/go-lakekeeper)](https://goreportcard.com/report/github.com/lakekeeper/go-lakekeeper)
[![GoDoc](https://godoc.org/github.com/lakekeeper/go-lakekeeper?status.svg)](https://godoc.org/github.com/lakekeeper/go-lakekeeper)
[![test](https://github.com/lakekeeper/go-lakekeeper/actions/workflows/test.yml/badge.svg)](https://github.com/lakekeeper/go-lakekeeper/actions/workflows/test.yml)
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
- [Package Reference](docs/PACKAGES.md) — every `pkg/` package, including `pkg/core` auth (`OAuthTokenSource`, `AccessTokenAuthSource`, `K8sServiceAccountAuthSource`)
- [Generated client](docs/GENERATION.md) — what's generated, how to regenerate, why a YAML preprocessor is involved
- [CLI Reference](docs/CLI.md) — `lkctl` command tree, environment variables, examples

## Quick start — CLI

Pre-built binaries on the [Releases page](https://github.com/lakekeeper/go-lakekeeper/releases/latest),
or via Docker:

```sh
docker run --rm quay.io/lakekeeper/lkctl version
```

Authenticate via flags or environment variables (`.env` files are loaded
automatically):

```sh
export LAKEKEEPER_SERVER=http://localhost:8181
export LAKEKEEPER_AUTH_URL=http://localhost:30080/realms/iceberg/protocol/openid-connect/token
export LAKEKEEPER_CLIENT_ID=<your-client-id>
export LAKEKEEPER_CLIENT_SECRET=<your-client-secret>
export LAKEKEEPER_SCOPE=lakekeeper

lkctl server bootstrap --accept-terms-of-use --as-operator
PROJECT_ID=$(lkctl project create new-project | jq -r .)
lkctl role create "new-role" --project $PROJECT_ID --description "A new role"
```

See [docs/CLI.md](docs/CLI.md) for the full command reference.

## Quick start — Go SDK

```sh
go get github.com/lakekeeper/go-lakekeeper
```

Requires Go 1.24+.

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

    as := &core.OAuthTokenSource{TokenSource: cfg.TokenSource(ctx)}
    c, err := client.NewWithAuthSource(ctx, "http://localhost:8181", as,
        client.WithInitialBootstrap(true, true, core.Ptr(managementv1.USERTYPE_APPLICATION)),
    )
    if err != nil {
        log.Fatalf("create client: %v", err)
    }

    info, _, err := c.ServerAPI.GetServerInfo(ctx).Execute()
    if err != nil {
        log.Fatalf("server info: %v", err)
    }
    log.Printf("connected to lakekeeper %s", info.Version)
}
```

The generated services are exposed as fields on `*client.Client`:
`c.ServerAPI`, `c.ProjectAPI`, `c.WarehouseAPI`, `c.RoleAPI`, `c.UserAPI`,
`c.PermissionsOpenfgaAPI`, etc. See
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

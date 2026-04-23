# Architecture

This document describes the high-level architecture of `go-lakekeeper`: a Go client SDK and CLI (`lkctl`) for the [Lakekeeper](https://docs.lakekeeper.io) Management and Iceberg REST Catalog APIs.

## Component Overview

```mermaid
graph TD
    subgraph "Your code / lkctl CLI"
        APP["Application code\nor lkctl commands"]
    end

    subgraph "pkg/client"
        CLIENT["client.Client\nNewAuthSourceClient()"]
    end

    subgraph "pkg/core"
        CORE["AuthSource\nHTTP helpers\nAPIError"]
    end

    subgraph "pkg/apis/management/v1"
        MGMT["ProjectService\nWarehouseService\nRoleService\nUserService\nServerService"]
        PERM["permission/\nPermissionService"]
        PROFILE["storage/profile\nstorage/credential"]
    end

    subgraph "External"
        ICEBERG["apache/iceberg-go\nrest.Catalog"]
        LKAPI["Lakekeeper\nManagement API\n/management/v1/…"]
        LKCAT["Lakekeeper\nIceberg REST Catalog\n/catalog/…"]
    end

    APP --> CLIENT
    CLIENT --> CORE
    CLIENT --> MGMT
    MGMT --> PERM
    MGMT --> PROFILE
    CLIENT --> ICEBERG
    CORE --> LKAPI
    ICEBERG --> LKCAT
```

The `client.Client` is the single entry point. It multiplexes calls to:

- **Management API** — via `ProjectV1()`, `WarehouseV1(projectID)`, `RoleV1(projectID)`, `UserV1()`, `ServerV1()`, `PermissionV1()`. All requests go through `pkg/core` for auth token injection and HTTP retry.
- **Iceberg REST Catalog** — via `CatalogV1(ctx, projectID, warehouse)`, which delegates to the upstream `apache/iceberg-go` REST client with the same auth token.

## Server-Side Context

`go-lakekeeper` is a **client** — it does not run any server-side components. For reference, the Lakekeeper server itself depends on:

```mermaid
graph LR
    LK["Lakekeeper Server"]
    PG["PostgreSQL\n(metadata store)"]
    OPENFGA["OpenFGA\n(authorization)"]
    VAULT["Vault / KMS\n(optional, secret storage)"]
    S3["Object Storage\n(S3 / GCS / ADLS)"]

    LK --> PG
    LK --> OPENFGA
    LK --> VAULT
    LK -.->|"warehouse data\n(via credentials)"| S3
```

The integration-test stack (`make test-integration`) brings up Lakekeeper + **Keycloak** (OIDC IdP) + **MinIO** (S3-compatible storage) + **OpenFGA** via `docker-compose`, which is the canonical reference environment for this SDK.

## Request Lifecycle

Every SDK call follows the same path from the service layer through the client to the wire.

```mermaid
sequenceDiagram
    participant Caller
    participant Service as ProjectService<br/>(or any *Service)
    participant Client as client.Client
    participant Auth as AuthSource
    participant HTTP as retryablehttp.Client
    participant API as Lakekeeper API

    Caller->>Service: Get(ctx, id, opts...)
    Service->>Client: NewRequest(ctx, GET, "/project", nil, opts)
    Note over Client: Appends /management/v1 base path<br/>Encodes query params via go-querystring
    Client-->>Service: *retryablehttp.Request
    Service->>Client: Do(req, &result)
    Client->>Auth: Init(ctx) [sync.Once]
    Client->>Auth: Header(ctx)
    Auth-->>Client: "Authorization", "Bearer <token>"
    Client->>HTTP: client.Do(req)
    Note over HTTP: Retries on 429 and 5xx<br/>Linear jitter backoff<br/>100ms–400ms, max 5 retries
    HTTP->>API: GET /management/v1/project
    API-->>HTTP: 200 OK + JSON body
    HTTP-->>Client: *http.Response
    Note over Client: CheckResponse() maps non-2xx<br/>to *core.APIError
    Client->>Client: json.Decode(body, &result)
    Client-->>Service: resp, nil
    Service-->>Caller: *Project, *http.Response, nil
```

Key points:

- `NewRequest` always prepends `/management/v1` to the path. The base URL is set once at client construction and never changes per-request.
- Auth headers are injected lazily in `Do()`: `Init` is called exactly once (via `sync.Once`), then `Header` is called on every request.
- The `retryablehttp` layer transparently retries `429 Too Many Requests` and any `>= 500` status with linear-jitter backoff. Retries can be disabled with `WithoutRetries()`.
- A non-2xx response is converted to `*core.APIError` by `CheckResponse`, but the raw `*http.Response` is still returned so callers can inspect status codes or headers.

## Bootstrap Flow

When `WithInitialBootstrapV1Enabled(true, isOperator, userType)` is passed to `NewAuthSourceClient`, the client calls `ServerV1().Info()` during construction and, if the server reports `bootstrapped: false`, calls `ServerV1().Bootstrap()` automatically. This happens exactly once per client instance via `sync.Once`.

## Project Scoping

Resources that belong to a project (warehouses, roles) are accessed via project-scoped service constructors:

```go
client.RoleV1(projectID).Get(ctx, roleID)
client.WarehouseV1(projectID).List(ctx, opts)
```

Internally these send the `x-project-id` header on each request (see `managementv1.WithProject`). Project-independent resources (`ServerV1()`, `UserV1()`, `ProjectV1()`) do not set this header.

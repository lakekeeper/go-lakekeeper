# lkctl — CLI Reference

`lkctl` is the command-line interface for Lakekeeper. It is currently in
**preview**; the command surface may change before a stable release.

## Installation

Download a pre-built binary from the
[Releases page](https://github.com/lakekeeper/go-lakekeeper/releases/latest),
or run via Docker:

```sh
docker run --rm quay.io/lakekeeper/lkctl version
```

To build from source:

```sh
make build          # produces dist/lkctl
# make build runs: vet → test → go build
```

Other useful make targets:

| Target | Description |
|---|---|
| `make build` | Vet, test, then build `dist/lkctl` |
| `make validate` | `go vet` + `golangci-lint` (no build) |
| `make test` | Unit tests under `./pkg/…` with coverage |
| `make test-integration` | Spin up Lakekeeper + Keycloak + MinIO + OpenFGA via docker-compose and run integration tests |
| `make test-e2e-compose` | E2E suite for `lkctl` itself, against the docker-compose stack (build tag `e2e_cli`). See [`e2e/`](../e2e/) |
| `make test-e2e-kind` | E2E suite for `lkctl --auth-mode k8s` against a real kind cluster. See [`e2e/kind/`](../e2e/kind/) |
| `make generate` | Regenerate the Management API client from the OpenAPI spec — see [GENERATION.md](GENERATION.md) |
| `make snapshot` | goreleaser snapshot (multi-arch, no publish) |
| `make clean` | Tear down compose stack and remove build artefacts |

---

## Global flags & environment variables

Every `lkctl` command accepts these persistent flags. Environment
variables or a `.env` file in the working directory replace any flag.

| Flag | Env variable | Default | Description |
|---|---|---|---|
| `--base-url` | `LAKEKEEPER_BASE_URL` | `http://localhost:8181` | Lakekeeper base URL |
| `--auth-mode` | `LAKEKEEPER_AUTH_MODE` | `oauth2` | Auth mode: `oauth2`, `token`, or `k8s` |
| `--token-url` | `LAKEKEEPER_TOKEN_URL` | _(none)_ | OAuth2 token endpoint (`auth-mode=oauth2`) |
| `--client-id` | `LAKEKEEPER_CLIENT_ID` | _(none)_ | OAuth2 `client_id` (`auth-mode=oauth2`) |
| `--client-secret` | `LAKEKEEPER_CLIENT_SECRET` | _(none)_ | OAuth2 `client_secret` (`auth-mode=oauth2`) |
| `--scopes` | `LAKEKEEPER_SCOPE` | `lakekeeper` | Space-separated OAuth2 scopes (`auth-mode=oauth2`) |
| `--access-token` | `LAKEKEEPER_ACCESS_TOKEN` | _(none)_ | Static bearer token (`auth-mode=token`) |
| `--k8s-token-path` | `LAKEKEEPER_K8S_TOKEN_PATH` | `/var/run/secrets/kubernetes.io/serviceaccount/token` | Service-account token file (`auth-mode=k8s`) |
| `--bootstrap` | `LAKEKEEPER_BOOTSTRAP` | `false` | Auto-bootstrap server on startup |
| `--debug` | _(none)_ | `false` | Enable debug logging |

### Auth mode examples

For a guide-level walkthrough that pairs each flow with the equivalent
Go SDK setup — and a reference table of every auth-related environment
variable — see [AUTHENTICATION.md](AUTHENTICATION.md).

**OAuth2 client credentials** (default — used by the integration stack):

```sh
export LAKEKEEPER_BASE_URL=http://localhost:8181
export LAKEKEEPER_TOKEN_URL=http://localhost:30080/realms/iceberg/protocol/openid-connect/token
export LAKEKEEPER_CLIENT_ID=<your-client-id>
export LAKEKEEPER_CLIENT_SECRET=<your-client-secret>
export LAKEKEEPER_SCOPE=lakekeeper
```

**Static access token** — when you already hold a bearer token (CI, short-lived
credentials, scripted impersonation):

```sh
export LAKEKEEPER_BASE_URL=http://localhost:8181
export LAKEKEEPER_AUTH_MODE=token
export LAKEKEEPER_ACCESS_TOKEN=<your-bearer-token>
```

**Kubernetes service account** — when running `lkctl` from inside a pod whose
projected service-account token is accepted by the Lakekeeper server. The
default path is the standard projected-volume mount; override via
`--k8s-token-path` only if your pod uses a non-default location.

```sh
export LAKEKEEPER_BASE_URL=http://lakekeeper.lakekeeper.svc:8181
export LAKEKEEPER_AUTH_MODE=k8s
```

---

## Command tree

Top-level commands (and their aliases):

| Command | Alias | Purpose |
|---|---|---|
| `server` | `srv` | Server info, bootstrap, and server-level permissions |
| `project` | `proj` | Manage projects and their permissions |
| `warehouse` | `wh` | Manage warehouses (project-scoped) and their permissions |
| `role` | _(none)_ | Manage roles (project-scoped) and their permissions |
| `user` | _(none)_ | Manage users |
| `whoami` | _(none)_ | Print the authenticated principal |
| `version` | _(none)_ | Print version, commit, build date |
| `catalog` | _(none)_ | Placeholder — not yet implemented; use `client.CatalogV1` from the SDK |

The grant / revoke / access / assignments verbs follow the same shape
across `server`, `project`, `warehouse`, and `role`. Read this section
once for `project` and the others should feel familiar. For the
underlying authorization model, the assignment-value reference, and an
end-to-end CLI ↔ SDK workflow, see [AUTHORIZATION.md](AUTHORIZATION.md).

### `lkctl server`

```sh
# Show server info (version, auth backend, bootstrap state)
lkctl server info

# Bootstrap the server (first-time setup)
lkctl server bootstrap --accept-terms-of-use --as-operator

# Or auto-bootstrap via the global flag (bootstraps if needed, then runs the command)
lkctl project list --bootstrap

# Server-level permissions (no resource ID — server is implicit)
lkctl server access
lkctl server assignments
lkctl server grant   --users <USER-ID> --assignments admin
lkctl server revoke  --users <USER-ID> --assignments admin
```

### `lkctl project`

```sh
# List all projects
lkctl project list

# Get a project by ID (sent as the x-project-id header)
lkctl project get <PROJECT-ID>

# Create a project
PROJECT_ID=$(lkctl project create my-project -o json | jq -r '."project-id"')

# Rename a project
lkctl project rename <PROJECT-ID> new-name

# Delete a project
lkctl project delete <PROJECT-ID>

# Permissions — PROJECT-ID is optional and defaults to the bootstrap project
lkctl project access      [PROJECT-ID]                 # show allowed actions for the current user
lkctl project access      [PROJECT-ID] --user <ID>     # or for a specific user
lkctl project assignments [PROJECT-ID]                 # list assignments
lkctl project grant       [PROJECT-ID] --users <U> --assignments project_admin
lkctl project revoke      [PROJECT-ID] --roles <R> --assignments select
```

`grant` and `revoke` accept `--users`, `--roles`, and `--assignments` as
repeatable / comma-separated string slices. `--user` / `--role` are
mutually exclusive on the singular `access` command.

### `lkctl warehouse`

Warehouses belong to a project. `list` and `create` take the project
UUID from `--project` / `-p`. When omitted, no project is sent and the
server's default project is used (the nil UUID unless the operator
configured another; an error if the operator disabled it). The other
subcommands address a warehouse by its id and ignore `-p`.

```sh
# List warehouses in a project (filter with --status active|inactive, repeatable)
lkctl warehouse list --project <PROJECT-ID>

# Get a specific warehouse
lkctl warehouse get <WAREHOUSE-ID>

# Create a warehouse from a JSON config file
lkctl warehouse create "my-warehouse" -f warehouse-config.json --project <PROJECT-ID>

# Create from stdin
cat warehouse-config.json | lkctl warehouse create "my-warehouse" -f - --project <PROJECT-ID>

# Lifecycle
lkctl warehouse rename     <WAREHOUSE-ID> "new-name"
lkctl warehouse activate   <WAREHOUSE-ID>
lkctl warehouse deactivate <WAREHOUSE-ID>
lkctl warehouse delete     <WAREHOUSE-ID>

# Protection (boolean, blocks delete while true)
lkctl warehouse set-protection <WAREHOUSE-ID> --protected=true

# Statistics (paginated; --page-size, default 100, and --page-token)
lkctl warehouse statistics <WAREHOUSE-ID>

# Permissions
lkctl warehouse access      <WAREHOUSE-ID>
lkctl warehouse assignments <WAREHOUSE-ID>
lkctl warehouse grant       <WAREHOUSE-ID> --users <U> --assignments ownership
lkctl warehouse revoke      <WAREHOUSE-ID> --roles <R> --assignments describe
```

The `-f` flag for `create` accepts a JSON file that maps to
`managementv1.CreateWarehouseRequest`; its `warehouse-name` must match the
name argument. A `project-id` in the config is sent like `-p`, as the
`x-project-id` header; when both are given they must match. Example config:

```json
{
  "warehouse-name": "my-warehouse",
  "storage-profile": { ... },
  "storage-credential": { ... },
  "delete-profile": { "type": "hard" }
}
```

### `lkctl role`

Roles are project-scoped. Use `--project` / `-p` to specify the project
UUID. When omitted, no project is sent and the server's default project is
used, as for `lkctl warehouse`. The permission subcommands (`access`,
`assignments`, `grant`, `revoke`) ignore `-p`: their endpoints address the
role by id alone.

```sh
# List roles (one page; --limit, --token)
lkctl role list --project <PROJECT-ID>

# Find the role named exactly "data-readers" anywhere in the project
lkctl role list --name data-readers --project <PROJECT-ID>

# Get / create / update / delete
lkctl role get    <ROLE-ID>
lkctl role create "New Role" --description "Optional description"
lkctl role update <ROLE-ID> "New Name" --description "Updated description"
lkctl role delete <ROLE-ID>

# Permissions
lkctl role access      <ROLE-ID>
lkctl role assignments <ROLE-ID>
lkctl role grant       <ROLE-ID> --users <U> --assignments assignee   # alias: role assign
lkctl role revoke      <ROLE-ID> --roles <R> --assignments ownership  # alias: role unassign
```

`role list --name` returns the role whose name matches exactly, from the
whole project rather than the current page; it cannot be combined with
`--token`. It asks the server's role search first. That search returns only
the 10 closest names (and only looks at the first 64 characters), so when it
finds no exact match, `lkctl` pages through every role before printing none.

### `lkctl user`

```sh
# List users; --name filters on the server, across all pages
lkctl user list
lkctl user list --name peter

# Get a user
lkctl user get <USER-ID>

# Create / provision a user — USERID NAME USERTYPE
lkctl user create oidc~d223d88c-85b6-4859-b5c5-27f3825e47f6 "Peter Cold" human
lkctl user create kubernetes~... "Service Account" application --email me@example.com --update

# Delete a user
lkctl user delete <USER-ID>
```

`USERTYPE` is one of `human` or `application`. Use `--update` to upsert
when the user already exists.

### `lkctl version` / `lkctl whoami`

```sh
lkctl version       # version, commit, date, tree state
lkctl whoami        # the authenticated principal, including whether it is an instance admin
```

---

## Pagination

`role list` and `user list` accept `--limit` (default 100), `--token`
(page token), and `--name` (name filter; for `role list` an exact match
across the whole project, see [`lkctl role`](#lkctl-role)):

```sh
lkctl user list --limit 10
lkctl user list --limit 10 --token <next-page-token>
```

`project list` is not paginated: the server returns every project visible
to the caller in one response.

`warehouse statistics` is paginated separately with `--page-size` and
`--page-token`.

---

## Output format

Most commands accept `--output` / `-o`, with `text` (default) and `json`
supported. A few list commands additionally support `wide`.

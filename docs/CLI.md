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
# make build runs: mod → fmt → vet → test → go build
```

Other useful make targets:

| Target | Description |
|---|---|
| `make build` | Lint, test, then build `dist/lkctl` |
| `make validate` | `go vet` + `golangci-lint` (no build) |
| `make test` | Unit tests under `./pkg/…` with coverage |
| `make test-integration` | Spin up Lakekeeper + Keycloak + MinIO + OpenFGA via docker-compose and run integration tests |
| `make generate` | Regenerate the Management API client from the OpenAPI spec — see [GENERATION.md](GENERATION.md) |
| `make snapshot` | goreleaser snapshot (multi-arch, no publish) |
| `make clean` | Tear down compose stack and remove build artefacts |

---

## Global flags & environment variables

Every `lkctl` command accepts these persistent flags. Environment
variables or a `.env` file in the working directory replace any flag.

| Flag | Env variable | Default | Description |
|---|---|---|---|
| `--server` | `LAKEKEEPER_SERVER` | `http://localhost:8181` | Lakekeeper base URL |
| `--auth-url` | `LAKEKEEPER_AUTH_URL` | _(none)_ | OAuth2 token endpoint |
| `--client-id` | `LAKEKEEPER_CLIENT_ID` | _(none)_ | OAuth2 `client_id` |
| `--client-secret` | `LAKEKEEPER_CLIENT_SECRET` | _(none)_ | OAuth2 `client_secret` |
| `--scopes` | `LAKEKEEPER_SCOPE` | `lakekeeper` | Space-separated OAuth2 scopes |
| `--bootstrap` | `LAKEKEEPER_BOOTSTRAP` | `false` | Auto-bootstrap server on startup |
| `--debug` | _(none)_ | `false` | Enable debug logging |

Example with environment variables:

```sh
export LAKEKEEPER_SERVER=http://localhost:8181
export LAKEKEEPER_AUTH_URL=http://localhost:30080/realms/iceberg/protocol/openid-connect/token
export LAKEKEEPER_CLIENT_ID=<your-client-id>
export LAKEKEEPER_CLIENT_SECRET=<your-client-secret>
export LAKEKEEPER_SCOPE=lakekeeper
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
once for `project` and the others should feel familiar.

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

# Get a project by ID
lkctl project get <PROJECT-ID>

# Create a project
PROJECT_ID=$(lkctl project create my-project | jq -r .)

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

Warehouses are project-scoped. Use `--project` / `-p` to specify the
project UUID.

```sh
# List warehouses in a project (filter with --status active|inactive, repeatable)
lkctl warehouse list --project <PROJECT-ID>

# Get a specific warehouse
lkctl warehouse get <WAREHOUSE-ID> --project <PROJECT-ID>

# Create a warehouse from a JSON config file
lkctl warehouse create "my-warehouse" -f warehouse-config.json --project <PROJECT-ID>

# Create from stdin
cat warehouse-config.json | lkctl warehouse create "my-warehouse" -f - --project <PROJECT-ID>

# Lifecycle
lkctl warehouse rename     <WAREHOUSE-ID> "new-name" --project <PROJECT-ID>
lkctl warehouse activate   <WAREHOUSE-ID>            --project <PROJECT-ID>
lkctl warehouse deactivate <WAREHOUSE-ID>            --project <PROJECT-ID>
lkctl warehouse delete     <WAREHOUSE-ID>            --project <PROJECT-ID>

# Protection (boolean, blocks delete while true)
lkctl warehouse set-protection <WAREHOUSE-ID> --protected=true  --project <PROJECT-ID>

# Statistics (paginated; --page-size, --page-token)
lkctl warehouse statistics <WAREHOUSE-ID> --project <PROJECT-ID>

# Permissions
lkctl warehouse access      <WAREHOUSE-ID>
lkctl warehouse assignments <WAREHOUSE-ID>
lkctl warehouse grant       <WAREHOUSE-ID> --users <U> --assignments ownership
lkctl warehouse revoke      <WAREHOUSE-ID> --roles <R> --assignments describe
```

The `-f` flag for `create` accepts a JSON file that maps to
`managementv1.CreateWarehouseRequest`. Example config:

```json
{
  "storage-profile": { ... },
  "storage-credential": { ... },
  "delete-profile": { "type": "hard" }
}
```

### `lkctl role`

Roles are project-scoped. Use `--project` / `-p`.

```sh
# List roles
lkctl role list --project <PROJECT-ID>

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

### `lkctl user`

```sh
# List users
lkctl user list

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
lkctl whoami        # the identity of the authenticated principal
```

---

## Pagination

`list` subcommands accept `--limit` (default 100), `--token` (page token),
and `--name` (server-side name filter):

```sh
lkctl project list --limit 10
lkctl project list --limit 10 --token <next-page-token>
```

`warehouse statistics` is paginated separately with `--page-size` and
`--page-token`.

---

## Output format

Most commands accept `--output` / `-o`, with `text` (default) and `json`
supported. A few list commands additionally support `wide`.

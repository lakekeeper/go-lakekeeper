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
| `make cover` | `make test`, then `coverage.html` and a per-package `coverage.md`, generated SDK excluded |
| `make test-integration` | Spin up Lakekeeper + Keycloak + MinIO + OpenFGA via docker-compose and run integration tests |
| `make test-e2e-compose` | E2E suite for `lkctl` itself, against the docker-compose stack (build tag `e2e_cli`). See [`e2e/`](../e2e/) |
| `make test-e2e-kind` | E2E suite for `lkctl --auth-mode k8s` against a real kind cluster. See [`e2e/kind/`](../e2e/kind/) |
| `make test-plus` | Lakekeeper+ E2E and integration tests (build tags `e2e_plus`, `integration_plus`) against the public Lakekeeper+ image, with Cedar and with OpenFGA. `LAKEKEEPER__LICENSE__KEY` is passed through when set; optional for local runs, CI uses one. See [`e2e/plus/run.sh`](../e2e/plus/run.sh) |
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
| `cedar` | _(none)_ | Cedar policies (Lakekeeper+, Cedar authorization backend) |
| `task-queue` | `tq` | Show and set task-queue configuration, schedule maintenance tasks; some queues need Lakekeeper+ |

The grant / revoke / access / assignments verbs follow the same shape
across `server`, `project`, `warehouse`, and `role`. Read this section
once for `project` and the others should feel familiar. For the
underlying authorization model, the assignment-value reference, and an
end-to-end CLI ↔ SDK workflow, see [AUTHORIZATION.md](AUTHORIZATION.md).

### Lakekeeper+ commands

One `lkctl` binary works against both editions, Lakekeeper and
Lakekeeper+. Commands that need Lakekeeper+ are always listed, and their
help says "(Lakekeeper+)"; `--help` never contacts a server.

Run against a server that lacks what a command needs, the command exits
non-zero with one sentence naming the requirement and what the server is:

```text
Error: get cedar schema: requires Lakekeeper+; server is Lakekeeper 0.15.1
Error: get cedar schema: requires the cedar authorization backend; server is Lakekeeper+ 0.15.1 with the openfga authorization backend
Error: get optimize queue config: requires the optimize task queue; server is Lakekeeper+ 0.15.1 without it
```

`lkctl` reads the server info for this only after the server rejects a
request as unknown (404 or 405); a genuine not-found is still reported as
the server's error. See
[ARCHITECTURE.md](ARCHITECTURE.md#editions-and-capabilities).

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

# Permissions (PROJECT-ID is sent as the x-project-id header)
lkctl project access      [PROJECT-ID]                 # show allowed actions for the current user
lkctl project access      [PROJECT-ID] --user <ID>     # or for a specific user
lkctl project assignments [PROJECT-ID]                 # list assignments
lkctl project grant       PROJECT-ID --users <U> --assignments project_admin
lkctl project revoke      PROJECT-ID --roles <R> --assignments select
```

For `access` and `assignments`, `[PROJECT-ID]` is optional: when omitted,
no project is sent and the server's default project is used, the same as
omitting `-p` on other commands. `grant` and `revoke` change permissions,
so `PROJECT-ID` is required and never falls back to the default project;
to target it, pass its id (the nil UUID
`00000000-0000-0000-0000-000000000000` unless the operator configured
another).
An empty `PROJECT-ID` (`""`, e.g. from an unset variable) is an error, on
these commands and on `get`, `rename`, and `delete`; so is an explicit
empty `-p ""`.

`grant` and `revoke` accept `--users`, `--roles`, and `--assignments` as
repeatable / comma-separated string slices. `--user` / `--role` are
mutually exclusive on the singular `access` command.

### `lkctl warehouse`

Warehouses belong to a project. `list` and `create` take the project
UUID from `--project` / `-p`. When omitted, no project is sent and the
server's default project is used (the nil UUID unless the operator
configured another; an error if the operator disabled it). The other
subcommands address a warehouse by its id and ignore `-p`.

`list` and `get` print each warehouse's storage family in the `STORAGE`
column: the `type` of its storage profile, one of `s3`, `gcs`, `adls`,
`onelake`, `stackit` or `memory` (Lakekeeper+ only), and `unknown` for any
other.

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

# Fuzzy-search users (closest matches only)
lkctl user search peter

# Roles a user is directly assigned to; --transitive adds roles reached
# through role membership. Project-scoped: -p, else the default project
lkctl user roles <USER-ID>
lkctl user roles <USER-ID> --transitive -p <PROJECT-ID>

# Allowed actions on a user (current user's view, or --user / --role)
lkctl user access <USER-ID>
lkctl user access <USER-ID> --role <ROLE-ID>
```

`USERTYPE` is one of `human` or `application`. Use `--update` to upsert
when the user already exists.

`user search` returns only the server's closest matches; for an exhaustive
name filter use `user list --name`. `user roles --transitive` works only when
role assignments are catalog-managed; with OpenFGA the server answers
`501 Not Implemented`.

### `lkctl cedar` (Lakekeeper+)

Needs Lakekeeper+ with the Cedar authorization backend.

```sh
# Print the active Cedar schema: each fragment under a "// <file>" comment
lkctl cedar schema

# The fragments as JSON ({"schema-fragments": {"<file>": "<content>"}})
lkctl cedar schema -o json

# A project's stored policies (-p as elsewhere; default project without it),
# then one by name; -o json includes the policy in Cedar's JSON format
lkctl cedar project-policies list -p <project-id>
lkctl cedar project-policies get readers -o json

# Create, update and delete policies in one transaction, from a file or stdin
lkctl cedar project-policies apply -f policies.json
cat policies.json | lkctl cedar project-policies apply -f -

# The same for a warehouse; the warehouse names its project, so no -p
lkctl cedar warehouse-policies list <warehouse-id>
lkctl cedar warehouse-policies get <warehouse-id> readers
lkctl cedar warehouse-policies apply <warehouse-id> -f policies.json

# Whether break-glass is available to you for a project (project-id query parameter)
lkctl cedar break-glass-status -p <project-id>

# Emergency access, with the reason the server records
lkctl cedar project-policies apply -f fix.json --break-glass "INC-42: locked out"

# Where the server loads its Cedar policies and entities from (files, ConfigMaps)
lkctl cedar policy-sources
lkctl cedar entity-sources

# The server's active policies (EFFECT PRINCIPAL ACTION RESOURCE); -o json
# prints them in Cedar's JSON policy format, conditions included
lkctl cedar server-policies list
lkctl cedar server-policies list --source <source-id> --effect forbid
lkctl cedar server-policies list --page-size 50 --page-token <token>

# The Cedar entities the server derives for a resource, for you or another principal
lkctl cedar resolve-entities server
lkctl cedar resolve-entities project <project-id>
lkctl cedar resolve-entities warehouse <warehouse-id> --user <user-id>
```

`server-policies list` without `--source` includes the predefined
policies; it never includes the policies stored for projects and
warehouses. When more policies follow, text output ends with
`Next page token`. `resolve-entities` text output (`TYPE ID PARENTS`)
leaves out the action entities that every result carries; `-o wide`
shows them, `-o json` adds the attributes. It takes no `-p`: the API
gives the operation no project header, so `project` without an id
leaves the choice of project to the server.

The apply file is the Management API's apply request: `writes` (each a
`name`, optional `description`, and a `policy` in Cedar's JSON policy
format), `deletes`, `replace`, `dry-run` and `if-scope-version`. Text output
lists the changes (`OP NAME ID`) and the scope version after the apply.
`list` ends with the scope version, the project's scope version on a
warehouse, and whether you can change the policies (`Can write`).

`--break-glass REASON` sends the `x-break-glass` header. Only `list` and
`apply` of `project-policies` and `warehouse-policies` have it: those are
the operations that accept the header.

#### Predefined policies

Lakekeeper+ provides predefined Cedar policies that a project or a warehouse
switches on or off; they are never edited.

```sh
# The predefined policies with their enablement (-p as elsewhere)
lkctl cedar project-predefined-policies list -p <project-id>
lkctl cedar project-predefined-policies list -o json --include-policy   # with each rule

# Switch policies in one request; prints the listing as of the write
lkctl cedar project-predefined-policies set --disable predefined-grants-table-select
lkctl cedar project-predefined-policies set --enable predefined-a,predefined-b --break-glass "INC-42"

# Drop every decision: the project follows the provided defaults again
lkctl cedar project-predefined-policies reset

# The same for a warehouse; reset makes it follow its project again
lkctl cedar warehouse-predefined-policies list <warehouse-id>
lkctl cedar warehouse-predefined-policies set <warehouse-id> --disable predefined-grants-table-select
lkctl cedar warehouse-predefined-policies reset <warehouse-id>
```

`list` and `set` print `ID ENABLED DISABLED BY CAN TOGGLE` (`-o wide` adds
`UPDATED AT`, `UPDATED BY` and `DESCRIPTION`), then whether the scope has
decided anything itself (`Curated`) and whether the server evaluates
predefined policies at all. `DISABLED BY` is `project`, `warehouse` or
`new-since-curated`, `(retired)` for a policy the server no longer provides,
and `-` for a policy in force. `list` and `set` take `--break-glass`;
`reset` does not, as its operation does not accept the header.

### `lkctl task-queue`

One subcommand per task queue, each with `get` and `set`; the Lakekeeper+
queues also have `schedule`. The subcommand also accepts the server's queue name (`expire_snapshots`) as an alias.

| Queue | Scope | Needs |
|---|---|---|
| `tabular-purge` | warehouse | |
| `soft-deletion` | warehouse | |
| `task-log-cleanup` | project (`-p`, else the server's default project) | |
| `expire-snapshots` | warehouse | Lakekeeper+ running the queue |
| `optimize` | warehouse | Lakekeeper+ running the queue |
| `remove-orphan-files` | warehouse | Lakekeeper+ running the queue |

`set` sends only the flags given: every other field is left unset and
inherits the server default, and the configuration replaces the stored
one. Each queue-config field is a flag of the same name (see
`lkctl task-queue <queue> set --help`); `--max-seconds-since-last-heartbeat`
applies to every queue. `get` prints every field, `unset` for one that
inherits the default; `-o json` prints the server's response.

```sh
# Expire snapshots older than 7 days; the other fields keep the server defaults
lkctl task-queue expire-snapshots set <warehouse-id> --max-snapshot-age-ms 604800000
lkctl task-queue expire-snapshots get <warehouse-id>

# Keep task logs for 30 days in a project
lkctl task-queue task-log-cleanup set -p <project-id> --cleanup-period P1D --retention-period P30D
```

Lakekeeper 0.15 rejects a `task-log-cleanup set` without `--cleanup-period`.

`schedule <warehouse-id> <table-id>` (Lakekeeper+) schedules a task of the
queue for one table and prints its task id; `-o json` prints the server's
response. The task runs on the next worker poll, or at `--scheduled-for`
(RFC 3339). `optimize schedule` takes `--step` once per step to run
(`compaction`, `rewrite-position-deletes`, `remove-dangling-deletes`,
`rewrite-manifests`); without it the server runs its default steps. A
missing warehouse or table is reported as not found.

```sh
lkctl task-queue expire-snapshots schedule <warehouse-id> <table-id>
lkctl task-queue optimize schedule <warehouse-id> <table-id> --step compaction --step rewrite-manifests
lkctl task-queue remove-orphan-files schedule <warehouse-id> <table-id> --scheduled-for 2030-01-02T03:00:00Z
```

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

`user roles` takes `--limit` and `--token` the same way, without `--name`.

`project list` is not paginated: the server returns every project visible
to the caller in one response.

`warehouse statistics` is paginated separately with `--page-size` and
`--page-token`.

---

## Output format

Most commands accept `--output` / `-o`, with `text` (default) and `json`
supported. A few list commands additionally support `wide`.

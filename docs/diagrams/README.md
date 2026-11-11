# Diagrams

Diagrams of how the go-lakekeeper SDK and `lkctl` are built, ordered from concepts to code. Each one is embedded in the prose doc that explains its topic, and that doc defines the terms the diagram uses. Read them in this order:

1. [System context](../ARCHITECTURE.md#system-context): what the SDK and `lkctl` talk to, and what the Lakekeeper server runs on its side
2. [Domain model](../AUTHORIZATION.md#the-lakekeeper-authorization-model): the server, projects, warehouses, users and roles, and how an assignment gives a principal a relation on a resource
3. [Request lifecycle](../ARCHITECTURE.md#request-lifecycle): one façade call through the generated service, the retrying HTTP client and `AuthRoundTripper` to the API, and back as a typed result or one `APIError`

The [drift table](#drift-table) lists the source each diagram shows.

## Reading and updating the diagrams

Each diagram has four files here, named after its slug:

- `<slug>-light.svg` and `<slug>-dark.svg`: the images embedded in the prose docs, matching your GitHub theme
- `<slug>.html`: an explorable version with zoom, focus, search and themes. GitHub shows its source, so download the file or clone the repository and open it locally in a browser.
- `<slug>.json`: the diagram's source. Review diagram changes in this file.

Reading and reviewing the diagrams needs nothing beyond a browser.

When a change touches source listed in the [drift table](#drift-table), update that diagram in two steps:

1. **Edit `<slug>.json`.** This part needs judgement, from a person or a coding agent with the [Archify](https://github.com/tt-a1i/archify) skill: decide what the diagram shows, label it with the terms the embedding doc defines, and cite the source file and lines behind each element. It needs only an editor.
2. **Render it:** `make diagrams-render DIAGRAMS=<slug>`, then set the diagram's date in the drift table to today. This part is mechanical. It needs Archify, Chrome and Node; set `ARCHIFY` to Archify's `bin/archify.mjs` if `archify` is not on your `PATH`. The command checks every citation against the current commit (`HEAD`), so commit your code changes first. It writes `HEAD` into `meta.repository.revision`; don't edit that by hand.

Without `DIAGRAMS`, it renders every diagram. Renders go to `.archify/` (gitignored) and are then copied here.

If you rendered with the Archify skill instead, into `.archify/<type>-<slug>-<YYYYMMDD-HHMMSS>/`, run `make diagrams` to copy the newest render of each diagram here; it does not run Archify. A failed render leaves an incomplete folder that blocks the copy for that slug: delete it and render again.

A new diagram also needs a `<picture>` embed and a caption linking its HTML in its prose doc, the terms it uses defined there, and a row in the reading order and the drift table.

## Drift table

Each diagram, the source it shows, and the date it was last checked against the code. The exact commit is in each JSON's `meta.repository.revision`.

| Diagram | Source it shows | Last checked |
| --- | --- | --- |
| [System context](../ARCHITECTURE.md#system-context) | `cmd/lkctl/commands/client.go`, `.golangci.yml`, `pkg/lakekeeper/lakekeeper.go`, `pkg/client/client.go`, `pkg/core/transport.go`, `pkg/core/auth.go`, `pkg/apis/management/v1/client.go`, `pkg/storage/profile/profile.go`, `go.mod`, `docker-compose.yml` | 2026-11-11 |
| [Domain model](../AUTHORIZATION.md#the-lakekeeper-authorization-model) | `pkg/client/permissions.go`, `pkg/client/server.go`, `pkg/client/projects.go`, `pkg/client/warehouses.go`, `pkg/client/users.go`, `pkg/client/roles.go`, `pkg/permissions/assignment.go`, the generated models of these resources and their relations in `pkg/apis/management/v1/` | 2026-11-11 |
| [Request lifecycle](../ARCHITECTURE.md#request-lifecycle) | `pkg/client/projects.go`, `pkg/client/errors.go`, `pkg/client/client.go`, `pkg/client/client_options.go`, `pkg/core/transport.go`, `pkg/apis/management/v1/client.go`, `pkg/apis/management/v1/api_project.go`, `api/openapi/management-open-api.yaml` | 2026-11-11 |

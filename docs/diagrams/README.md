# Diagrams

Diagrams of how the go-lakekeeper SDK and `lkctl` are built, ordered from concepts to code. Each one is embedded in the prose doc that explains its topic, and that doc defines the terms the diagram uses. Read them in this order:

1. [System context](../ARCHITECTURE.md#system-context): what the SDK and `lkctl` talk to, and what the Lakekeeper server runs on its side
2. [Domain model](../AUTHORIZATION.md#the-lakekeeper-authorization-model): the server, projects, warehouses, users and roles, and how an assignment gives a principal a relation on a resource
3. [Request lifecycle](../ARCHITECTURE.md#request-lifecycle): one façade call through the generated service, the retrying HTTP client and `AuthRoundTripper` to the API, and back as a typed result or one `APIError`
4. [Project scoping](../ARCHITECTURE.md#project-scoping): what a project-scoped façade call sends with `WithProject(id)`, with no option, with `WithProject("")`, on a destructive call that requires a project, and with the deprecated body field, and when the server picks the default project
5. [Authentication](../AUTHENTICATION.md#overview): how `lkctl`'s auth mode, the umbrella constructors or your own code pick one of the three auth sources, which `NewWithAuthSource` validates once with `Init`; how `AuthRoundTripper` fetches the header on every request; and the optional bootstrap at construction
6. [Package layers](../PACKAGES.md#package-layers): which package new code belongs in, how application code and `lkctl` reach the Management API, that a `forbidigo` rule keeps `lkctl` on the resource façades and off the escape hatch, and that an operation without a façade gets a façade method first
7. [Generation pipeline](../GENERATION.md#how-to-regenerate): how `make generate` turns the OpenAPI spec into the generated SDK through openapi-generator and the postprocessor, that generated code changes only through the spec, the postprocessor and the generator config, never by hand, and that the generator config lists only deviations from the defaults
8. [Development workflow](../../CONTRIBUTING.md#development-workflow): the steps from edit to release, which tests need the compose stack or a kind cluster, and how a Conventional Commit pull request reaches release-please

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

If you rendered with the Archify skill instead, into `.archify/<type>-<slug>-<YYYYMMDD-HHMMSS>/`, first export its SVGs with `node scripts/export-diagram-svg.mjs <render-folder>/<slug>.html`, then run `make diagrams` to copy the newest render of each diagram here; it does not run Archify. A failed render leaves an incomplete folder that blocks the copy for that slug: delete it and render again.

A new diagram also needs a `<picture>` embed and a caption linking its HTML in its prose doc, the terms it uses defined there, and a row in the reading order and the drift table.

## Drift table

Each diagram, the source it shows, and the date it was last checked against the code. The exact commit is in each JSON's `meta.repository.revision`.

| Diagram | Source it shows | Last checked |
| --- | --- | --- |
| [System context](../ARCHITECTURE.md#system-context) | `cmd/lkctl/commands/client.go`, `.golangci.yml`, `pkg/lakekeeper/lakekeeper.go`, `pkg/client/client.go`, `pkg/core/transport.go`, `pkg/core/auth.go`, `pkg/apis/management/v1/client.go`, `pkg/storage/profile/profile.go`, `go.mod`, `docker-compose.yml` | 2026-11-11 |
| [Domain model](../AUTHORIZATION.md#the-lakekeeper-authorization-model) | `pkg/client/permissions.go`, `pkg/client/server.go`, `pkg/client/projects.go`, `pkg/client/warehouses.go`, `pkg/client/users.go`, `pkg/client/roles.go`, `pkg/permissions/assignment.go`, the generated models of these resources and their relations in `pkg/apis/management/v1/` | 2026-11-11 |
| [Request lifecycle](../ARCHITECTURE.md#request-lifecycle) | `pkg/client/projects.go`, `pkg/client/errors.go`, `pkg/client/client.go`, `pkg/client/client_options.go`, `pkg/core/transport.go`, `pkg/apis/management/v1/client.go`, `pkg/apis/management/v1/api_project.go`, `api/openapi/management-open-api.yaml` | 2026-11-11 |
| [Project scoping](../ARCHITECTURE.md#project-scoping) | `pkg/client/call_options.go`, the façade methods that call it (`pkg/client/warehouses.go`, `pkg/client/roles.go`, `pkg/client/projects.go`, `pkg/client/permissions.go`), the `x-project-id` parameter in `api/openapi/management-open-api.yaml` | 2026-11-11 |
| [Authentication](../AUTHENTICATION.md#overview) | `pkg/core/auth.go`, `pkg/core/transport.go`, `pkg/client/client.go`, `pkg/client/client_options.go`, `pkg/client/server.go`, `pkg/lakekeeper/lakekeeper.go`, `cmd/lkctl/commands/client.go`, `cmd/lkctl/commands/root.go` | 2026-11-11 |
| [Package layers](../PACKAGES.md#package-layers) | the imports between this module's packages (`go list -f '{{.Imports}}' ./pkg/... ./cmd/...`), the `forbidigo` and `importas` settings in `.golangci.yml`, `pkg/lakekeeper/lakekeeper.go`, `pkg/client/client.go`, `pkg/permissions/assignment.go`, `pkg/storage/profile/profile.go`, `pkg/storage/credential/credential.go`, `pkg/core/auth.go`, `pkg/core/transport.go`, `pkg/apis/management/v1/client.go`, `cmd/lkctl/commands/` | 2026-11-11 |
| [Generation pipeline](../GENERATION.md#how-to-regenerate) | the `generate` target in `Makefile`, `api/openapi/management-open-api.yaml`, `api/openapi/management-config.yaml`, `api/openapi/.openapi-generator-ignore`, `api/openapi/postprocess/main.go`, `pkg/apis/management/v1/.openapi-generator/VERSION` | 2026-11-11 |
| [Development workflow](../../CONTRIBUTING.md#development-workflow) | the `validate`, `build`, `test`, `test-integration`, `test-e2e-compose` and `test-e2e-kind` targets in `Makefile`, `run-tests.sh`, `e2e/compose/run.sh`, `e2e/kind/run.sh`, `.github/workflows/ci.yml`, `.github/workflows/lint-pr.yml`, `.github/workflows/release-please.yml`, `.github/workflows/release.yml`, `release-please-config.json`, the Commits and branches section of `CONTRIBUTING.md` | 2026-11-11 |

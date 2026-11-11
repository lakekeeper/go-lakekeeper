---
name: api-coverage
description: Check that every Management API operation in the OpenAPI spec is generated into the Go SDK (pkg/apis/management/v1) and wrapped by a pkg/client façade, and draft a gap report. Use when asked what the SDK or façades are missing, after bumping the spec or running `make generate`, or before planning new façade work.
---

# API coverage check

The source of truth is the OpenAPI spec `api/openapi/management-open-api.yaml`.
Every operation (method + path) is checked at two levels:

1. **SDK**: a generated `XAPIService.Op` in `pkg/apis/management/v1/` whose
   `OpExecute` sends that method to that path.
2. **Façade**: an exported method in non-test `pkg/client/*.go` that calls that
   SDK method, either directly or through unexported helpers in the same package
   (e.g. `Grant` → `p.update` → `UpdateWarehouseAssignmentsById`).

Matching uses method + path, not names, so a renamed `operationId` or façade
method cannot hide a gap.

## Run

From the repo root:

```sh
go run .agents/skills/api-coverage/apicoverage.go -missing   # gaps only, grouped by spec tag
go run .agents/skills/api-coverage/apicoverage.go            # full table
go run .agents/skills/api-coverage/apicoverage.go -json      # machine-readable
```

The script has a `//go:build ignore` tag and lives under `.agents/`, so
`./...`, `make test` and `make validate` never pick it up. It uses
`gopkg.in/yaml.v3`, which is already in `go.sum`.

The script parses each file on its own, so it still works when `pkg/client`
doesn't compile. But it does read untracked or WIP files in `pkg/client/`. If
`git status` lists any there, say so in the report.

## Interpreting results

- **SDK `MISSING`**: the generated code is behind the spec. Fix it with
  `make generate` (see `docs/GENERATION.md`). Never hand-edit the SDK.
- **"SDK operations not in the spec"**: the generated code is stale in the
  other direction. Regenerate.
- **Façade `—`**: the operation can only be called through the embedded
  generated service (`c.TagAPI…`). This is expected for some operations (see
  the escape-hatch scope in `docs/PACKAGES.md`, `pkg/client` → Coverage), but
  `lkctl` can only call façades (`forbidigo`). So any CLI command that needs one
  of these operations needs a façade first.
- Operations marked `(deprecated)` usually have a non-deprecated replacement
  (e.g. `/project/{project_id}` → `/project` + `x-project-id`). Don't count
  them as gaps when the replacement is covered.

## Drafting the gap report

1. Run with `-missing`.
2. Group the façade gaps by resource and existing façade type (`Warehouses`,
   `Roles`, `Users`, `Server`, `Projects`, `Permissions.*`), and list any
   resource with no façade type yet (tags, tasks, grants, authorization).
3. Mark deprecated operations whose replacement is already covered, and leave
   them out of the counts.
4. Rank by value: operations `lkctl` or common SDK flows would need first;
   `*-actions` / `authorizer-actions` introspection last.
5. Note any SDK-level gaps or stale SDK operations separately. These need
   regeneration, not façade work.
6. If a façade-coverage paragraph in `docs/PACKAGES.md` now disagrees with the
   output, flag it.

Report only. Don't add façades unless the user asks.

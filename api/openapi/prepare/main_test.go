package main

import (
	"bytes"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

// lakekeeperSpec is the base fixture: one path, one plain schema, and one
// discriminated union.
const lakekeeperSpec = `openapi: 3.1.0
info:
  title: Lakekeeper Management API
  license:
    name: Apache-2.0
    identifier: Apache-2.0
  version: 0.0.0
servers:
  - url: 'https://lakekeeper'
paths:
  /management/v1/warehouse:
    get:
      tags:
        - warehouse
      operationId: list_warehouses
      responses:
        '200':
          description: Warehouses
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Warehouse'
components:
  schemas:
    Warehouse:
      type: object
      description: A warehouse.
      required:
        - name
      properties:
        name:
          type: string
          maxLength: 64
    StorageProfile:
      oneOf:
        - $ref: '#/components/schemas/StorageProfileS3'
      discriminator:
        propertyName: type
        mapping:
          s3: '#/components/schemas/StorageProfileS3'
    StorageProfileS3:
      type: object
      required:
        - type
      properties:
        type:
          type: string
  securitySchemes:
    bearerAuth:
      type: http
      scheme: bearer
security:
  - bearerAuth: []
tags:
  - name: warehouse
    description: Manage Warehouses
`

// mustPrepare runs prepare on the Lakekeeper fixture and plus without patches and decodes the
// merged spec into a generic map for assertions.
func mustPrepare(t *testing.T, plus string) map[string]any {
	t.Helper()
	out, err := prepare(spec{name: "lakekeeper", data: []byte(lakekeeperSpec)}, spec{name: "lakekeeper-plus", data: []byte(plus)}, &bytes.Buffer{})
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}
	var merged map[string]any
	if err := yaml.Unmarshal(out, &merged); err != nil {
		t.Fatalf("merged spec is not YAML: %v\n%s", err, out)
	}
	return merged
}

// prepareErr runs prepare on the Lakekeeper fixture and plus and expects it to fail.
func prepareErr(t *testing.T, plus string) error {
	t.Helper()
	_, err := prepare(spec{name: "lakekeeper", data: []byte(lakekeeperSpec)}, spec{name: "lakekeeper-plus", data: []byte(plus)}, &bytes.Buffer{})
	if err == nil {
		t.Fatal("prepare succeeded, want an error")
	}
	return err
}

// lookup walks a decoded spec by keys.
func lookup(t *testing.T, doc map[string]any, keys ...string) any {
	t.Helper()
	var cur any = doc
	for _, k := range keys {
		m, ok := cur.(map[string]any)
		if !ok {
			t.Fatalf("%s: not a map at %q", strings.Join(keys, "."), k)
		}
		cur, ok = m[k]
		if !ok {
			return nil
		}
	}
	return cur
}

func TestPrepare_LakekeeperWinsOnSharedSchema(t *testing.T) {
	plus := strings.Replace(lakekeeperSpec, "description: A warehouse.", "description: A Lakekeeper+ warehouse.", 1)
	plus = strings.Replace(plus, "maxLength: 64", "maxLength: 4096", 1)

	merged := mustPrepare(t, plus)

	if got := lookup(t, merged, "components", "schemas", "Warehouse", "description"); got != "A warehouse." {
		t.Errorf("Warehouse description: got %q, want the Lakekeeper one", got)
	}
	if got := lookup(t, merged, "components", "schemas", "Warehouse", "properties", "name", "maxLength"); got != 64 {
		t.Errorf("Warehouse name maxLength: got %v, want 64", got)
	}
}

// withMemoryVariant adds a `memory` variant to the StorageProfile union, as
// the Lakekeeper+ spec does.
func withMemoryVariant(lk string) string {
	s := strings.Replace(lk, `        - $ref: '#/components/schemas/StorageProfileS3'
`, `        - $ref: '#/components/schemas/StorageProfileS3'
        - $ref: '#/components/schemas/StorageProfileMemory'
`, 1)
	return strings.Replace(s, `          s3: '#/components/schemas/StorageProfileS3'
`, `          s3: '#/components/schemas/StorageProfileS3'
          memory: '#/components/schemas/StorageProfileMemory'
    StorageProfileMemory:
      type: object
      required:
        - type
      properties:
        type:
          type: string
`, 1)
}

func TestPrepare_AddsUnionVariant(t *testing.T) {
	merged := mustPrepare(t, withMemoryVariant(lakekeeperSpec))

	oneOf := lookup(t, merged, "components", "schemas", "StorageProfile", "oneOf").([]any)
	var refs []string
	for _, v := range oneOf {
		refs = append(refs, v.(map[string]any)["$ref"].(string))
	}
	want := "#/components/schemas/StorageProfileS3,#/components/schemas/StorageProfileMemory"
	if got := strings.Join(refs, ","); got != want {
		t.Errorf("oneOf: got %s, want %s", got, want)
	}
	if got := lookup(t, merged, "components", "schemas", "StorageProfile", "discriminator", "mapping", "memory"); got != "#/components/schemas/StorageProfileMemory" {
		t.Errorf("discriminator mapping memory: got %v", got)
	}
	if lookup(t, merged, "components", "schemas", "StorageProfileMemory") == nil {
		t.Error("variant schema StorageProfileMemory missing")
	}
}

func TestPrepare_NonAdditiveSchemaConflictFails(t *testing.T) {
	// Lakekeeper+ changes a property's type on a shared schema.
	plus := strings.Replace(lakekeeperSpec, `        name:
          type: string`, `        name:
          type: integer`, 1)

	err := prepareErr(t, plus)

	if !strings.Contains(err.Error(), "schema Warehouse") {
		t.Errorf("error does not name the schema: %v", err)
	}
}

func TestPrepare_DuplicateOperationIDFails(t *testing.T) {
	// Lakekeeper+ adds a different operation under an operationId Lakekeeper
	// already uses.
	plus := strings.Replace(plusAdditions(lakekeeperSpec), "operationId: get_cedar_schema", "operationId: list_warehouses", 1)

	err := prepareErr(t, plus)

	if !strings.Contains(err.Error(), "operationId list_warehouses") {
		t.Errorf("error does not name the operationId: %v", err)
	}
}

func TestPrepare_DescriptiveDifferencesDoNotFail(t *testing.T) {
	// Lakekeeper+ rewords descriptions, changes a maxLength and documents an
	// extra response code on a shared operation.
	plus := strings.Replace(lakekeeperSpec, "description: Warehouses", "description: All warehouses", 1)
	plus = strings.Replace(plus, "maxLength: 64", "maxLength: 4096", 1)
	plus = strings.Replace(plus, `                $ref: '#/components/schemas/Warehouse'
`, `                $ref: '#/components/schemas/Warehouse'
        '404':
          description: Not found
`, 1)

	merged := mustPrepare(t, plus)

	if lookup(t, merged, "paths", "/management/v1/warehouse", "get", "responses", "404") != nil {
		t.Error("Lakekeeper+'s extra response code was merged; the Lakekeeper operation should be kept")
	}
	if got := lookup(t, merged, "paths", "/management/v1/warehouse", "get", "responses", "200", "description"); got != "Warehouses" {
		t.Errorf("response description: got %q, want the Lakekeeper one", got)
	}
}

func TestPrepare_ConflictingSharedOperationFails(t *testing.T) {
	// Lakekeeper+ changes the response schema of a shared operation.
	plus := strings.Replace(withMemoryVariant(lakekeeperSpec), `                $ref: '#/components/schemas/Warehouse'
`, `                $ref: '#/components/schemas/StorageProfile'
`, 1)

	err := prepareErr(t, plus)

	if !strings.Contains(err.Error(), "operation get /management/v1/warehouse") {
		t.Errorf("error does not name the operation: %v", err)
	}
}

func TestPrepare_InfoServersAndSecurityComeFromLakekeeper(t *testing.T) {
	plus := strings.Replace(lakekeeperSpec, "title: Lakekeeper Management API", "title: Lakekeeper＋ Management API", 1)
	plus = strings.Replace(plus, `    name: Apache-2.0
    identifier: Apache-2.0`, `    name: Vakamo Enterprise License`, 1)
	plus = strings.Replace(plus, "https://lakekeeper", "https://lakekeeper-plus", 1)
	plus = strings.Replace(plus, "scheme: bearer", "scheme: basic", 1)

	merged := mustPrepare(t, plus)

	if got := lookup(t, merged, "info", "title"); got != "Lakekeeper Management API" {
		t.Errorf("info.title: got %q", got)
	}
	if got := lookup(t, merged, "info", "license", "name"); got != "Apache-2.0" {
		t.Errorf("info.license.name: got %q", got)
	}
	if got := lookup(t, merged, "servers").([]any)[0].(map[string]any)["url"]; got != "https://lakekeeper" {
		t.Errorf("servers[0].url: got %q", got)
	}
	if got := lookup(t, merged, "components", "securitySchemes", "bearerAuth", "scheme"); got != "bearer" {
		t.Errorf("bearerAuth scheme: got %q", got)
	}
}

func TestPrepare_AppliesEachSpecsPatchesBeforeMerging(t *testing.T) {
	// The Lakekeeper+ spec declares the name property as an integer; its patch
	// turns it back into a string, so the merge sees no conflict.
	plus := strings.Replace(plusAdditions(lakekeeperSpec), `        name:
          type: string`, `        name:
          type: integer`, 1)
	nameIsInteger := func(s *yaml.Node) *yaml.Node {
		n := get(get(get(get(get(get(s, "components"), "schemas"), "Warehouse"), "properties"), "name"), "type")
		if n != nil && n.Value == "integer" {
			return n
		}
		return nil
	}
	fixNameType := patch{
		name:   "warehouse-name-type",
		defect: func(s *yaml.Node) bool { return nameIsInteger(s) != nil },
		fix:    func(s *yaml.Node) { nameIsInteger(s).Value = "string" },
	}
	var log bytes.Buffer

	out, err := prepare(
		spec{name: "lakekeeper", data: []byte(lakekeeperSpec)},
		spec{name: "lakekeeper-plus", data: []byte(plus), patches: []patch{fixNameType}},
		&log,
	)
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}

	if !strings.Contains(string(out), "CedarSchema") {
		t.Error("merged spec lacks the Lakekeeper+ additions")
	}
	if !strings.Contains(log.String(), "patch warehouse-name-type: applied") {
		t.Errorf("log: got %q", log.String())
	}
}

// plusAdditions adds to the Lakekeeper fixture what a Lakekeeper+ spec adds: a
// path, an operation on a shared path, a schema and a tag.
func plusAdditions(lk string) string {
	s := strings.Replace(lk, `components:
  schemas:
`, `  /management/v1/permissions/cedar/schema:
    get:
      tags:
        - permissions-cedar
      operationId: get_cedar_schema
      responses:
        '200':
          description: Cedar schema
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/CedarSchema'
components:
  schemas:
    CedarSchema:
      type: object
`, 1)
	s = strings.Replace(s, `  /management/v1/warehouse:
`, `  /management/v1/warehouse:
    post:
      tags:
        - warehouse
      operationId: create_plus_warehouse
      responses:
        '201':
          description: Created
`, 1)
	return s + `  - name: permissions-cedar
    description: Cedar Authorization Management.
`
}

func TestPrepare_AddsLakekeeperPlusOnlyItems(t *testing.T) {
	merged := mustPrepare(t, plusAdditions(lakekeeperSpec))

	if lookup(t, merged, "paths", "/management/v1/permissions/cedar/schema", "get") == nil {
		t.Error("Lakekeeper+-only path missing")
	}
	if got := lookup(t, merged, "paths", "/management/v1/warehouse", "post", "operationId"); got != "create_plus_warehouse" {
		t.Errorf("Lakekeeper+-only operation on a shared path: got %v", got)
	}
	if lookup(t, merged, "components", "schemas", "CedarSchema") == nil {
		t.Error("Lakekeeper+-only schema missing")
	}
	var names []string
	for _, tag := range lookup(t, merged, "tags").([]any) {
		names = append(names, tag.(map[string]any)["name"].(string))
	}
	if got := strings.Join(names, ","); got != "warehouse,permissions-cedar" {
		t.Errorf("tags: got %s, want warehouse,permissions-cedar", got)
	}
}

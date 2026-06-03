package main

import (
	"testing"

	"gopkg.in/yaml.v3"
)

// loadDoc parses a YAML document literal and returns the root mapping node
// (i.e. the document's first content child).
func loadDoc(t *testing.T, src string) *yaml.Node {
	t.Helper()
	var doc yaml.Node
	if err := yaml.Unmarshal([]byte(src), &doc); err != nil {
		t.Fatalf("yaml.Unmarshal: %v", err)
	}
	return docRoot(&doc)
}

// findGetQueryParam returns the parameter mapping for (path, name) under
// paths.<path>.get.parameters, or nil if not found.
func findGetQueryParam(t *testing.T, root *yaml.Node, path, name string) *yaml.Node {
	t.Helper()
	paths := lookup(root, "paths")
	if paths == nil {
		t.Fatalf("paths not found")
	}
	op := lookup(lookup(paths, path), "get")
	if op == nil {
		t.Fatalf("operation GET %s not found", path)
	}
	params := lookup(op, "parameters")
	if params == nil || params.Kind != yaml.SequenceNode {
		t.Fatalf("parameters not found for GET %s", path)
	}
	for _, p := range params.Content {
		n := lookup(p, "name")
		if n != nil && n.Value == name {
			return p
		}
	}
	return nil
}

// scalarValue returns the scalar value of the child key, or "" if absent.
func scalarValue(n *yaml.Node, key string) string {
	v := lookup(n, key)
	if v == nil || v.Kind != yaml.ScalarNode {
		return ""
	}
	return v.Value
}

func TestForceBracketsOnRelations_IgnoresOtherArrayParams(t *testing.T) {
	src := `
paths:
  /things:
    get:
      parameters:
        - name: tags
          in: query
          schema:
            type: array
            items:
              type: string
`
	root := loadDoc(t, src)
	forceBracketsOnRelations(root)

	p := findGetQueryParam(t, root, "/things", "tags")
	if p == nil {
		t.Fatal("tags parameter not found")
	}
	if got := scalarValue(p, "style"); got != "" {
		t.Errorf("style: got %q, want empty (untouched)", got)
	}
	if got := scalarValue(p, "explode"); got != "" {
		t.Errorf("explode: got %q, want empty (untouched)", got)
	}
}

func TestForceBracketsOnRelations_Idempotent(t *testing.T) {
	src := `
paths:
  /assignments:
    get:
      parameters:
        - name: relations
          in: query
          style: deepObject
          explode: false
          schema:
            type: array
            items:
              type: string
`
	root := loadDoc(t, src)
	forceBracketsOnRelations(root)

	p := findGetQueryParam(t, root, "/assignments", "relations")
	if p == nil {
		t.Fatal("relations parameter not found")
	}

	// Count occurrences of each key under the parameter mapping. With the
	// naive append impl, re-running over an already-styled param produces
	// duplicate `style`/`explode` keys.
	count := func(key string) int {
		n := 0
		for i := 0; i+1 < len(p.Content); i += 2 {
			if p.Content[i].Value == key {
				n++
			}
		}
		return n
	}
	if got := count("style"); got != 1 {
		t.Errorf("style key count: got %d, want 1", got)
	}
	if got := count("explode"); got != 1 {
		t.Errorf("explode key count: got %d, want 1", got)
	}
	if got := scalarValue(p, "style"); got != "deepObject" {
		t.Errorf("style: got %q, want %q", got, "deepObject")
	}
	if got := scalarValue(p, "explode"); got != "false" {
		t.Errorf("explode: got %q, want %q", got, "false")
	}
}

func TestForceBracketsOnRelations_IgnoresScalarRelations(t *testing.T) {
	src := `
paths:
  /ops:
    get:
      parameters:
        - name: relations
          in: query
          schema:
            type: string
`
	root := loadDoc(t, src)
	forceBracketsOnRelations(root)

	p := findGetQueryParam(t, root, "/ops", "relations")
	if p == nil {
		t.Fatal("relations parameter not found")
	}
	if got := scalarValue(p, "style"); got != "" {
		t.Errorf("style: got %q, want empty (scalar relations untouched)", got)
	}
	if got := scalarValue(p, "explode"); got != "" {
		t.Errorf("explode: got %q, want empty (scalar relations untouched)", got)
	}
}

func TestForceBracketsOnRelations_AddsStyleAndExplodeWhenMissing(t *testing.T) {
	src := `
paths:
  /assignments:
    get:
      parameters:
        - name: relations
          in: query
          schema:
            type: array
            items:
              type: string
`
	root := loadDoc(t, src)
	forceBracketsOnRelations(root)

	p := findGetQueryParam(t, root, "/assignments", "relations")
	if p == nil {
		t.Fatal("relations parameter not found after rewrite")
	}
	if got := scalarValue(p, "style"); got != "deepObject" {
		t.Errorf("style: got %q, want %q", got, "deepObject")
	}
	if got := scalarValue(p, "explode"); got != "false" {
		t.Errorf("explode: got %q, want %q", got, "false")
	}
}

// schemasOf returns the components.schemas mapping node from a parsed doc.
func schemasOf(t *testing.T, root *yaml.Node) *yaml.Node {
	t.Helper()
	schemas := lookup(lookup(root, "components"), "schemas")
	if schemas == nil || schemas.Kind != yaml.MappingNode {
		t.Fatal("components.schemas not found")
	}
	return schemas
}

// TestExtractAndDiscriminate_FlatEnumUnion covers a oneOf whose members are
// `allOf([{$ref}, {type: single-enum}])` with no nested oneOf — the shape that
// expandOneOf leaves untouched. After the loop stopped gating on expandOneOf,
// such unions must still be extracted into named leaf schemas and gain a
// discriminator, so the generator emits discriminator-based (not match-counting)
// decode. This is the StorageProfile / TabularDeleteProfile case.
func TestExtractAndDiscriminate_FlatEnumUnion(t *testing.T) {
	src := `
components:
  schemas:
    S3Profile:
      type: object
      properties:
        bucket:
          type: string
    GcsProfile:
      type: object
      properties:
        bucket:
          type: string
    StorageProfile:
      oneOf:
        - allOf:
            - $ref: '#/components/schemas/S3Profile'
            - type: object
              required:
                - type
              properties:
                type:
                  type: string
                  enum:
                    - s3
        - allOf:
            - $ref: '#/components/schemas/GcsProfile'
            - type: object
              required:
                - type
              properties:
                type:
                  type: string
                  enum:
                    - gcs
`
	root := loadDoc(t, src)
	schemas := schemasOf(t, root)
	sp := lookupSchema(schemas, "StorageProfile")

	// Mirror the main loop: expandOneOf is a no-op here, then discriminate.
	expandOneOf(sp, schemas)
	if err := extractAndDiscriminate("StorageProfile", sp, schemas); err != nil {
		t.Fatalf("extractAndDiscriminate: %v", err)
	}

	// A discriminator on `type` with a mapping to the synthesized leaves.
	disc := lookup(sp, "discriminator")
	if disc == nil {
		t.Fatal("no discriminator block emitted")
	}
	if got := scalarValue(disc, "propertyName"); got != "type" {
		t.Errorf("discriminator propertyName: got %q, want %q", got, "type")
	}
	mapping := lookup(disc, "mapping")
	if mapping == nil || mapping.Kind != yaml.MappingNode {
		t.Fatal("discriminator mapping not found")
	}
	if got := scalarValue(mapping, "s3"); got != refPrefix+"StorageProfileS3" {
		t.Errorf("mapping[s3]: got %q, want %q", got, refPrefix+"StorageProfileS3")
	}
	if got := scalarValue(mapping, "gcs"); got != refPrefix+"StorageProfileGcs" {
		t.Errorf("mapping[gcs]: got %q, want %q", got, refPrefix+"StorageProfileGcs")
	}

	// Each oneOf member is now a $ref to a named leaf schema.
	oneOf := lookup(sp, "oneOf")
	if oneOf == nil || len(oneOf.Content) != 2 {
		t.Fatalf("oneOf: want 2 members, got %v", oneOf)
	}
	for _, m := range oneOf.Content {
		if lookup(m, "$ref") == nil {
			t.Errorf("oneOf member is not a $ref: %+v", m)
		}
	}
	if lookupSchema(schemas, "StorageProfileS3") == nil {
		t.Error("leaf schema StorageProfileS3 was not appended")
	}
	if lookupSchema(schemas, "StorageProfileGcs") == nil {
		t.Error("leaf schema StorageProfileGcs was not appended")
	}
}

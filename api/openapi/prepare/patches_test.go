package main

import (
	"bytes"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

// patchSpec parses src, runs the patches on it and returns the patched spec
// re-encoded, and the log.
func patchSpec(t *testing.T, src string, patches ...patch) (string, string) {
	t.Helper()
	var doc yaml.Node
	if err := yaml.Unmarshal([]byte(src), &doc); err != nil {
		t.Fatalf("fixture: %v", err)
	}
	var log bytes.Buffer
	if err := applyPatches(doc.Content[0], patches, &log); err != nil {
		t.Fatalf("applyPatches: %v", err)
	}
	out, err := encode(doc.Content[0])
	if err != nil {
		t.Fatal(err)
	}
	return string(out), log.String()
}

const plusLicenseDefect = `openapi: 3.1.0
info:
  title: Lakekeeper＋ Management API
  license:
    name: Vakamo Enterprise License
  version: 0.0.0
`

func TestLicenseIdentifier_FixesMissingIdentifier(t *testing.T) {
	out, log := patchSpec(t, plusLicenseDefect, licenseIdentifier)

	if !strings.Contains(out, "identifier: LicenseRef-Vakamo-Enterprise-License") {
		t.Errorf("license identifier not added:\n%s", out)
	}
	if !strings.Contains(out, "name: Vakamo Enterprise License") {
		t.Errorf("license name lost:\n%s", out)
	}
	if !strings.Contains(log, "patch license-identifier: applied") {
		t.Errorf("log: got %q", log)
	}
}

func TestLicenseIdentifier_Idempotent(t *testing.T) {
	once, _ := patchSpec(t, plusLicenseDefect, licenseIdentifier)
	twice, log := patchSpec(t, once, licenseIdentifier)

	if twice != once {
		t.Errorf("second run changed the spec:\nonce:\n%s\ntwice:\n%s", once, twice)
	}
	if !strings.Contains(log, "patch license-identifier: not needed, skipped") {
		t.Errorf("second run log: got %q", log)
	}
}

func TestLicenseIdentifier_SkippedWhenDefectAbsent(t *testing.T) {
	fixed := strings.Replace(plusLicenseDefect, "name: Vakamo Enterprise License", "name: Vakamo Enterprise License\n    url: https://vakamo.com/license", 1)

	out, log := patchSpec(t, fixed, licenseIdentifier)

	if strings.Contains(out, "identifier:") {
		t.Errorf("identifier added although the license has a url:\n%s", out)
	}
	if got, want := strings.TrimSpace(log), "patch license-identifier: not needed, skipped"; got != want {
		t.Errorf("log: got %q, want %q", got, want)
	}
}

// assertSkipped checks that patch p leaves the defect-free src unchanged and
// logs only that it was skipped.
func assertSkipped(t *testing.T, src string, p patch) {
	t.Helper()
	unpatched, _ := patchSpec(t, src)
	out, log := patchSpec(t, src, p)

	if out != unpatched {
		t.Errorf("spec changed although the defect is absent:\n%s", out)
	}
	if got, want := strings.TrimSpace(log), "patch "+p.name+": not needed, skipped"; got != want {
		t.Errorf("log: got %q, want %q", got, want)
	}
}

// assertIdempotent checks that a second run of patch p on the patched src
// changes nothing and is skipped.
func assertIdempotent(t *testing.T, src string, p patch) {
	t.Helper()
	once, _ := patchSpec(t, src, p)
	twice, log := patchSpec(t, once, p)

	if twice != once {
		t.Errorf("second run changed the spec:\nonce:\n%s\ntwice:\n%s", once, twice)
	}
	if !strings.Contains(log, "patch "+p.name+": not needed, skipped") {
		t.Errorf("second run log: got %q", log)
	}
}

const plusBooleanDefaultDefect = `components:
  schemas:
    ExpireSnapshotsQueueConfig:
      type: object
      properties:
        disable-min-age-check:
          type: boolean
          default: false
        enable-expire-snapshots:
          type:
            - boolean
            - 'null'
          default: 'false'
        mode:
          type: string
          default: 'false'
`

func TestBooleanDefault_FixesStringDefault(t *testing.T) {
	out, log := patchSpec(t, plusBooleanDefaultDefect, booleanDefault)

	want := `        enable-expire-snapshots:
          type:
            - boolean
            - 'null'
          default: false
`
	if !strings.Contains(out, want) {
		t.Errorf("boolean default not fixed:\n%s", out)
	}
	if !strings.Contains(out, "type: string\n          default: 'false'") {
		t.Errorf("a string property's default changed:\n%s", out)
	}
	if !strings.Contains(log, "patch boolean-default: applied") {
		t.Errorf("log: got %q", log)
	}
}

func TestBooleanDefault_Idempotent(t *testing.T) {
	assertIdempotent(t, plusBooleanDefaultDefect, booleanDefault)
}

func TestBooleanDefault_SkippedWhenDefectAbsent(t *testing.T) {
	fixed := strings.Replace(plusBooleanDefaultDefect, "- 'null'\n          default: 'false'", "- 'null'\n          default: false", 1)
	assertSkipped(t, fixed, booleanDefault)
}

const plusInheritDefaultsDefect = `components:
  schemas:
    ExpireSnapshotsQueueConfig:
      type: object
      properties:
        disable-min-age-check:
          type: boolean
          default: false
        max-snapshot-age-ms:
          type:
            - integer
            - 'null'
          format: int64
          default: 432000000
        min-snapshots-to-keep:
          type:
            - integer
            - 'null'
          default: 1
    OptimizeQueueConfig:
      type: object
      properties:
        enable-compaction:
          type:
            - boolean
            - 'null'
    TableSettings:
      type: object
      properties:
        retries:
          type:
            - integer
            - 'null'
          default: 3
`

func TestNoInheritDefaults_DropsNullableQueueConfigDefaults(t *testing.T) {
	out, log := patchSpec(t, plusInheritDefaultsDefect, noInheritDefaults)

	if strings.Contains(out, "default: 432000000") || strings.Contains(out, "default: 1\n") {
		t.Errorf("nullable queue-config defaults kept:\n%s", out)
	}
	if !strings.Contains(out, "format: int64\n") {
		t.Errorf("other keywords of the field lost:\n%s", out)
	}
	if !strings.Contains(out, "type: boolean\n          default: false") {
		t.Errorf("a non-nullable field's default was dropped:\n%s", out)
	}
	if !strings.Contains(out, "default: 3") {
		t.Errorf("a default outside the queue configs was dropped:\n%s", out)
	}
	if !strings.Contains(log, "patch no-inherit-defaults: applied") {
		t.Errorf("log: got %q", log)
	}
}

func TestNoInheritDefaults_Idempotent(t *testing.T) {
	assertIdempotent(t, plusInheritDefaultsDefect, noInheritDefaults)
}

func TestNoInheritDefaults_SkippedWhenDefectAbsent(t *testing.T) {
	fixed := strings.NewReplacer("          default: 432000000\n", "", "          default: 1\n", "").Replace(plusInheritDefaultsDefect)
	assertSkipped(t, fixed, noInheritDefaults)
}

const plusInlineEnumsDefect = `components:
  schemas:
    CedarResolveResource:
      oneOf:
        - type: string
          enum:
            - server
        - type: object
          title: CedarResolveResourceProject
          properties:
            project:
              type: object
    ScheduleOptimizeTaskRequest:
      type: object
      properties:
        payload:
          type: object
          properties:
            steps:
              type:
                - array
                - 'null'
              items:
                type: string
                description: A step of an optimize run, in run order.
                enum:
                  - compaction
                  - rewrite-manifests
`

const plusInlineEnumsFixed = `components:
  schemas:
    CedarResolveResource:
      oneOf:
        - $ref: '#/components/schemas/CedarResolveResourceServer'
        - type: object
          title: CedarResolveResourceProject
          properties:
            project:
              type: object
    ScheduleOptimizeTaskRequest:
      type: object
      properties:
        payload:
          type: object
          properties:
            steps:
              type:
                - array
                - 'null'
              items:
                $ref: '#/components/schemas/OptimizeStep'
    CedarResolveResourceServer:
      type: string
      enum:
        - server
    OptimizeStep:
      type: string
      description: A step of an optimize run, in run order.
      enum:
        - compaction
        - rewrite-manifests
`

func TestNamedInlineEnums_MovesEnumsToNamedSchemas(t *testing.T) {
	out, log := patchSpec(t, plusInlineEnumsDefect, namedInlineEnums)

	if want := mustEncode(t, plusInlineEnumsFixed); out != want {
		t.Errorf("got:\n%s\nwant:\n%s", out, want)
	}
	if !strings.Contains(log, "patch named-inline-enums: applied") {
		t.Errorf("log: got %q", log)
	}
}

func TestNamedInlineEnums_Idempotent(t *testing.T) {
	assertIdempotent(t, plusInlineEnumsDefect, namedInlineEnums)
}

func TestNamedInlineEnums_SkippedWhenDefectAbsent(t *testing.T) {
	assertSkipped(t, plusInlineEnumsFixed, namedInlineEnums)
}

const plusUntypedCedarListsDefect = `components:
  schemas:
    GetCedarPoliciesResponse:
      type: object
      properties:
        policies:
          type: array
          items: {}
          description: List of JSON formatted Cedar policies.
    ResolveCedarEntitiesResponse:
      type: object
      properties:
        entities:
          type: array
          items: {}
          description: Cedar entities in native JSON format.
`

func TestTypedCedarLists_NamesTheItemSchemas(t *testing.T) {
	out, log := patchSpec(t, plusUntypedCedarListsDefect, typedCedarLists)

	for _, want := range []string{
		"items:\n            $ref: '#/components/schemas/CedarPolicy'\n          description: List of JSON formatted Cedar policies.",
		"items:\n            $ref: '#/components/schemas/CedarEntity'\n          description: Cedar entities in native JSON format.",
		"\n    CedarPolicy:\n", "\n    CedarPolicyScopeConstraint:\n", "\n    CedarPolicyCondition:\n",
		"\n    CedarEntity:\n", "\n    CedarEntityUid:\n",
		"effect:\n          $ref: '#/components/schemas/PolicyEffect'",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
	if strings.Contains(out, "items: {}") {
		t.Errorf("an untyped list is left:\n%s", out)
	}
	if !strings.Contains(log, "patch typed-cedar-lists: applied") {
		t.Errorf("log: got %q", log)
	}
}

func TestTypedCedarLists_FixesTheListStillUntyped(t *testing.T) {
	src := strings.Replace(plusUntypedCedarListsDefect, "items: {}\n          description: Cedar entities", "items:\n            type: string\n          description: Cedar entities", 1)

	out, _ := patchSpec(t, src, typedCedarLists)

	if !strings.Contains(out, "$ref: '#/components/schemas/CedarPolicy'") {
		t.Errorf("policies list not typed:\n%s", out)
	}
	if strings.Contains(out, "\n    CedarEntity:\n") || !strings.Contains(out, "type: string\n          description: Cedar entities") {
		t.Errorf("the already typed entities list changed:\n%s", out)
	}
}

func TestTypedCedarLists_Idempotent(t *testing.T) {
	assertIdempotent(t, plusUntypedCedarListsDefect, typedCedarLists)
}

func TestTypedCedarLists_SkippedWhenDefectAbsent(t *testing.T) {
	typed := strings.ReplaceAll(plusUntypedCedarListsDefect, "items: {}", "items:\n            type: object")
	assertSkipped(t, typed, typedCedarLists)
}

// mustEncode returns src parsed and re-encoded without patches, in the form
// patchSpec returns.
func mustEncode(t *testing.T, src string) string {
	t.Helper()
	out, _ := patchSpec(t, src)
	return out
}

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

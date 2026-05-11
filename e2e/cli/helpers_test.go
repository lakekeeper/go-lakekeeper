//go:build e2e_cli

// Test helpers used across compose-only specs to provision (and clean up)
// shared resource shapes through the CLI. These are *not* SDK helpers — every
// helper here goes through lkctl, so the assertion stays on the CLI surface
// even when the resource is incidental to the test under inspection.

package clie2e

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"
)

// MustProvisionUser creates a fresh user via `lkctl user create` and registers
// a cleanup. Returns the user id (the OIDC subject string).
func MustProvisionUser(t *testing.T) string {
	t.Helper()

	id := "oidc~" + randomUUID()
	name := randomName("e2e-user")
	runOK(t, "user", "create", id, name, "human", "--email", name+"@example.com")
	t.Cleanup(func() {
		_, _, _, _ = activeBackend.Exec(t.Context(), nil,
			append(authFlagsOAuth2(), "user", "delete", id)...)
	})
	return id
}

// MustProvisionRole creates a fresh role on the default project via
// `lkctl role create` and registers a cleanup. Returns the role id.
func MustProvisionRole(t *testing.T) string {
	t.Helper()

	name := randomName("e2e-role")
	out := runOK(t, "role", "create", name, "--output", "json")
	var role struct {
		ID string `json:"id"`
	}
	decodeJSON(t, out, &role)
	if role.ID == "" {
		t.Fatalf("create role: empty id in output %s", out)
	}
	t.Cleanup(func() {
		_, _, _, _ = activeBackend.Exec(t.Context(), nil,
			append(authFlagsOAuth2(), "role", "delete", role.ID)...)
	})
	return role.ID
}

// MustProvisionWarehouse creates a fresh MinIO-backed warehouse on the
// default project via `lkctl warehouse create -f -` (config piped on stdin).
// Returns (warehouseID, warehouseName).
//
// Storage credentials match the docker-compose MinIO root user. Endpoint
// uses the in-stack DNS name `minio:9000` because the catalog validates
// connectivity from inside the network.
func MustProvisionWarehouse(t *testing.T) (string, string) {
	t.Helper()

	name := randomName("e2e-wh")
	cfg := warehouseConfig(name)

	args := append(authFlagsOAuth2(),
		"warehouse", "create", name,
		"-f", "-",
	)
	stdout, stderr, code := runRaw(t, cfg, args...)
	if code != 0 {
		t.Fatalf("create warehouse exit %d\nstdout: %s\nstderr: %s", code, stdout, stderr)
	}

	id := parseIDFromCreate(t, string(stdout))

	t.Cleanup(func() {
		_, _, _, _ = activeBackend.Exec(t.Context(), nil,
			append(authFlagsOAuth2(), "warehouse", "delete", id)...)
	})
	return id, name
}

// warehouseConfig produces a minimal CreateWarehouseRequest JSON for the
// in-stack MinIO. We hand-roll the JSON rather than depending on
// pkg/storage/profile to keep this package free of SDK type imports beyond
// what the harness itself needs.
//
// The bucket is shared across all e2e warehouses (one MinIO, one bucket),
// so we scope each warehouse to a unique key-prefix derived from the
// warehouse name. Without this, parallel `lkctl warehouse create` calls
// race on the default empty prefix and the second one fails with
// CreateWarehouseStorageProfileOverlap.
func warehouseConfig(name string) []byte {
	cfg := map[string]any{
		"warehouse-name": name,
		"storage-profile": map[string]any{
			"type":              "s3",
			"bucket":            "testacc",
			"key-prefix":        "warehouses/" + name,
			"region":            "eu-local-1",
			"endpoint":          "http://minio:9000/",
			"path-style-access": true,
			"sts-enabled":       false,
		},
		"storage-credential": map[string]any{
			"type":              "s3",
			"credential-type":   "access-key",
			"access-key-id":     "minio-root-user",
			"secret-access-key": "minio-root-password",
		},
	}
	out, err := json.Marshal(cfg)
	if err != nil {
		panic(fmt.Sprintf("marshal warehouse config: %v", err))
	}
	return out
}

// parseIDFromCreate digs the UUID out of any `Foo bar created with id <uuid>`
// shaped output line. The marker is the same for every lkctl create verb, so
// it is wired here rather than passed by each caller.
func parseIDFromCreate(t *testing.T, out string) string {
	t.Helper()
	const marker = "with id "
	i := strings.Index(out, marker)
	if i < 0 {
		t.Fatalf("create output missing %q marker: %q", marker, out)
	}
	return strings.TrimSpace(out[i+len(marker):])
}

// randomUUID is a thin wrapper around uuid.NewString for readability at call
// sites that compose subject strings like `oidc~<uuid>`.
func randomUUID() string {
	return uuid.NewString()
}

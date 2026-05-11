//go:build e2e_cli

// Exercises the warehouse-create CLI surface for each of the three cloud
// storage backends (S3, ADLS, GCS) using synthetic credentials. The compose
// Lakekeeper service runs with LAKEKEEPER__SKIP_STORAGE_VALIDATION=true so
// the server accepts these payloads without trying to talk to AWS / Azure /
// GCP. Coverage is for request-shape / SDK-generation drift, not for real
// cloud connectivity.

package clie2e

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWarehouseStorageProfiles(t *testing.T) {
	requireBackend(t, BackendCompose)
	t.Parallel()

	cases := []struct {
		name string
		cfg  map[string]any
	}{
		{name: "s3", cfg: s3FakeProfile()},
		{name: "adls", cfg: adlsFakeProfile()},
		{name: "gcs", cfg: gcsFakeProfile()},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			whName := randomName("e2e-wh-" + tc.name)
			tc.cfg["warehouse-name"] = whName
			body, err := json.Marshal(tc.cfg)
			require.NoError(t, err)

			args := append(authFlagsOAuth2(),
				"warehouse", "create", whName, "-f", "-")
			stdout, stderr, code := runRaw(t, body, args...)
			require.Zero(t, code, "stdout: %s\nstderr: %s", stdout, stderr)

			id := parseIDFromCreate(t, string(stdout))
			t.Cleanup(func() {
				_, _, _, _ = activeBackend.Exec(t.Context(), nil,
					append(authFlagsOAuth2(), "warehouse", "delete", id)...)
			})

			getOut := runOK(t, "warehouse", "get", id, "--output", "json")
			require.Contains(t, string(getOut), id)
		})
	}
}

// s3FakeProfile returns a CreateWarehouseRequest body targeting a synthetic
// S3 bucket. Deliberately *not* MinIO — using a non-existent bucket proves
// that LAKEKEEPER__SKIP_STORAGE_VALIDATION is actually engaged on the server.
func s3FakeProfile() map[string]any {
	return map[string]any{
		"storage-profile": map[string]any{
			"type":        "s3",
			"bucket":      "e2e-fake-bucket",
			"region":      "us-east-1",
			"sts-enabled": false,
		},
		"storage-credential": map[string]any{
			"type":              "s3",
			"credential-type":   "access-key",
			"access-key-id":     "AKIAFAKEACCESSKEYID",
			"secret-access-key": "fakeSecretAccessKey0000000000000000000000",
		},
	}
}

// adlsFakeProfile returns a CreateWarehouseRequest body targeting a
// synthetic ADLS Gen2 filesystem with throwaway client-credentials auth.
func adlsFakeProfile() map[string]any {
	return map[string]any{
		"storage-profile": map[string]any{
			"type":         "adls",
			"account-name": "e2efakeaccount",
			"filesystem":   "e2e-fake-filesystem",
		},
		"storage-credential": map[string]any{
			"type":            "az",
			"credential-type": "client-credentials",
			"client-id":       "00000000-0000-0000-0000-000000000000",
			"tenant-id":       "11111111-1111-1111-1111-111111111111",
			"client-secret":   "fake-client-secret",
		},
	}
}

// gcsFakeProfile returns a CreateWarehouseRequest body targeting a synthetic
// GCS bucket with the system-identity credential branch (no key payload —
// keeps the request compact while still exercising the GcsCredential oneOf).
func gcsFakeProfile() map[string]any {
	return map[string]any{
		"storage-profile": map[string]any{
			"type":   "gcs",
			"bucket": "e2e-fake-bucket",
		},
		"storage-credential": map[string]any{
			"type":            "gcs",
			"credential-type": "gcp-system-identity",
		},
	}
}

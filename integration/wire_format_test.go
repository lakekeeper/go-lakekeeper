//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	managementv1 "github.com/lakekeeper/go-lakekeeper/pkg/apis/management/v1"
	"github.com/lakekeeper/go-lakekeeper/pkg/storage/credential"
	"github.com/lakekeeper/go-lakekeeper/pkg/storage/profile"
)

// TestWireFormat_CredentialVariants confirms — does not discover — that the
// preprocessor + generator combination keeps each S3 credential variant's JSON
// wire shape isolated: the discriminator + variant-specific fields appear,
// and zero-valued fields from sibling variants do not bleed through. If this
// ever fails, that's a regression in the preprocessor (api/openapi/preprocess)
// or the spec, not a fresh discovery — bisect those, not this test.
//
// Each variant is checked locally (CreateWarehouseRequest → JSON) and the
// access-key variant is also pushed end-to-end against the docker-compose
// stack (the only variant whose backend is wired up). For aws-system-identity
// and cloudflare-r2, end-to-end exercise is impractical because the stack
// doesn't enable AWS system identities and has no R2 endpoint; the local
// round-trip is sufficient confirmation that the generated types still
// produce the expected wire shape.
func TestWireFormat_CredentialVariants(t *testing.T) {
	tests := []struct {
		name              string
		credential        managementv1.StorageCredential
		expectedType      string
		expectedCredType  string
		mustHaveFields    []string // fields that must appear in marshalled JSON
		mustNotHaveFields []string // sibling-variant fields that must not bleed through
	}{
		{
			name:             "access-key",
			credential:       credential.NewS3AccessKey("ak", "sk"),
			expectedType:     "s3",
			expectedCredType: "access-key",
			mustHaveFields:   []string{"access-key-id", "secret-access-key"},
			// account-id / token belong to cloudflare-r2; aws-system-identity
			// has no extra mandatory fields, so the only sibling fingerprint
			// to look for is the cloudflare-r2 ones.
			mustNotHaveFields: []string{"account-id", "token"},
		},
		{
			name:              "aws-system-identity",
			credential:        credential.NewS3AwsSystemIdentity(),
			expectedType:      "s3",
			expectedCredType:  "aws-system-identity",
			mustHaveFields:    []string{"credential-type", "type"},
			mustNotHaveFields: []string{"access-key-id", "secret-access-key", "account-id", "token"},
		},
		{
			name:              "cloudflare-r2",
			credential:        credential.NewS3CloudflareR2("ak", "sk", "acct", "tok"),
			expectedType:      "s3",
			expectedCredType:  "cloudflare-r2",
			mustHaveFields:    []string{"access-key-id", "secret-access-key", "account-id", "token"},
			mustNotHaveFields: []string{}, // cloudflare-r2 is a superset of access-key fields
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			sp := managementv1.StorageProfileS3AsStorageProfile(profile.NewS3Profile(
				"testacc", "eu-local-1",
				profile.WithS3Endpoint("http://minio:9000/"),
				profile.WithS3PathStyleAccess(),
			))

			req := managementv1.NewCreateWarehouseRequest(sp, "wire-format-"+tc.name)
			req.SetProjectId(defaultProjectID)
			req.SetStorageCredential(tc.credential)

			b, err := json.Marshal(req)
			require.NoError(t, err)

			// the credential is nested inside the storage-credential field;
			// extract that subtree to assert only the variant's fields are
			// present.
			var envelope struct {
				StorageCredential map[string]any `json:"storage-credential"`
			}
			require.NoError(t, json.Unmarshal(b, &envelope))
			require.NotNil(t, envelope.StorageCredential)

			assert.Equal(t, tc.expectedType, envelope.StorageCredential["type"])
			assert.Equal(t, tc.expectedCredType, envelope.StorageCredential["credential-type"])
			for _, f := range tc.mustHaveFields {
				_, ok := envelope.StorageCredential[f]
				assert.Truef(t, ok, "expected field %q in marshalled credential, got %v", f, envelope.StorageCredential)
			}
			for _, f := range tc.mustNotHaveFields {
				_, ok := envelope.StorageCredential[f]
				assert.Falsef(t, ok, "sibling-variant field %q leaked into marshalled credential: %v", f, envelope.StorageCredential)
			}
		})
	}
}

// TestWireFormat_AccessKeyEndToEnd is the access-key-specific end-to-end
// confirmation: create a warehouse, fetch it, assert the response's
// StorageProfile JSON also stays clean (no sibling-union bleed-through on the
// way back). The credential isn't returned by the server, so we can only
// inspect the StorageProfile half of the round-trip — but that exercises the
// same preprocessor-generated *AsStorageProfile / oneOf code paths.
func TestWireFormat_AccessKeyEndToEnd(t *testing.T) {
	c := sharedClient

	sp := managementv1.StorageProfileS3AsStorageProfile(profile.NewS3Profile(
		"testacc", "eu-local-1",
		profile.WithS3Endpoint("http://minio:9000/"),
		profile.WithS3PathStyleAccess(),
	))
	sc := credential.NewS3AccessKey("minio-root-user", "minio-root-password")

	req := managementv1.NewCreateWarehouseRequest(sp, "wire-format-roundtrip")
	req.SetProjectId(defaultProjectID)
	req.SetStorageCredential(sc)

	wh, r, err := c.WarehouseAPI.CreateWarehouse(t.Context()).CreateWarehouseRequest(*req).Execute()
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, r.StatusCode)

	t.Cleanup(func() {
		if _, err := c.WarehouseAPI.DeleteWarehouse(context.Background(), wh.WarehouseId).Execute(); err != nil {
			t.Errorf("delete warehouse: %v", err)
		}
	})

	got, _, err := c.WarehouseAPI.GetWarehouse(t.Context(), wh.WarehouseId).Execute()
	require.NoError(t, err)
	require.NotNil(t, got)

	// Marshal the round-tripped storage profile and decode to a map so we
	// assert on keys, not on whitespace/field-order quirks of the encoder.
	bytes, err := json.Marshal(got.StorageProfile)
	require.NoError(t, err)

	var wire map[string]any
	require.NoError(t, json.Unmarshal(bytes, &wire))

	assert.Equal(t, "s3", wire["type"])
	assert.Equal(t, "testacc", wire["bucket"])
	for _, ghost := range []string{"filesystem", "account-name", "host", "shared-access-key"} {
		_, ok := wire[ghost]
		assert.Falsef(t, ok, "sibling-variant field %q leaked into round-tripped StorageProfile: %v", ghost, wire)
	}

	// Quick sanity: the wire form parses back into the same union with the
	// S3 variant populated and GCS/ADLS nil.
	var rt managementv1.StorageProfile
	require.NoError(t, json.Unmarshal(bytes, &rt))
	require.NotNil(t, rt.StorageProfileS3)
	assert.Nil(t, rt.StorageProfileGcs)
	assert.Nil(t, rt.StorageProfileAdls)
	assert.Equal(t, "s3", rt.StorageProfileS3.Type)
}

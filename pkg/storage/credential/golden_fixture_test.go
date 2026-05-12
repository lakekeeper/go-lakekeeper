package credential_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	managementv1 "github.com/lakekeeper/go-lakekeeper/pkg/apis/management/v1"
)

// Golden fixtures pulled verbatim from the spec's inline Rust examples in
// api/openapi/management-open-api.yaml (lines 6997–7128). These payloads are
// the authoritative wire-format reference: any conformant client must accept
// them on the unmarshal side and reproduce them on the marshal side.

const (
	goldenS3AccessKey = `{
		"type": "s3",
		"credential-type": "access-key",
		"access-key-id": "minio-root-user",
		"secret-access-key": "minio-root-password"
	}`

	goldenS3AwsSystemIdentity = `{
		"type": "s3",
		"credential-type": "aws-system-identity",
		"external-id": "ext-123"
	}`

	goldenS3CloudflareR2 = `{
		"type": "s3",
		"credential-type": "cloudflare-r2",
		"access-key-id": "r2-access",
		"secret-access-key": "r2-secret",
		"account-id": "cf-account",
		"token": "cf-token"
	}`

	goldenAzClientCredentials = `{
		"type": "az",
		"credential-type": "client-credentials",
		"client-id": "client",
		"client-secret": "secret",
		"tenant-id": "tenant"
	}`

	goldenAzSharedAccessKey = `{
		"type": "az",
		"credential-type": "shared-access-key",
		"key": "base64key"
	}`

	goldenAzManagedIdentity = `{
		"type": "az",
		"credential-type": "azure-system-identity"
	}`

	goldenGcsServiceAccountKey = `{
		"type": "gcs",
		"credential-type": "service-account-key",
		"key": {
			"type": "service_account",
			"project_id": "example-project-1234",
			"private_key_id": "....",
			"private_key": "-----BEGIN PRIVATE KEY-----\n.....\n-----END PRIVATE KEY-----\n",
			"client_email": "abc@example-project-1234.iam.gserviceaccount.com",
			"client_id": "123456789012345678901",
			"auth_uri": "https://accounts.google.com/o/oauth2/auth",
			"token_uri": "https://oauth2.googleapis.com/token",
			"auth_provider_x509_cert_url": "https://www.googleapis.com/oauth2/v1/certs",
			"client_x509_cert_url": "https://www.googleapis.com/robot/v1/metadata/x509/abc%example-project-1234.iam.gserviceaccount.com",
			"universe_domain": "googleapis.com"
		}
	}`

	goldenGcsSystemIdentity = `{
		"type": "gcs",
		"credential-type": "gcp-system-identity"
	}`
)

// assertJSONEqual compares two JSON byte slices for semantic equality
// (key order independent).
func assertJSONEqual(t *testing.T, want, got []byte) {
	t.Helper()
	var wantMap, gotMap map[string]any
	require.NoError(t, json.Unmarshal(want, &wantMap))
	require.NoError(t, json.Unmarshal(got, &gotMap))
	assert.Equal(t, wantMap, gotMap)
}

// assertNoExtraKeys verifies that the marshaled payload does not contain
// keys for variant fields that don't belong to this credential type.
func assertNoExtraKeys(t *testing.T, payload []byte, forbidden ...string) {
	t.Helper()
	var got map[string]any
	require.NoError(t, json.Unmarshal(payload, &got))
	for _, k := range forbidden {
		_, present := got[k]
		assert.Falsef(t, present, "payload should not contain key %q, got %v", k, got)
	}
}

func TestGoldenFixture_S3AccessKey_RoundTrip(t *testing.T) {
	t.Parallel()

	var c managementv1.StorageCredential
	require.NoError(t, json.Unmarshal([]byte(goldenS3AccessKey), &c))

	out, err := json.Marshal(c)
	require.NoError(t, err)

	assertJSONEqual(t, []byte(goldenS3AccessKey), out)
	assertNoExtraKeys(t, out, "account-id", "token", "external-id")
}

func TestGoldenFixture_S3AwsSystemIdentity_RoundTrip(t *testing.T) {
	t.Parallel()

	var c managementv1.StorageCredential
	require.NoError(t, json.Unmarshal([]byte(goldenS3AwsSystemIdentity), &c))

	out, err := json.Marshal(c)
	require.NoError(t, err)

	assertJSONEqual(t, []byte(goldenS3AwsSystemIdentity), out)
	assertNoExtraKeys(t, out, "access-key-id", "secret-access-key", "account-id", "token")
}

func TestGoldenFixture_S3CloudflareR2_RoundTrip(t *testing.T) {
	t.Parallel()

	var c managementv1.StorageCredential
	require.NoError(t, json.Unmarshal([]byte(goldenS3CloudflareR2), &c))

	out, err := json.Marshal(c)
	require.NoError(t, err)

	assertJSONEqual(t, []byte(goldenS3CloudflareR2), out)
	assertNoExtraKeys(t, out, "external-id")
}

func TestGoldenFixture_AzClientCredentials_RoundTrip(t *testing.T) {
	t.Parallel()

	var c managementv1.StorageCredential
	require.NoError(t, json.Unmarshal([]byte(goldenAzClientCredentials), &c))

	out, err := json.Marshal(c)
	require.NoError(t, err)

	assertJSONEqual(t, []byte(goldenAzClientCredentials), out)
	assertNoExtraKeys(t, out, "key")
}

func TestGoldenFixture_AzSharedAccessKey_RoundTrip(t *testing.T) {
	t.Parallel()

	var c managementv1.StorageCredential
	require.NoError(t, json.Unmarshal([]byte(goldenAzSharedAccessKey), &c))

	out, err := json.Marshal(c)
	require.NoError(t, err)

	assertJSONEqual(t, []byte(goldenAzSharedAccessKey), out)
	assertNoExtraKeys(t, out, "client-id", "client-secret", "tenant-id")
}

func TestGoldenFixture_AzManagedIdentity_RoundTrip(t *testing.T) {
	t.Parallel()

	var c managementv1.StorageCredential
	require.NoError(t, json.Unmarshal([]byte(goldenAzManagedIdentity), &c))

	out, err := json.Marshal(c)
	require.NoError(t, err)

	assertJSONEqual(t, []byte(goldenAzManagedIdentity), out)
	assertNoExtraKeys(t, out, "client-id", "client-secret", "tenant-id", "key")
}

func TestGoldenFixture_GcsServiceAccountKey_RoundTrip(t *testing.T) {
	t.Parallel()

	var c managementv1.StorageCredential
	require.NoError(t, json.Unmarshal([]byte(goldenGcsServiceAccountKey), &c))

	out, err := json.Marshal(c)
	require.NoError(t, err)

	assertJSONEqual(t, []byte(goldenGcsServiceAccountKey), out)
}

func TestGoldenFixture_GcsSystemIdentity_RoundTrip(t *testing.T) {
	t.Parallel()

	var c managementv1.StorageCredential
	require.NoError(t, json.Unmarshal([]byte(goldenGcsSystemIdentity), &c))

	out, err := json.Marshal(c)
	require.NoError(t, err)

	assertJSONEqual(t, []byte(goldenGcsSystemIdentity), out)
	assertNoExtraKeys(t, out, "key")
}

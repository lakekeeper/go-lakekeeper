package credential_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	managementv1 "github.com/lakekeeper/go-lakekeeper/pkg/apis/management/v1"
	"github.com/lakekeeper/go-lakekeeper/pkg/storage/credential"
)

func TestNewGCSServiceAccountKey(t *testing.T) {
	t.Parallel()

	key := managementv1.GcsServiceKey{
		Type:                    "service_account",
		ProjectId:               "my-project",
		PrivateKeyId:            "key-id",
		PrivateKey:              "----- BEGIN PRIVATE KEY -----\n...",
		ClientEmail:             "sa@my-project.iam.gserviceaccount.com",
		ClientId:                "1234567890",
		AuthUri:                 "https://accounts.google.com/o/oauth2/auth",
		TokenUri:                "https://oauth2.googleapis.com/token",
		AuthProviderX509CertUrl: "https://www.googleapis.com/oauth2/v1/certs",
		ClientX509CertUrl:       "https://...",
		UniverseDomain:          "googleapis.com",
	}

	c := credential.NewGCSServiceAccountKey(key)

	require.NotNil(t, c)
	assert.Equal(t, "service-account-key", c.CredentialType)
	assert.Equal(t, "gcs", c.Type)
	assert.Equal(t, key, c.Key)
}

func TestNewGCSSystemIdentity(t *testing.T) {
	t.Parallel()

	c := credential.NewGCSSystemIdentity()

	require.NotNil(t, c)
	assert.Equal(t, "gcp-system-identity", c.CredentialType)
	assert.Equal(t, "gcs", c.Type)
}

// TestNewGCSSystemIdentity_WireFormat documents the flattened-struct wire
// quirk for system-identity GCS credentials. See s3_test.go for the same
// pattern.
func TestNewGCSSystemIdentity_WireFormat(t *testing.T) {
	t.Parallel()

	c := credential.NewGCSSystemIdentity()

	data, err := json.Marshal(c)
	require.NoError(t, err)

	var got map[string]any
	require.NoError(t, json.Unmarshal(data, &got))

	assert.Equal(t, "gcs", got["type"])
	assert.Equal(t, "gcp-system-identity", got["credential-type"])
}

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

	data, err := json.Marshal(c)
	require.NoError(t, err)

	var got map[string]any
	require.NoError(t, json.Unmarshal(data, &got))

	assert.Equal(t, "gcs", got["type"])
	assert.Equal(t, "service-account-key", got["credential-type"])
	assert.NotNil(t, got["key"])
	gotKey, ok := got["key"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "my-project", gotKey["project_id"])
}

func TestNewGCSSystemIdentity(t *testing.T) {
	t.Parallel()

	c := credential.NewGCSSystemIdentity()

	data, err := json.Marshal(c)
	require.NoError(t, err)

	var got map[string]any
	require.NoError(t, json.Unmarshal(data, &got))

	assert.Equal(t, "gcs", got["type"])
	assert.Equal(t, "gcp-system-identity", got["credential-type"])
	assert.NotContains(t, got, "key")
}

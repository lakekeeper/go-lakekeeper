package credential_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lakekeeper/go-lakekeeper/pkg/storage/credential"
)

func TestNewAZClientCredentials(t *testing.T) {
	t.Parallel()

	c := credential.NewAZClientCredentials("tenant", "client", "secret")

	require.NotNil(t, c)
	assert.Equal(t, "tenant", c.TenantId)
	assert.Equal(t, "client", c.ClientId)
	assert.Equal(t, "secret", c.ClientSecret)
	assert.Equal(t, "client-credentials", c.CredentialType)
	assert.Equal(t, "az", c.Type)
}

func TestNewAZSharedAccessKey(t *testing.T) {
	t.Parallel()

	c := credential.NewAZSharedAccessKey("base64key")

	require.NotNil(t, c)
	assert.Equal(t, "base64key", c.Key)
	assert.Equal(t, "shared-access-key", c.CredentialType)
	assert.Equal(t, "az", c.Type)
}

func TestNewAZManagedIdentity(t *testing.T) {
	t.Parallel()

	c := credential.NewAZManagedIdentity()

	require.NotNil(t, c)
	assert.Equal(t, "azure-system-identity", c.CredentialType)
	assert.Equal(t, "az", c.Type)
}

// TestNewAZManagedIdentity_WireFormat documents the flattened-struct quirk for
// managed-identity AZ credentials. See s3_test.go for the same pattern.
func TestNewAZManagedIdentity_WireFormat(t *testing.T) {
	t.Parallel()

	c := credential.NewAZManagedIdentity()

	data, err := json.Marshal(c)
	require.NoError(t, err)

	var got map[string]any
	require.NoError(t, json.Unmarshal(data, &got))

	assert.Equal(t, "az", got["type"])
	assert.Equal(t, "azure-system-identity", got["credential-type"])
}

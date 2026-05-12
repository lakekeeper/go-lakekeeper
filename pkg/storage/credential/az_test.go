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

	data, err := json.Marshal(c)
	require.NoError(t, err)

	var got map[string]any
	require.NoError(t, json.Unmarshal(data, &got))

	assert.Equal(t, "az", got["type"])
	assert.Equal(t, "client-credentials", got["credential-type"])
	assert.Equal(t, "tenant", got["tenant-id"])
	assert.Equal(t, "client", got["client-id"])
	assert.Equal(t, "secret", got["client-secret"])
	assert.NotContains(t, got, "key")
}

func TestNewAZSharedAccessKey(t *testing.T) {
	t.Parallel()

	c := credential.NewAZSharedAccessKey("base64key")

	data, err := json.Marshal(c)
	require.NoError(t, err)

	var got map[string]any
	require.NoError(t, json.Unmarshal(data, &got))

	assert.Equal(t, "az", got["type"])
	assert.Equal(t, "shared-access-key", got["credential-type"])
	assert.Equal(t, "base64key", got["key"])
	assert.NotContains(t, got, "client-id")
	assert.NotContains(t, got, "client-secret")
	assert.NotContains(t, got, "tenant-id")
}

func TestNewAZManagedIdentity(t *testing.T) {
	t.Parallel()

	c := credential.NewAZManagedIdentity()

	data, err := json.Marshal(c)
	require.NoError(t, err)

	var got map[string]any
	require.NoError(t, json.Unmarshal(data, &got))

	assert.Equal(t, "az", got["type"])
	assert.Equal(t, "azure-system-identity", got["credential-type"])
	assert.NotContains(t, got, "client-id")
	assert.NotContains(t, got, "client-secret")
	assert.NotContains(t, got, "tenant-id")
	assert.NotContains(t, got, "key")
}

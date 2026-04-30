package credential_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lakekeeper/go-lakekeeper/pkg/storage/credential"
)

func TestNewS3AccessKey(t *testing.T) {
	t.Parallel()

	c := credential.NewS3AccessKey("AKIA...", "secret")

	data, err := json.Marshal(c)
	require.NoError(t, err)

	var got map[string]any
	require.NoError(t, json.Unmarshal(data, &got))

	assert.Equal(t, "s3", got["type"])
	assert.Equal(t, "access-key", got["credential-type"])
	assert.Equal(t, "AKIA...", got["access-key-id"])
	assert.Equal(t, "secret", got["secret-access-key"])
	assert.NotContains(t, got, "external-id")
	assert.NotContains(t, got, "account-id")
	assert.NotContains(t, got, "token")
}

func TestNewS3AccessKeyWithExternalID(t *testing.T) {
	t.Parallel()

	c := credential.NewS3AccessKeyWithExternalID("AKIA...", "secret", "ext-123")

	data, err := json.Marshal(c)
	require.NoError(t, err)

	var got map[string]any
	require.NoError(t, json.Unmarshal(data, &got))

	assert.Equal(t, "ext-123", got["external-id"])
}

func TestNewS3AwsSystemIdentity(t *testing.T) {
	t.Parallel()

	c := credential.NewS3AwsSystemIdentity()

	data, err := json.Marshal(c)
	require.NoError(t, err)

	var got map[string]any
	require.NoError(t, json.Unmarshal(data, &got))

	assert.Equal(t, "s3", got["type"])
	assert.Equal(t, "aws-system-identity", got["credential-type"])
	assert.NotContains(t, got, "access-key-id")
	assert.NotContains(t, got, "secret-access-key")
	assert.NotContains(t, got, "external-id")
}

func TestNewS3AwsSystemIdentityWithExternalID(t *testing.T) {
	t.Parallel()

	c := credential.NewS3AwsSystemIdentityWithExternalID("ext-456")

	data, err := json.Marshal(c)
	require.NoError(t, err)

	var got map[string]any
	require.NoError(t, json.Unmarshal(data, &got))

	assert.Equal(t, "ext-456", got["external-id"])
	assert.Equal(t, "aws-system-identity", got["credential-type"])
}

func TestNewS3CloudflareR2(t *testing.T) {
	t.Parallel()

	c := credential.NewS3CloudflareR2("ak", "sk", "acct-id", "tok")

	data, err := json.Marshal(c)
	require.NoError(t, err)

	var got map[string]any
	require.NoError(t, json.Unmarshal(data, &got))

	assert.Equal(t, "s3", got["type"])
	assert.Equal(t, "cloudflare-r2", got["credential-type"])
	assert.Equal(t, "ak", got["access-key-id"])
	assert.Equal(t, "sk", got["secret-access-key"])
	assert.Equal(t, "acct-id", got["account-id"])
	assert.Equal(t, "tok", got["token"])
	assert.NotContains(t, got, "external-id")
}

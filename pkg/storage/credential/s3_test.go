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

	require.NotNil(t, c)
	assert.Equal(t, "AKIA...", c.AccessKeyId)
	assert.Equal(t, "secret", c.SecretAccessKey)
	assert.Equal(t, "access-key", c.CredentialType)
	assert.Equal(t, "s3", c.Type)
}

func TestNewS3AwsSystemIdentity(t *testing.T) {
	t.Parallel()

	c := credential.NewS3AwsSystemIdentity()

	require.NotNil(t, c)
	assert.Equal(t, "aws-system-identity", c.CredentialType)
	assert.Equal(t, "s3", c.Type)
	assert.Empty(t, c.AccessKeyId)
	assert.Empty(t, c.SecretAccessKey)
}

func TestNewS3CloudflareR2(t *testing.T) {
	t.Parallel()

	c := credential.NewS3CloudflareR2("ak", "sk", "acct-id", "tok")

	require.NotNil(t, c)
	assert.Equal(t, "ak", c.AccessKeyId)
	assert.Equal(t, "sk", c.SecretAccessKey)
	assert.Equal(t, "acct-id", c.AccountId)
	assert.Equal(t, "tok", c.Token)
	assert.Equal(t, "cloudflare-r2", c.CredentialType)
	assert.Equal(t, "s3", c.Type)
}

// TestNewS3AccessKey_WireFormat documents what actually gets sent to
// Lakekeeper when an access-key credential is marshaled.
//
// The generated StorageCredentialS3 struct is a *flattened* representation of
// the access-key, AWS-system-identity, and Cloudflare-R2 oneOf variants — its
// MarshalJSON always emits all fields, so an access-key payload includes
// `account-id: ""` and `token: ""` even though they're irrelevant. If
// Lakekeeper later rejects this, the fix is a custom MarshalJSON in this
// package; this test will catch a generator behaviour change either way.
func TestNewS3AccessKey_WireFormat(t *testing.T) {
	t.Parallel()

	c := credential.NewS3AccessKey("ak", "sk")

	data, err := json.Marshal(c)
	require.NoError(t, err)

	var got map[string]any
	require.NoError(t, json.Unmarshal(data, &got))

	assert.Equal(t, "s3", got["type"])
	assert.Equal(t, "access-key", got["credential-type"])
	assert.Equal(t, "ak", got["access-key-id"])
	assert.Equal(t, "sk", got["secret-access-key"])
	// Flattened-struct quirk: irrelevant fields ship as empty strings.
	assert.Empty(t, got["account-id"])
	assert.Empty(t, got["token"])
}

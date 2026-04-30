package profile_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	managementv1 "github.com/lakekeeper/go-lakekeeper/pkg/apis/management/v1"
	"github.com/lakekeeper/go-lakekeeper/pkg/storage/profile"
)

func TestNewS3Profile_Defaults(t *testing.T) {
	t.Parallel()

	p := profile.NewS3Profile("my-bucket", "us-east-1")

	require.NotNil(t, p)
	assert.Equal(t, "my-bucket", p.Bucket)
	assert.Equal(t, "us-east-1", p.Region)
	assert.Equal(t, "s3", p.Type, "discriminator must be set so the StorageProfile oneOf marshaler picks the right variant")
	assert.False(t, p.StsEnabled, "STS is opt-in")
	assert.Nil(t, p.Endpoint)
	assert.Nil(t, p.KeyPrefix)
	assert.Nil(t, p.Flavor)
}

func TestNewS3Profile_AllOptionsApplied(t *testing.T) {
	t.Parallel()

	flavor := managementv1.S3FLAVOR_S3_COMPAT
	style := managementv1.S3URLSTYLEDETECTIONMODE_PATH

	p := profile.NewS3Profile("bucket", "eu-central-1",
		profile.WithS3Endpoint("https://minio:9000"),
		profile.WithS3KeyPrefix("warehouses/foo"),
		profile.WithS3Flavor(flavor),
		profile.WithS3PathStyleAccess(),
		profile.WithS3AlternativeProtocols(),
		profile.WithS3AssumeRoleARN("arn:aws:iam::123:role/assume"),
		profile.WithS3AWSKMSKeyARN("arn:aws:kms::123:key/abc"),
		profile.WithS3STSEnabled(),
		profile.WithS3STSRoleARN("arn:aws:iam::123:role/sts"),
		profile.WithS3STSEndpoint("https://sts.example.com"),
		profile.WithS3STSTokenValidity(2*time.Hour),
		profile.WithS3RemoteSigningURLStyle(style),
	)

	require.Equal(t, "https://minio:9000", *p.Endpoint)
	assert.Equal(t, "warehouses/foo", *p.KeyPrefix)
	assert.Equal(t, flavor, *p.Flavor)
	assert.True(t, *p.PathStyleAccess)
	assert.True(t, *p.AllowAlternativeProtocols)
	assert.Equal(t, "arn:aws:iam::123:role/assume", *p.AssumeRoleArn)
	assert.Equal(t, "arn:aws:kms::123:key/abc", *p.AwsKmsKeyArn)
	assert.True(t, p.StsEnabled)
	assert.Equal(t, "arn:aws:iam::123:role/sts", *p.StsRoleArn)
	assert.Equal(t, "https://sts.example.com", *p.StsEndpoint)
	assert.Equal(t, int64((2 * time.Hour).Seconds()), *p.StsTokenValiditySeconds)
	assert.Equal(t, style, *p.RemoteSigningUrlStyle)
}

func TestNewS3Profile_OptionsAreOverridable(t *testing.T) {
	t.Parallel()

	p := profile.NewS3Profile("b", "r",
		profile.WithS3Endpoint("first"),
		profile.WithS3Endpoint("second"),
	)
	assert.Equal(t, "second", *p.Endpoint, "later options must override earlier ones")
}

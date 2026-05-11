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

	sp := profile.NewS3Profile("my-bucket", "us-east-1")

	require.NotNil(t, sp.StorageProfileS3, "S3 builder must populate the S3 variant of the union")
	p := sp.StorageProfileS3
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

	flavor := managementv1.S3FlavorS3Compat
	style := managementv1.S3UrlStyleDetectionModePath

	tags := map[string]string{"env": "prod", "team": "data"}

	sp := profile.NewS3Profile("bucket", "eu-central-1",
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
		profile.WithS3PushS3DeleteDisabled(true),
		profile.WithS3LegacyMd5Behavior(true),
		profile.WithS3RemoteSigningEnabled(false),
		profile.WithS3StsSessionTags(tags),
	)

	require.NotNil(t, sp.StorageProfileS3)
	p := sp.StorageProfileS3
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
	assert.True(t, *p.PushS3DeleteDisabled)
	assert.True(t, *p.LegacyMd5Behavior)
	assert.False(t, *p.RemoteSigningEnabled)
	assert.Equal(t, tags, p.StsSessionTags)
}

func TestAsS3_OnS3Union(t *testing.T) {
	t.Parallel()

	sp := profile.NewS3Profile("b", "r")
	got, ok := profile.AsS3(sp)
	require.True(t, ok)
	require.NotNil(t, got)
	assert.Equal(t, "b", got.Bucket)

	// AsGCS / AsADLS on an S3 union must report the variant mismatch.
	_, ok = profile.AsGCS(sp)
	assert.False(t, ok)
	_, ok = profile.AsADLS(sp)
	assert.False(t, ok)
}

func TestAsS3_OnNonS3Union(t *testing.T) {
	t.Parallel()

	gcs := profile.NewGCSProfile("g")
	_, ok := profile.AsS3(gcs)
	assert.False(t, ok, "AsS3 must return false on a non-S3 union")
}

func TestNewS3Profile_OptionsAreOverridable(t *testing.T) {
	t.Parallel()

	sp := profile.NewS3Profile("b", "r",
		profile.WithS3Endpoint("first"),
		profile.WithS3Endpoint("second"),
	)
	require.NotNil(t, sp.StorageProfileS3)
	assert.Equal(t, "second", *sp.StorageProfileS3.Endpoint, "later options must override earlier ones")
}

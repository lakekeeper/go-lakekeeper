package profile_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lakekeeper/go-lakekeeper/pkg/storage/profile"
)

func TestNewGCSProfile_Defaults(t *testing.T) {
	t.Parallel()

	p := profile.NewGCSProfile("my-gcs-bucket")

	require.NotNil(t, p)
	assert.Equal(t, "my-gcs-bucket", p.Bucket)
	assert.Equal(t, "gcs", p.Type)
	assert.Nil(t, p.KeyPrefix)
	assert.Nil(t, p.StsEnabled, "spec default is true server-side; nil here means 'use server default'")
}

func TestNewGCSProfile_OptionsApplied(t *testing.T) {
	t.Parallel()

	p := profile.NewGCSProfile("bucket",
		profile.WithGCSKeyPrefix("warehouses/foo"),
		profile.WithGCSSTSEnabled(false),
	)

	assert.Equal(t, "warehouses/foo", *p.KeyPrefix)
	require.NotNil(t, p.StsEnabled)
	assert.False(t, *p.StsEnabled)
}

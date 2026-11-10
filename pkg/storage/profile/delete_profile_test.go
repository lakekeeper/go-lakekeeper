package profile_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	managementv1 "github.com/lakekeeper/go-lakekeeper/pkg/apis/management/v1"
	"github.com/lakekeeper/go-lakekeeper/pkg/storage/profile"
)

func TestNewHardDeleteProfile_WireFormat(t *testing.T) {
	t.Parallel()

	b, err := json.Marshal(profile.NewHardDeleteProfile())

	require.NoError(t, err)
	assert.JSONEq(t, `{"type":"hard"}`, string(b))
}

func TestNewSoftDeleteProfile_WireFormat(t *testing.T) {
	t.Parallel()

	b, err := json.Marshal(profile.NewSoftDeleteProfile(time.Hour))

	require.NoError(t, err)
	assert.JSONEq(t, `{"type":"soft","expiration-seconds":3600}`, string(b))
}

func TestNewSoftDeleteProfile_TruncatesToSeconds(t *testing.T) {
	t.Parallel()

	soft, ok := profile.AsSoftDeleteProfile(profile.NewSoftDeleteProfile(90*time.Second + 999*time.Millisecond))

	require.True(t, ok)
	assert.Equal(t, int64(90), soft.GetExpirationSeconds())
}

func TestDeleteProfileAccessors(t *testing.T) {
	t.Parallel()

	hard := profile.NewHardDeleteProfile()
	soft := profile.NewSoftDeleteProfile(time.Minute)

	h, ok := profile.AsHardDeleteProfile(hard)
	require.True(t, ok)
	assert.Equal(t, profile.DeleteProfileTypeHard, h.GetType())
	_, ok = profile.AsSoftDeleteProfile(hard)
	assert.False(t, ok)

	s, ok := profile.AsSoftDeleteProfile(soft)
	require.True(t, ok)
	assert.Equal(t, profile.DeleteProfileTypeSoft, s.GetType())
	assert.Equal(t, int64(60), s.GetExpirationSeconds())
	_, ok = profile.AsHardDeleteProfile(soft)
	assert.False(t, ok)
}

func TestDeleteProfileAccessors_DecodedFromServer(t *testing.T) {
	t.Parallel()

	var dp managementv1.TabularDeleteProfile
	require.NoError(t, json.Unmarshal([]byte(`{"type":"soft","expiration-seconds":120}`), &dp))

	s, ok := profile.AsSoftDeleteProfile(dp)
	require.True(t, ok, "the generated decoder must pick the soft variant from the discriminator")
	assert.Equal(t, int64(120), s.GetExpirationSeconds())
}

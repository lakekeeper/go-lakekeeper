package profile_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	managementv1 "github.com/lakekeeper/go-lakekeeper/pkg/apis/management/v1"
	"github.com/lakekeeper/go-lakekeeper/pkg/storage/profile"
)

// unmarshalProfile decodes a server-style StorageProfile, the way a warehouse
// response reaches the accessors.
func unmarshalProfile(t *testing.T, raw string) managementv1.StorageProfile {
	t.Helper()
	var sp managementv1.StorageProfile
	require.NoError(t, json.Unmarshal([]byte(raw), &sp))
	return sp
}

func TestAsMemory(t *testing.T) {
	t.Parallel()

	sp := unmarshalProfile(t, `{"type":"memory","base-location":"memory://wh"}`)
	got, ok := profile.AsMemory(sp)
	require.True(t, ok)
	assert.Equal(t, "memory://wh", got.BaseLocation)
	assert.Equal(t, "memory", profile.StorageProfileTypeMemory)

	_, ok = profile.AsMemory(profile.NewS3Profile("b", "r"))
	assert.False(t, ok, "AsMemory must return false on a non-memory union")
}

func TestAsOneLake(t *testing.T) {
	t.Parallel()

	sp := unmarshalProfile(t, `{"type":"onelake","workspace-id":"ws","lakehouse-id":"lh"}`)
	got, ok := profile.AsOneLake(sp)
	require.True(t, ok)
	assert.Equal(t, "ws", got.WorkspaceId)
	assert.Equal(t, "lh", got.LakehouseId)
	assert.Equal(t, "onelake", profile.StorageProfileTypeOneLake)

	_, ok = profile.AsOneLake(profile.NewADLSProfile("acct", "fs"))
	assert.False(t, ok, "AsOneLake must return false on a non-OneLake union")
}

func TestAsStackit(t *testing.T) {
	t.Parallel()

	sp := unmarshalProfile(t, `{"type":"stackit","bucket":"b","region":"eu01"}`)
	got, ok := profile.AsStackit(sp)
	require.True(t, ok)
	assert.Equal(t, "b", got.Bucket)
	assert.Equal(t, "eu01", got.Region)
	assert.Equal(t, "stackit", profile.StorageProfileTypeStackit)

	_, ok = profile.AsStackit(profile.NewS3Profile("b", "r"))
	assert.False(t, ok, "AsStackit must return false on a non-STACKIT union")
}

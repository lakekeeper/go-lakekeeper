package profile_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lakekeeper/go-lakekeeper/pkg/storage/profile"
)

func TestNewADLSProfile_Defaults(t *testing.T) {
	t.Parallel()

	p := profile.NewADLSProfile("acct", "fs")

	require.NotNil(t, p)
	assert.Equal(t, "acct", p.AccountName)
	assert.Equal(t, "fs", p.Filesystem)
	assert.Equal(t, "adls", p.Type)
	assert.Nil(t, p.KeyPrefix)
	assert.Nil(t, p.AuthorityHost)
	assert.Nil(t, p.SasEnabled)
}

func TestNewADLSProfile_OptionsApplied(t *testing.T) {
	t.Parallel()

	p := profile.NewADLSProfile("acct", "fs",
		profile.WithADLSKeyPrefix("warehouses/foo"),
		profile.WithADLSAuthorityHost("https://login.microsoftonline.de"),
		profile.WithADLSHost("dfs.core.windows.de"),
		profile.WithADLSAlternativeProtocols(),
		profile.WithADLSSASEnabled(false),
		profile.WithADLSSASTokenValidity(30*time.Minute),
	)

	assert.Equal(t, "warehouses/foo", *p.KeyPrefix)
	assert.Equal(t, "https://login.microsoftonline.de", *p.AuthorityHost)
	assert.Equal(t, "dfs.core.windows.de", *p.Host)
	assert.True(t, *p.AllowAlternativeProtocols)
	assert.False(t, *p.SasEnabled)
	assert.Equal(t, int64(1800), *p.SasTokenValiditySeconds)
}

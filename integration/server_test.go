//go:build integration

package integration

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	managementv1 "github.com/lakekeeper/go-lakekeeper/pkg/apis/management/v1"
)

func TestServer_Info(t *testing.T) {
	c := sharedClient

	info, _, err := c.ServerAPI.GetServerInfo(t.Context()).Execute()
	require.NoError(t, err)
	assert.True(t, info.Bootstrapped)
	assert.NotEmpty(t, info.ServerId)
	assert.True(t, info.DefaultProjectId.IsSet())
	assert.NotEmpty(t, *info.DefaultProjectId.Get())
	assert.Equal(t, "openfga", info.AuthzBackend)
}

// TestServer_Bootstrap_RejectsRebootstrap confirms that the server rejects a
// repeat Bootstrap call once it has already been bootstrapped. The "already
// bootstrapped" precondition is established in TestMain via the
// `client.WithInitialBootstrap` option, so this test relies on TestMain
// having run; do not run it in isolation against a fresh stack.
//
// This catches a regression where the server would silently accept the
// second call and risk overwriting state.
func TestServer_Bootstrap_RejectsRebootstrap(t *testing.T) {
	c := sharedClient

	req := managementv1.NewBootstrapRequest(true)
	r, err := c.ServerAPI.Bootstrap(t.Context()).BootstrapRequest(*req).Execute()
	require.Error(t, err)
	require.NotNil(t, r)
	assert.Equal(t, http.StatusBadRequest, r.StatusCode)
}

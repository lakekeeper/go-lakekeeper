//go:build integration

package integration

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	managementv1 "github.com/lakekeeper/go-lakekeeper/pkg/apis/management/v1"
	"github.com/lakekeeper/go-lakekeeper/pkg/client"
)

func TestServer_Info(t *testing.T) {
	c := sharedClient

	info, err := c.Server.Info(t.Context())
	require.NoError(t, err)
	assert.True(t, info.GetBootstrapped())
	assert.NotEmpty(t, info.GetServerId())
	assert.NotEmpty(t, info.GetDefaultProjectId())
	assert.Equal(t, "openfga", info.GetAuthzBackend())
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

	err := c.Server.Bootstrap(t.Context(), managementv1.NewBootstrapRequest(true))
	var apiErr *client.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, http.StatusBadRequest, apiErr.StatusCode)
}

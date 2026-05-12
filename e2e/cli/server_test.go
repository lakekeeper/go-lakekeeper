//go:build e2e_cli

package clie2e

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServerInfoJSON(t *testing.T) {
	t.Parallel()

	out := runOK(t, "server", "info", "--output", "json")

	var info struct {
		ServerID     string `json:"server-id"`
		Version      string `json:"version"`
		Bootstrapped bool   `json:"bootstrapped"`
		AuthzBackend string `json:"authz-backend"`
	}
	decodeJSON(t, out, &info)
	assert.NotEmpty(t, info.ServerID)
	assert.NotEmpty(t, info.Version)
	assert.True(t, info.Bootstrapped, "server should be bootstrapped after TestMain setup")
}

func TestServerInfoText(t *testing.T) {
	t.Parallel()

	out := runOK(t, "server", "info")
	require.Contains(t, string(out), "ID:")
	require.Contains(t, string(out), "Bootstrapped: true")
}

// TestServerBootstrapRejectsRebootstrap confirms the server rejects a second
// bootstrap call after TestMain's initial bootstrap. Mirrors the SDK-level
// assertion in integration/server_test.go (TestServer_Bootstrap_RejectsRebootstrap).
//
// Intentionally NOT parallel: bootstrap is server-wide; running concurrently
// with other tests creates a race on the bootstrap state.
func TestServerBootstrapRejectsRebootstrap(t *testing.T) {
	_, _, code := runFail(t, "server", "bootstrap", "--accept-terms-of-use")
	assert.NotZero(t, code, "expected re-bootstrap to fail post-initial-bootstrap")
}

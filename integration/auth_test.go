//go:build integration

// Coverage for the non-OAuth2 core.AuthSource implementations against the
// live docker-compose stack.
//
// The K8s test exercises the *client-side* file-read + Bearer-injection path
// only — the stack has no Kubernetes API server, so Lakekeeper validates the
// token as a Keycloak OIDC JWT, not via Kubernetes TokenReview. Standing up
// a real kube control plane in docker-compose is not justified for a single
// assertion; if Lakekeeper-side TokenReview ever needs end-to-end coverage,
// it belongs in a separate kind/k3s harness.

package integration

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lakekeeper/go-lakekeeper/pkg/client"
	"github.com/lakekeeper/go-lakekeeper/pkg/core"
)

func TestIntegrationAccessTokenAuthSource(t *testing.T) {
	t.Parallel()

	c, err := client.NewWithAuthSource(
		t.Context(),
		os.Getenv("LAKEKEEPER_BASE_URL"),
		&core.AccessTokenAuthSource{Token: freshKeycloakToken(t)},
	)
	require.NoError(t, err)

	info, _, err := c.ServerAPI.GetServerInfo(t.Context()).Execute()
	require.NoError(t, err)
	assert.NotEmpty(t, info.ServerId)
	assert.True(t, info.Bootstrapped)
}

func TestIntegrationK8sServiceAccountAuthSource(t *testing.T) {
	t.Parallel()

	tokenPath := filepath.Join(t.TempDir(), "token")
	require.NoError(t, os.WriteFile(tokenPath, []byte(freshKeycloakToken(t)), 0o600))

	c, err := client.NewWithAuthSource(
		t.Context(),
		os.Getenv("LAKEKEEPER_BASE_URL"),
		&core.K8sServiceAccountAuthSource{ServiceAccountTokenPath: &tokenPath},
	)
	require.NoError(t, err)

	info, _, err := c.ServerAPI.GetServerInfo(t.Context()).Execute()
	require.NoError(t, err)
	assert.NotEmpty(t, info.ServerId)
	assert.True(t, info.Bootstrapped)
}

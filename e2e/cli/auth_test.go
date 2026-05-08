//go:build e2e_cli

// `whoami` smoke across every supported auth mode. Validates that cobra flag
// wiring composes with the underlying core.AuthSource for each mode and that
// the resulting bearer is accepted by the server.
//
// Compose covers all three modes — for k8s mode it points --k8s-token-path
// at a temp file containing a Keycloak JWT, mirroring the existing
// integration cli_test pattern (see integration/cli_test.go for rationale).
//
// Kind exercises k8s mode against a *real* projected SA token mounted at the
// default path — that's the only thing kind buys us over compose. The token
// path flag is omitted so the AuthSource picks up its built-in default.

package clie2e

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWhoamiOAuth2(t *testing.T) {
	t.Parallel()

	args := append(authFlagsOAuth2(), "whoami", "--output", "json")
	stdout, stderr, code := runRaw(t, nil, args...)
	require.Equalf(t, 0, code, "exit %d\nstderr: %s", code, stderr)

	var user struct {
		ID string `json:"id"`
	}
	decodeJSON(t, stdout, &user)
	assert.NotEmpty(t, user.ID)
}

func TestWhoamiAccessToken(t *testing.T) {
	t.Parallel()

	args := append(authFlagsAccessToken(t), "whoami", "--output", "json")
	stdout, stderr, code := runRaw(t, nil, args...)
	require.Equalf(t, 0, code, "exit %d\nstderr: %s", code, stderr)

	var user struct {
		ID string `json:"id"`
	}
	decodeJSON(t, stdout, &user)
	assert.NotEmpty(t, user.ID)
}

func TestWhoamiK8sServiceAccount(t *testing.T) {
	t.Parallel()

	args := []string{
		"--base-url", os.Getenv("LAKEKEEPER_BASE_URL"),
		"--auth-mode", "k8s",
	}

	switch activeBackend.Name() {
	case BackendCompose:
		// Compose has no real Kubernetes; we point --k8s-token-path
		// at a Keycloak JWT and the server accepts it as OIDC.
		tokenPath := filepath.Join(t.TempDir(), "token")
		require.NoError(t, os.WriteFile(tokenPath, []byte(freshKeycloakToken(t)), 0o600))
		args = append(args, "--k8s-token-path", tokenPath)
	case BackendKind:
		// Kind: the projected SA token sits at the default path
		// inside the pod — no flag needed.
	}

	args = append(args, "whoami", "--output", "json")
	stdout, stderr, code := runRaw(t, nil, args...)
	require.Equalf(t, 0, code, "exit %d\nstderr: %s", code, stderr)

	var user struct {
		ID string `json:"id"`
	}
	decodeJSON(t, stdout, &user)
	assert.NotEmpty(t, user.ID)
}

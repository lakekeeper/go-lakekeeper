//go:build integration

// Subprocess coverage for the new lkctl auth-mode flags. Each test spawns
// the freshly-built lkctl binary and runs `whoami` against the live stack,
// mirroring what an operator does on the command line.
//
// SDK-level coverage of the underlying core.AuthSource implementations
// lives in auth_test.go; these tests pin the cobra flag wiring on top of
// that.

package integration

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	lkctlBinaryOnce sync.Once
	lkctlBinaryPath string
	lkctlBuildDir   string
	lkctlBuildErr   error
)

// lkctlBinary builds cmd/lkctl once per test run and returns the resulting
// binary path. The temp dir is captured in lkctlBuildDir so TestMain can
// remove it after m.Run() — sync.Once rules out t.TempDir, which would tie
// the build artefact to whichever test called lkctlBinary first.
func lkctlBinary(t *testing.T) string {
	t.Helper()
	lkctlBinaryOnce.Do(func() {
		dir, err := os.MkdirTemp("", "lkctl-int-")
		if err != nil {
			lkctlBuildErr = err
			return
		}
		lkctlBuildDir = dir
		bin := filepath.Join(dir, "lkctl")
		out, err := exec.Command("go", "build", "-o", bin, "../cmd").CombinedOutput()
		if err != nil {
			lkctlBuildErr = &buildError{err: err, output: string(out)}
			return
		}
		lkctlBinaryPath = bin
	})
	if lkctlBuildErr != nil {
		t.Fatalf("build lkctl: %v", lkctlBuildErr)
	}
	return lkctlBinaryPath
}

type buildError struct {
	err    error
	output string
}

func (e *buildError) Error() string {
	return e.err.Error() + "\n" + e.output
}

// runLkctl execs lkctl with the given args, returning combined stdout+stderr.
// On non-zero exit, t.Fatalf so the test fails with the captured output —
// debugging a CLI failure without seeing the output is painful.
func runLkctl(t *testing.T, args ...string) []byte {
	t.Helper()
	cmd := exec.CommandContext(t.Context(), lkctlBinary(t), args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("lkctl %v failed: %v\n%s", redactArgs(args), err, out)
	}
	return out
}

// redactArgs scrubs values following sensitive flags so bearer tokens don't
// leak into CI test logs on failure.
func redactArgs(args []string) []string {
	out := append([]string(nil), args...)
	for i := 0; i < len(out)-1; i++ {
		if out[i] == "--access-token" {
			out[i+1] = "<redacted>"
		}
	}
	return out
}

func TestCLIAccessTokenWhoami(t *testing.T) {
	t.Parallel()

	out := runLkctl(t,
		"whoami",
		"--output", "json",
		"--base-url", os.Getenv("LAKEKEEPER_BASE_URL"),
		"--auth-mode", "token",
		"--access-token", freshKeycloakToken(t),
	)

	var user struct {
		ID string `json:"id"`
	}
	require.NoError(t, json.Unmarshal(out, &user), "decode lkctl output: %s", out)
	assert.NotEmpty(t, user.ID)
}

func TestCLIK8sServiceAccountWhoami(t *testing.T) {
	t.Parallel()

	tokenPath := filepath.Join(t.TempDir(), "token")
	require.NoError(t, os.WriteFile(tokenPath, []byte(freshKeycloakToken(t)), 0o600))

	out := runLkctl(t,
		"whoami",
		"--output", "json",
		"--base-url", os.Getenv("LAKEKEEPER_BASE_URL"),
		"--auth-mode", "k8s",
		"--k8s-token-path", tokenPath,
	)

	var user struct {
		ID string `json:"id"`
	}
	require.NoError(t, json.Unmarshal(out, &user), "decode lkctl output: %s", out)
	assert.NotEmpty(t, user.ID)
}

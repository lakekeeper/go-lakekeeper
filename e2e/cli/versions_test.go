//go:build e2e_cli && e2e_versions

package clie2e

import (
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Supported-version tests carry the extra build tag e2e_versions and the
// name prefix TestVersions; e2e/versions/run.sh runs them against Lakekeeper
// pinned to the minor before the Lakekeeper spec release (the oldest
// supported minor) and to the minor before that (outside the range).

// versionWarning is the line lkctl prints to stderr for a server outside the
// supported range.
var versionWarning = regexp.MustCompile(
	`^Warning: server is Lakekeeper \d+\.\d+\.\d+\S*; this lkctl supports Lakekeeper \d+\.\d+\.x and \d+\.\d+\.x$`)

// runAdmin runs lkctl with admin credentials and returns stdout, stderr and
// the exit code, whatever the exit code.
func runAdmin(t *testing.T, args ...string) (string, string, int) {
	t.Helper()
	stdout, stderr, code := runRaw(t, nil, append(backendAdminFlags(), args...)...)
	return string(stdout), string(stderr), code
}

func TestVersionsPreviousMinorIsSupportedWithoutWarning(t *testing.T) {
	requireBackend(t, BackendComposePreviousMinor)
	t.Parallel()

	stdout, stderr, code := runAdmin(t, "server", "info", "--output", "json")

	require.Zero(t, code, "stderr: %s", stderr)
	var info struct {
		Version string `json:"version"`
	}
	decodeJSON(t, []byte(stdout), &info)
	assert.NotEmpty(t, info.Version)
	assert.NotContains(t, stderr, "Warning", "a server in the supported range gets no warning")
}

func TestVersionsPreviousMinorBootstrapAndInfoGetNoWarning(t *testing.T) {
	requireBackend(t, BackendComposePreviousMinor)
	t.Parallel()

	// --bootstrap reads the server info before server info reads it again.
	_, stderr, code := runAdmin(t, "--bootstrap", "server", "info")

	require.Zero(t, code, "stderr: %s", stderr)
	assert.NotContains(t, stderr, "Warning")
}

// The SDK cannot decode the server info of the older minor (it lacks
// properties the spec requires), so server info fails there as it does
// without the check; the warning, printed once before the error, says why
// such failures are to be expected.
func TestVersionsOlderMinorWarnsOnceBeforeTheCommandsOwnResult(t *testing.T) {
	requireBackend(t, BackendComposeOlderMinor)
	t.Parallel()

	stdout, stderr, code := runAdmin(t, "server", "info", "--output", "json")

	assert.Equal(t, 1, code, "the command's own exit code")
	assert.Empty(t, stdout)
	lines := strings.Split(strings.TrimSpace(stderr), "\n")
	require.Len(t, lines, 2, "the warning and the command's error; stderr: %s", stderr)
	assert.Regexp(t, versionWarning, lines[0])
	assert.Regexp(t, `^Error: get server info: decode 200 OK response: `, lines[1])
}

// A command that reads no server info sends no extra request for the check,
// so it prints no warning even against a server outside the range.
func TestVersionsCommandWithoutServerInfoGetsNoWarning(t *testing.T) {
	requireBackend(t, BackendComposeOlderMinor)
	t.Parallel()

	_, stderr, code := runAdmin(t, "project", "list")

	require.Zero(t, code, "stderr: %s", stderr)
	assert.NotContains(t, stderr, "Warning")
}

package commands

import (
	"bytes"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lakekeeper/go-lakekeeper/pkg/client"
	"github.com/lakekeeper/go-lakekeeper/pkg/testutil"
)

// runLkctlSplit is runLkctl with stdout and stderr kept apart; extra global
// flags go first in args.
func runLkctlSplit(t *testing.T, baseURL string, args ...string) (string, string, error) {
	t.Helper()
	cmd := NewCommand()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs(append([]string{
		"--base-url", baseURL,
		"--auth-mode", authModeToken,
		"--access-token", "test-token",
	}, args...))
	cmd.SetContext(t.Context())
	err := cmd.Execute()
	return stdout.String(), stderr.String(), err
}

// infoWithVersion is serverTestInfoBody on another Lakekeeper version.
func infoWithVersion(version string) string {
	return strings.ReplaceAll(serverTestInfoBody, `"0.15.0"`, `"`+version+`"`)
}

// inRangeVersion is a version on the newest supported minor.
func inRangeVersion() string {
	return client.SupportedVersions(client.EditionLakekeeper).Newest + ".0"
}

func outsideRangeWarning(version string) string {
	return "Warning: server is Lakekeeper " + version + "; this lkctl supports " +
		client.SupportedVersions(client.EditionLakekeeper).String() + "\n"
}

func serverInfoRun(t *testing.T, version string, args ...string) (string, string, error) {
	t.Helper()
	mux, baseURL := testutil.NewServer(t)
	testutil.Respond(t, mux, "GET /management/v1/info", http.StatusOK, infoWithVersion(version))
	return runLkctlSplit(t, baseURL, args...)
}

func TestVersionWarning_OutsideRangeWarnsOnStderrAndCarriesOn(t *testing.T) {
	t.Parallel()

	wantOut, wantErrOut, err := serverInfoRun(t, inRangeVersion(), "server", "info", "-o", "json")
	require.NoError(t, err)
	require.Empty(t, wantErrOut, "no warning in range")

	out, errOut, err := serverInfoRun(t, "0.1.0", "server", "info", "-o", "json")

	require.NoError(t, err, "the warning does not fail the command")
	assert.Equal(t, outsideRangeWarning("0.1.0"), errOut)
	assert.Equal(t,
		strings.ReplaceAll(wantOut, `"`+inRangeVersion()+`"`, `"0.1.0"`), out,
		"stdout is the command's own output only")
}

func TestVersionWarning_NewerThanRangeWarnsToo(t *testing.T) {
	t.Parallel()

	_, errOut, err := serverInfoRun(t, "99.0.0", "server", "info")

	require.NoError(t, err)
	assert.Equal(t, outsideRangeWarning("99.0.0"), errOut)
}

// --bootstrap reads the server info, then server info reads it again: one
// warning per run.
func TestVersionWarning_OncePerRun(t *testing.T) {
	t.Parallel()

	_, errOut, err := serverInfoRun(t, "0.1.0", "--bootstrap", "server", "info")

	require.NoError(t, err)
	assert.Equal(t, outsideRangeWarning("0.1.0"), errOut)
}

// An older server's info the SDK cannot decode still names its version: the
// warning comes before the command's error, which is unchanged.
func TestVersionWarning_BeforeTheErrorOfAnUndecodableServerInfo(t *testing.T) {
	t.Parallel()
	mux, baseURL := testutil.NewServer(t)
	testutil.Respond(t, mux, "GET /management/v1/info", http.StatusOK,
		`{"version":"0.13.9","lakekeeper-version":"0.13.9","bootstrapped":true}`)

	_, errOut, err := runLkctlSplit(t, baseURL, "server", "info")

	require.ErrorContains(t, err, "get server info: decode 200 OK response")
	assert.Equal(t, outsideRangeWarning("0.13.9"), errOut)
}

// --bootstrap against a server whose info the SDK cannot decode fails, and
// still warns first.
func TestVersionWarning_BeforeAFailedBootstrap(t *testing.T) {
	t.Parallel()
	mux, baseURL := testutil.NewServer(t)
	testutil.Respond(t, mux, "GET /management/v1/info", http.StatusOK,
		`{"version":"0.13.9","lakekeeper-version":"0.13.9","bootstrapped":true}`)

	_, errOut, err := runLkctlSplit(t, baseURL, "--bootstrap", "project", "list")

	require.ErrorContains(t, err, "bootstrap: get server info: decode 200 OK response")
	assert.Equal(t, outsideRangeWarning("0.13.9"), errOut)
}

// The check adds no request: a command that reads no server info does not
// warn.
func TestVersionWarning_NoServerInfoRequestOfItsOwn(t *testing.T) {
	t.Parallel()
	mux, baseURL := testutil.NewServer(t)
	testutil.Respond(t, mux, "GET /management/v1/project-list", http.StatusOK, `{"projects":[]}`)
	info := testutil.Respond(t, mux, "GET /management/v1/info", http.StatusOK, infoWithVersion("0.1.0"))

	_, errOut, err := runLkctlSplit(t, baseURL, "project", "list")

	require.NoError(t, err)
	assert.Empty(t, errOut)
	assert.Zero(t, info.Len(), "server info requests")
}

package commands

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lakekeeper/go-lakekeeper/pkg/client"
	"github.com/lakekeeper/go-lakekeeper/pkg/testutil"
	"github.com/lakekeeper/go-lakekeeper/pkg/version"
)

func TestPrintClientVersionShort(t *testing.T) {
	t.Parallel()

	v := version.Version{Version: "v1.2.3"}
	got := printClientVersion(&v, true)
	assert.Equal(t, "lkctl: v1.2.3\n", got)
}

func TestPrintServerVersion(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "lakekeeper: 0.9.1\n", printServerVersion("0.9.1"))
}

func TestPrintClientVersionLong(t *testing.T) {
	t.Parallel()

	v := version.Version{
		Version:      "v1.2.3",
		BuildDate:    "2026-04-30",
		GitCommit:    "deadbeef",
		GitTreeState: "clean",
		GitTag:       "v1.2.3",
		GoVersion:    "go1.24",
		Compiler:     "gc",
		Platform:     "darwin/arm64",
	}
	got := printClientVersion(&v, false)
	assert.Contains(t, got, "lkctl: ")
	assert.Contains(t, got, "BuildDate: 2026-04-30")
	assert.Contains(t, got, "GitCommit: deadbeef")
	assert.Contains(t, got, "GitTreeState: clean")
	assert.Contains(t, got, "GitTag: v1.2.3")
	assert.Contains(t, got, "GoVersion: go1.24")
	assert.Contains(t, got, "Platform: darwin/arm64")
}

func TestPrintClientVersionLongOmitsEmptyGitTag(t *testing.T) {
	t.Parallel()

	v := version.Version{
		Version:      "v1.2.3+abcdef.dirty",
		BuildDate:    "2026-04-30",
		GitCommit:    "abcdef",
		GitTreeState: "dirty",
		// GitTag intentionally empty — dev build, never tagged.
		GoVersion: "go1.24",
		Compiler:  "gc",
		Platform:  "darwin/arm64",
	}
	got := printClientVersion(&v, false)
	assert.NotContains(t, got, "GitTag")
}

func TestVersion_PrintsServerVersion(t *testing.T) {
	t.Parallel()
	mux, baseURL := testutil.NewServer(t)
	testutil.Respond(t, mux, "GET /management/v1/info", http.StatusOK, serverTestInfoBody)

	out, err := runLkctl(t, baseURL, "version", "-o", "json", "--short")

	require.NoError(t, err)
	assert.Contains(t, out, `"lakekeeper": "0.15.0"`)
}

func TestVersion_ServerErrorKeepsStatus(t *testing.T) {
	t.Parallel()
	mux, baseURL := testutil.NewServer(t)
	testutil.Respond(t, mux, "GET /management/v1/info", http.StatusForbidden,
		`{"error":{"code":403,"message":"Forbidden","type":"NotAuthorized","stack":[]}}`)

	_, err := runLkctl(t, baseURL, "version")

	require.ErrorContains(t, err, "get server info: 403 Forbidden")
	var apiErr *client.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, http.StatusForbidden, apiErr.StatusCode)
}

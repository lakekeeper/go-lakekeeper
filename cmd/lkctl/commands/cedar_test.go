package commands

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lakekeeper/go-lakekeeper/pkg/client"
	"github.com/lakekeeper/go-lakekeeper/pkg/testutil"
)

// noServer returns the URL of a test server that fails the test on any
// request.
func noServer(t *testing.T) string {
	t.Helper()
	mux, baseURL := testutil.NewServer(t)
	mux.HandleFunc("/", func(_ http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
	})
	return baseURL
}

func TestCedarHelp_MarksLakekeeperPlusWithoutContactingServer(t *testing.T) {
	t.Parallel()
	baseURL := noServer(t)

	cedarHelp, err := runLkctl(t, baseURL, "cedar", "--help")
	require.NoError(t, err)
	rootHelp, err := runLkctl(t, baseURL, "--help")
	require.NoError(t, err)

	assert.Contains(t, cedarHelp, "(Lakekeeper+)")
	assert.Regexp(t, `schema\s+.*\(Lakekeeper\+\)`, cedarHelp)
	assert.Regexp(t, `cedar\s+.*\(Lakekeeper\+\)`, rootHelp)
}

func TestCedarSchema_PrintsFragmentsByPath(t *testing.T) {
	t.Parallel()
	mux, baseURL := testutil.NewServer(t)
	testutil.Respond(t, mux, "GET /management/v1/permissions/cedar/schema", http.StatusOK,
		`{"schema-fragments":{"/b.cedarschema":"entity B;","/a.cedarschema":"entity A;\n"}}`)

	out, err := runLkctl(t, baseURL, "cedar", "schema")

	require.NoError(t, err)
	assert.Equal(t, "// /a.cedarschema\nentity A;\n\n// /b.cedarschema\nentity B;\n", out)
}

func TestCedarSchema_OnLakekeeperIsUnsupported(t *testing.T) {
	t.Parallel()
	mux, baseURL := testutil.NewServer(t)
	testutil.Respond(t, mux, "GET /management/v1/permissions/cedar/schema", http.StatusNotFound, nil)
	testutil.Respond(t, mux, "GET /management/v1/info", http.StatusOK, serverTestInfoBody)

	_, err := runLkctl(t, baseURL, "cedar", "schema")

	require.ErrorIs(t, err, client.ErrUnsupported)
	require.EqualError(t, err, "get cedar schema: requires Lakekeeper+; server is Lakekeeper 0.15.0")
}

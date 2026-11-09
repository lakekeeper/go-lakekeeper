package commands

import (
	"bytes"
	"testing"
)

// runLkctl runs the full lkctl command tree from argv against baseURL (an
// httptest server from testutil.NewServer) with a static token, and returns
// what the command wrote to stdout. Connection flags are passed explicitly so
// LAKEKEEPER_* environment variables can't redirect the test to a real server.
// It is the command-level seam: use it only for behaviour a façade test can't
// observe, such as which flags turn into which request.
func runLkctl(t *testing.T, baseURL string, args ...string) (string, error) {
	t.Helper()
	cmd := NewCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs(append([]string{
		"--base-url", baseURL,
		"--auth-mode", authModeToken,
		"--access-token", "test-token",
		"--bootstrap=false",
	}, args...))
	cmd.SetContext(t.Context())
	err := cmd.Execute()
	return out.String(), err
}

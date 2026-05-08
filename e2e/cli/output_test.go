//go:build e2e_cli

// Single owner of the `--output {json,text,wide}` matrix across every
// list/get command. Lifecycle tests intentionally don't repeat list-format
// assertions — content checks for create/rename/delete transitions belong
// there; "this command produces valid json" lives here.
//
// `server info` has dedicated tests in server_test.go (TestServerInfoJSON /
// TestServerInfoText) with stronger field-level assertions, so it is not
// repeated here.

package clie2e

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOutputFormatsAcrossCommands(t *testing.T) {
	requireBackend(t, BackendCompose)
	t.Parallel()

	// Default project ID — always present per Lakekeeper bootstrap.
	const defaultProject = "00000000-0000-0000-0000-000000000000"
	whID, _ := MustProvisionWarehouse(t)
	roleID := MustProvisionRole(t)
	userID := MustProvisionUser(t)

	cases := []struct {
		name    string
		args    []string
		formats []string // defaults to json+text
	}{
		{name: "project-list", args: []string{"project", "list"}},
		{name: "project-get", args: []string{"project", "get", defaultProject}},
		{name: "warehouse-list", args: []string{"warehouse", "list"}},
		{name: "warehouse-get", args: []string{"warehouse", "get", whID}},
		{name: "role-list", args: []string{"role", "list"}},
		{name: "role-get", args: []string{"role", "get", roleID}},
		{name: "user-list", args: []string{"user", "list"}, formats: []string{"json", "text", "wide"}},
		{name: "user-get", args: []string{"user", "get", userID}},
	}

	for _, tc := range cases {
		formats := tc.formats
		if formats == nil {
			formats = []string{"json", "text"}
		}
		for _, format := range formats {
			t.Run(tc.name+"-"+format, func(t *testing.T) {
				t.Parallel()
				args := append([]string{}, tc.args...)
				args = append(args, "--output", format)
				out := runOK(t, args...)
				if format == "json" {
					var v any
					if err := json.Unmarshal(out, &v); err != nil {
						t.Fatalf("invalid json output: %v\nraw: %s", err, out)
					}
					return
				}
				require.NotEmpty(t, strings.TrimSpace(string(out)),
					"%s output is empty", format)
			})
		}
	}
}

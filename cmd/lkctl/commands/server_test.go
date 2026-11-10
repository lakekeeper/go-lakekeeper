package commands

import (
	"bytes"
	"net/http"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	managementv1 "github.com/lakekeeper/go-lakekeeper/pkg/apis/management/v1"
	"github.com/lakekeeper/go-lakekeeper/pkg/client"
	"github.com/lakekeeper/go-lakekeeper/pkg/testutil"
)

// serverTestInfoBody is a GET /management/v1/info response with every
// required ServerInfo property.
const serverTestInfoBody = `{
  "authz-backend": "openfga",
  "aws-system-identities-enabled": false,
  "azure-system-identities-enabled": false,
  "bootstrapped": true,
  "default-project-id": "00000000-0000-0000-0000-000000000000",
  "gcp-system-identities-enabled": false,
  "lakekeeper-version": "0.15.0",
  "license-status": {"license-type": "none", "valid": true},
  "managed-role-providers": [],
  "queues": [],
  "server-id": "srv-test",
  "version": "0.15.0"
}`

func TestServerInfo_ServerErrorKeepsStatus(t *testing.T) {
	t.Parallel()
	mux, baseURL := testutil.NewServer(t)
	testutil.Respond(t, mux, "GET /management/v1/info", http.StatusForbidden,
		`{"error":{"code":403,"message":"Forbidden","type":"NotAuthorized","stack":[]}}`)

	_, err := runLkctl(t, baseURL, "server", "info")

	require.ErrorContains(t, err, "get server info: 403 Forbidden")
	var apiErr *client.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, http.StatusForbidden, apiErr.StatusCode)
}

func TestServerBootstrap_ServerErrorKeepsStatus(t *testing.T) {
	t.Parallel()
	mux, baseURL := testutil.NewServer(t)
	testutil.Respond(t, mux, "POST /management/v1/bootstrap", http.StatusBadRequest,
		`{"error":{"code":400,"message":"Catalog already bootstrapped","type":"CatalogAlreadyBootstrapped","stack":[]}}`)

	_, err := runLkctl(t, baseURL, "server", "bootstrap", "--accept-terms-of-use")

	require.ErrorContains(t, err, "bootstrap: 400 Bad Request")
	var apiErr *client.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, http.StatusBadRequest, apiErr.StatusCode)
}

func TestServerBootstrap_FlagsBecomeRequestBody(t *testing.T) {
	t.Parallel()
	mux, baseURL := testutil.NewServer(t)
	boot := testutil.Respond(t, mux, "POST /management/v1/bootstrap", http.StatusNoContent, nil)
	info := testutil.Respond(t, mux, "GET /management/v1/info", http.StatusOK, serverTestInfoBody)

	out, err := runLkctl(t, baseURL, "server", "bootstrap", "--accept-terms-of-use", "--as-operator")

	require.NoError(t, err)
	assert.JSONEq(t, `{"accept-terms-of-use":true,"is-operator":true}`, string(boot.Last(t).Body))
	assert.Equal(t, "Server bootstrapped successfully\n", out)
	assert.Equal(t, 0, info.Len(), "text output needs no server info")
}

func TestServerBootstrap_JSONPrintsServerInfo(t *testing.T) {
	t.Parallel()
	mux, baseURL := testutil.NewServer(t)
	boot := testutil.Respond(t, mux, "POST /management/v1/bootstrap", http.StatusNoContent, nil)
	testutil.Respond(t, mux, "GET /management/v1/info", http.StatusOK, serverTestInfoBody)

	out, err := runLkctl(t, baseURL, "server", "bootstrap", "-o", "json")

	require.NoError(t, err)
	assert.JSONEq(t, `{"accept-terms-of-use":false,"is-operator":false}`, string(boot.Last(t).Body))
	assert.Contains(t, out, `"server-id": "srv-test"`)
}

func TestNewServerCmdHasSubcommands(t *testing.T) {
	t.Parallel()

	cmd := newServerCmd(&clientOptions{})
	assert.Equal(t, "server", cmd.Use)

	got := []string{}
	for _, sub := range cmd.Commands() {
		got = append(got, sub.Name())
	}
	sort.Strings(got)
	assert.Equal(t, []string{"access", "assignments", "bootstrap", "grant", "info", "revoke"}, got)
}

// TestServerRevokeNoUsersOrRoles verifies the pre-network guard: `lkctl
// server revoke --assignments admin` (without any --users or --roles) fails
// before newClient is called.
func TestServerRevokeNoUsersOrRoles(t *testing.T) {
	t.Parallel()

	root := newServerCmd(&clientOptions{})
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"revoke", "--assignments", "admin"})

	err := root.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "at least one --users or --roles")
}

func TestPrintServerInfoText(t *testing.T) {
	t.Parallel()

	info := &managementv1.ServerInfo{
		ServerId:                     "srv-1",
		Version:                      "0.9.1",
		LakekeeperVersion:            "0.9.1",
		DefaultProjectId:             *managementv1.NewNullableString(managementv1.PtrString("default-proj")),
		Bootstrapped:                 true,
		AuthzBackend:                 "openfga",
		AwsSystemIdentitiesEnabled:   true,
		AzureSystemIdentitiesEnabled: false,
		GcpSystemIdentitiesEnabled:   true,
		Queues:                       []string{"q1", "q2"},
	}

	var buf bytes.Buffer
	require.NoError(t, printServerInfo(&buf, info))
	out := buf.String()

	for _, want := range []string{"srv-1", "0.9.1", "openfga", "default-proj", "q1", "q2"} {
		assert.Contains(t, out, want, "expected %q in %q", want, out)
	}
}

func TestPrintAllowedActionsEmpty(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	require.NoError(t, printAllowedActions[managementv1.ServerAction](&buf, nil))
	assert.Equal(t, "No access\n", buf.String())
}

func TestPrintAllowedActionsTable(t *testing.T) {
	t.Parallel()

	actions := []managementv1.ServerAction{
		managementv1.SERVERACTION_CREATE_PROJECT,
		managementv1.SERVERACTION_PROVISION_USERS,
	}

	var buf bytes.Buffer
	require.NoError(t, printAllowedActions(&buf, actions))
	out := buf.String()

	assert.Contains(t, out, "ALLOWED ACTIONS")
	assert.Contains(t, out, string(managementv1.SERVERACTION_CREATE_PROJECT))
	assert.Contains(t, out, string(managementv1.SERVERACTION_PROVISION_USERS))
}

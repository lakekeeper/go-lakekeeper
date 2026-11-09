package commands

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lakekeeper/go-lakekeeper/pkg/testutil"
)

// Command-level tests for the warehouse and role permission commands: they
// pin which flags turn into which request. Request shape per operation is
// covered by the permissions façade tests in pkg/client.

const (
	permWarehouseID = "9d6651da-5f19-11f1-ad69-1b7104fe3cf7"
	permRoleID      = "22222222-2222-2222-2222-222222222222"
	permUserID      = "oidc~alice"
)

func TestWarehouseAccess_UserFlagSelectsPrincipal(t *testing.T) {
	t.Parallel()
	mux, baseURL := testutil.NewServer(t)
	reqs := testutil.Respond(t, mux, "GET /management/v1/permissions/warehouse/{id}/access", http.StatusOK,
		`{"allowed-actions":["create_namespace"]}`)

	out, err := runLkctl(t, baseURL, "warehouse", "access", permWarehouseID, "--user", permUserID)

	require.NoError(t, err)
	assert.Contains(t, out, "create_namespace")
	got := reqs.Last(t)
	assert.Equal(t, "/management/v1/permissions/warehouse/"+permWarehouseID+"/access", got.Path)
	assert.Equal(t, []string{permUserID}, got.Query["principalUser"])
}

func TestWarehouseAssignments_RelationsFlagFilters(t *testing.T) {
	t.Parallel()
	mux, baseURL := testutil.NewServer(t)
	reqs := testutil.Respond(t, mux, "GET /management/v1/permissions/warehouse/{id}/assignments", http.StatusOK,
		`{"assignments":[{"type":"ownership","user":"oidc~alice"}]}`)

	out, err := runLkctl(t, baseURL, "warehouse", "assignments", permWarehouseID, "--relations", "ownership,describe")

	require.NoError(t, err)
	assert.Contains(t, out, "oidc~alice")
	assert.Equal(t, []string{"ownership", "describe"}, reqs.Last(t).Query["relations"])
}

func TestWarehouseGrantAndRevoke_FlagsBuildAssignments(t *testing.T) {
	t.Parallel()
	mux, baseURL := testutil.NewServer(t)
	reqs := testutil.Respond(t, mux, "POST /management/v1/permissions/warehouse/{id}/assignments", http.StatusNoContent, nil)

	out, err := runLkctl(t, baseURL, "warehouse", "grant", permWarehouseID,
		"--users", permUserID, "--roles", permRoleID, "--assignments", "describe")
	require.NoError(t, err)
	assert.Contains(t, out, "Warehouse permissions updated")
	assert.JSONEq(t, `{"writes":[{"type":"describe","user":"oidc~alice"},{"type":"describe","role":"`+permRoleID+`"}]}`,
		string(reqs.Last(t).Body))

	_, err = runLkctl(t, baseURL, "warehouse", "revoke", permWarehouseID, "--users", permUserID, "--assignments", "describe")
	require.NoError(t, err)
	assert.JSONEq(t, `{"deletes":[{"type":"describe","user":"oidc~alice"}]}`, string(reqs.Last(t).Body))
}

func TestRoleAccess_RoleFlagSelectsPrincipal(t *testing.T) {
	t.Parallel()
	mux, baseURL := testutil.NewServer(t)
	reqs := testutil.Respond(t, mux, "GET /management/v1/permissions/role/{id}/authorizer-actions", http.StatusOK,
		`{"allowed-actions":["assume"]}`)

	out, err := runLkctl(t, baseURL, "role", "access", permRoleID, "--role", permRoleID)

	require.NoError(t, err)
	assert.Contains(t, out, "assume")
	got := reqs.Last(t)
	assert.Equal(t, []string{permRoleID}, got.Query["principalRole"])
	assert.NotContains(t, got.Header, "X-Project-Id")
}

func TestRoleAssignments_RelationsFlagFilters(t *testing.T) {
	t.Parallel()
	mux, baseURL := testutil.NewServer(t)
	reqs := testutil.Respond(t, mux, "GET /management/v1/permissions/role/{id}/assignments", http.StatusOK,
		`{"assignments":[{"type":"assignee","user":"oidc~alice"}]}`)

	out, err := runLkctl(t, baseURL, "role", "assignments", permRoleID, "--relations", "assignee")

	require.NoError(t, err)
	assert.Contains(t, out, "oidc~alice")
	assert.Equal(t, []string{"assignee"}, reqs.Last(t).Query["relations"])
}

func TestRoleGrantAndRevoke_FlagsBuildAssignments(t *testing.T) {
	t.Parallel()
	mux, baseURL := testutil.NewServer(t)
	reqs := testutil.Respond(t, mux, "POST /management/v1/permissions/role/{id}/assignments", http.StatusNoContent, nil)

	out, err := runLkctl(t, baseURL, "role", "grant", permRoleID, "--users", permUserID, "--assignments", "assignee")
	require.NoError(t, err)
	assert.Contains(t, out, "Role permissions updated")
	assert.JSONEq(t, `{"writes":[{"type":"assignee","user":"oidc~alice"}]}`, string(reqs.Last(t).Body))

	_, err = runLkctl(t, baseURL, "role", "revoke", permRoleID, "--roles", permRoleID, "--assignments", "assignee")
	require.NoError(t, err)
	assert.JSONEq(t, `{"deletes":[{"type":"assignee","role":"`+permRoleID+`"}]}`, string(reqs.Last(t).Body))
}

func TestPermissionCommands_ServerErrorShowsStatusAndMessage(t *testing.T) {
	t.Parallel()
	mux, baseURL := testutil.NewServer(t)
	testutil.Respond(t, mux, "POST /management/v1/permissions/warehouse/{id}/assignments", http.StatusConflict,
		`{"error":{"code":409,"message":"Tuple already exists","type":"TupleAlreadyExistsError","stack":[]}}`)

	_, err := runLkctl(t, baseURL, "warehouse", "grant", permWarehouseID, "--users", permUserID, "--assignments", "describe")

	require.ErrorContains(t, err, "grant warehouse assignments: 409 Conflict")
	require.ErrorContains(t, err, "TupleAlreadyExistsError")
}

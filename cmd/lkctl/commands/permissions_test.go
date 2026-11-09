package commands

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lakekeeper/go-lakekeeper/pkg/testutil"
)

// Command-level tests for the warehouse, role, project, and server
// permission commands: they pin which flags and arguments turn into which
// request. Request shape per operation is
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

func TestProjectAccess_ProjectArgumentSelectsProject(t *testing.T) {
	t.Parallel()
	mux, baseURL := testutil.NewServer(t)
	reqs := testutil.Respond(t, mux, "GET /management/v1/permissions/project/access", http.StatusOK,
		`{"allowed-actions":["create_warehouse"]}`)

	out, err := runLkctl(t, baseURL, "project", "access", "--user", permUserID)
	require.NoError(t, err)
	assert.Contains(t, out, "create_warehouse")
	got := reqs.Last(t)
	assert.NotContains(t, got.Header, "X-Project-Id", "omitted [PROJECT-ID] must leave the project to the server")
	assert.Equal(t, []string{permUserID}, got.Query["principalUser"])

	_, err = runLkctl(t, baseURL, "project", "access", projectTestID)
	require.NoError(t, err)
	assert.Equal(t, projectTestID, reqs.Last(t).Header.Get("X-Project-Id"))
}

func TestProjectAssignments_ProjectArgumentSelectsProject(t *testing.T) {
	t.Parallel()
	mux, baseURL := testutil.NewServer(t)
	reqs := testutil.Respond(t, mux, "GET /management/v1/permissions/project/assignments", http.StatusOK,
		`{"assignments":[{"type":"project_admin","user":"oidc~alice"}],"project-id":"`+projectTestID+`"}`)

	out, err := runLkctl(t, baseURL, "project", "assignments", "--relations", "project_admin,select")
	require.NoError(t, err)
	assert.Contains(t, out, "oidc~alice")
	got := reqs.Last(t)
	assert.NotContains(t, got.Header, "X-Project-Id", "omitted [PROJECT-ID] must leave the project to the server")
	assert.Equal(t, []string{"project_admin", "select"}, got.Query["relations"])

	_, err = runLkctl(t, baseURL, "project", "assignments", projectTestID)
	require.NoError(t, err)
	assert.Equal(t, projectTestID, reqs.Last(t).Header.Get("X-Project-Id"))
}

func TestProjectGrantAndRevoke_ProjectArgumentSelectsProject(t *testing.T) {
	t.Parallel()
	mux, baseURL := testutil.NewServer(t)
	reqs := testutil.Respond(t, mux, "POST /management/v1/permissions/project/assignments", http.StatusNoContent, nil)

	out, err := runLkctl(t, baseURL, "project", "grant", "--users", permUserID, "--assignments", "select")
	require.NoError(t, err)
	assert.Contains(t, out, "Project permissions updated")
	got := reqs.Last(t)
	assert.NotContains(t, got.Header, "X-Project-Id", "omitted [PROJECT-ID] must leave the project to the server")
	assert.JSONEq(t, `{"writes":[{"type":"select","user":"oidc~alice"}]}`, string(got.Body))

	_, err = runLkctl(t, baseURL, "project", "revoke", projectTestID, "--roles", permRoleID, "--assignments", "select")
	require.NoError(t, err)
	got = reqs.Last(t)
	assert.Equal(t, projectTestID, got.Header.Get("X-Project-Id"))
	assert.JSONEq(t, `{"deletes":[{"type":"select","role":"`+permRoleID+`"}]}`, string(got.Body))

	_, err = runLkctl(t, baseURL, "project", "grant", projectTestID, "--users", permUserID, "--assignments", "select")
	require.NoError(t, err)
	assert.Equal(t, projectTestID, reqs.Last(t).Header.Get("X-Project-Id"))

	_, err = runLkctl(t, baseURL, "project", "revoke", "--users", permUserID, "--assignments", "select")
	require.NoError(t, err)
	assert.NotContains(t, reqs.Last(t).Header, "X-Project-Id")
}

func TestProjectPermissionCommands_HelpNamesDefaultProject(t *testing.T) {
	t.Parallel()
	for _, sub := range []string{"access", "assignments", "grant", "revoke"} {
		t.Run(sub, func(t *testing.T) {
			t.Parallel()
			_, baseURL := testutil.NewServer(t)

			out, err := runLkctl(t, baseURL, "project", sub, "--help")

			require.NoError(t, err)
			assert.Contains(t, out, "PROJECT-ID is optional; when omitted, the server's default project is used.")
		})
	}
}

func TestServerAccess_RoleFlagSelectsPrincipal(t *testing.T) {
	t.Parallel()
	mux, baseURL := testutil.NewServer(t)
	reqs := testutil.Respond(t, mux, "GET /management/v1/permissions/server/access", http.StatusOK,
		`{"allowed-actions":["create_project"]}`)

	out, err := runLkctl(t, baseURL, "server", "access", "--role", permRoleID)

	require.NoError(t, err)
	assert.Contains(t, out, "create_project")
	got := reqs.Last(t)
	assert.Equal(t, []string{permRoleID}, got.Query["principalRole"])
	assert.NotContains(t, got.Header, "X-Project-Id")
}

func TestServerAssignments_RelationsFlagFilters(t *testing.T) {
	t.Parallel()
	mux, baseURL := testutil.NewServer(t)
	reqs := testutil.Respond(t, mux, "GET /management/v1/permissions/server/assignments", http.StatusOK,
		`{"assignments":[{"type":"admin","user":"oidc~alice"}]}`)

	out, err := runLkctl(t, baseURL, "server", "assignments", "--relations", "admin,operator")

	require.NoError(t, err)
	assert.Contains(t, out, "oidc~alice")
	assert.Equal(t, []string{"admin", "operator"}, reqs.Last(t).Query["relations"])
}

func TestServerGrantAndRevoke_ServerErrorNamesOperation(t *testing.T) {
	t.Parallel()
	mux, baseURL := testutil.NewServer(t)
	reqs := testutil.Respond(t, mux, "POST /management/v1/permissions/server/assignments", http.StatusForbidden,
		`{"error":{"code":403,"message":"Forbidden action","type":"Unauthorized","stack":[]}}`)

	_, err := runLkctl(t, baseURL, "server", "grant", "--users", permUserID, "--assignments", "operator")
	require.ErrorContains(t, err, "grant server assignments: 403 Forbidden")
	assert.JSONEq(t, `{"writes":[{"type":"operator","user":"oidc~alice"}]}`, string(reqs.Last(t).Body))

	_, err = runLkctl(t, baseURL, "server", "revoke", "--roles", permRoleID, "--assignments", "admin")
	require.ErrorContains(t, err, "revoke server assignments: 403 Forbidden")
	assert.JSONEq(t, `{"deletes":[{"type":"admin","role":"`+permRoleID+`"}]}`, string(reqs.Last(t).Body))
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

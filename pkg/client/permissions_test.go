package client_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	managementv1 "github.com/lakekeeper/go-lakekeeper/pkg/apis/management/v1"
	"github.com/lakekeeper/go-lakekeeper/pkg/client"
	"github.com/lakekeeper/go-lakekeeper/pkg/permissions"
	"github.com/lakekeeper/go-lakekeeper/pkg/testutil"
)

const (
	permWarehouseID = "9d6651da-5f19-11f1-ad69-1b7104fe3cf7"
	permUserID      = "oidc~alice"
	permRoleID      = "22222222-2222-2222-2222-222222222222"
)

func TestWarehousePermissions_Access_SendsPrincipalFilter(t *testing.T) {
	t.Parallel()
	mux, c := testutil.ServerMux(t)
	reqs := testutil.Respond(t, mux, "GET /management/v1/permissions/warehouse/{id}/access", http.StatusOK,
		`{"allowed-actions":["create_namespace","delete"]}`)

	got, err := c.Permissions.Warehouses.Access(t.Context(), permWarehouseID,
		&client.GetWarehouseAccessByIDOptions{PrincipalUser: permUserID})

	require.NoError(t, err)
	assert.Equal(t, []managementv1.WarehouseAction{
		managementv1.WAREHOUSEACTION_CREATE_NAMESPACE, managementv1.WAREHOUSEACTION_DELETE,
	}, got.GetAllowedActions())
	r := reqs.Last(t)
	assert.Equal(t, "/management/v1/permissions/warehouse/"+permWarehouseID+"/access", r.Path)
	assert.Equal(t, []string{permUserID}, r.Query["principalUser"])
	assert.False(t, r.Query.Has("principalRole"))
}

const permWarehouseAssignmentsBody = `{"assignments":[{"type":"ownership","user":"oidc~alice"},{"type":"describe","role":"22222222-2222-2222-2222-222222222222"}]}`

// Lakekeeper v0.15.0 reads an array query parameter only as repeated keys
// (relations=a&relations=b) and silently ignores relations[0]=a.
func TestWarehousePermissions_Assignments_SendsRelationsAsRepeatedKeys(t *testing.T) {
	t.Parallel()
	mux, c := testutil.ServerMux(t)
	reqs := testutil.Respond(t, mux, "GET /management/v1/permissions/warehouse/{id}/assignments", http.StatusOK, permWarehouseAssignmentsBody)

	got, err := c.Permissions.Warehouses.Assignments(t.Context(), permWarehouseID,
		&client.GetWarehouseAssignmentsByIDOptions{Relations: []managementv1.WarehouseRelation{
			managementv1.WAREHOUSERELATION_OWNERSHIP, managementv1.WAREHOUSERELATION_DESCRIBE,
		}})

	require.NoError(t, err)
	require.Len(t, got.GetAssignments(), 2)
	r := reqs.Last(t)
	assert.Equal(t, "/management/v1/permissions/warehouse/"+permWarehouseID+"/assignments", r.Path)
	assert.Equal(t, []string{"ownership", "describe"}, r.Query["relations"])
	assert.False(t, r.Query.Has("relations[0]"))
}

func TestWarehousePermissions_Grant_SendsWritesOnly(t *testing.T) {
	t.Parallel()
	mux, c := testutil.ServerMux(t)
	reqs := testutil.Respond(t, mux, "POST /management/v1/permissions/warehouse/{id}/assignments", http.StatusNoContent, nil)
	set, err := permissions.BuildAssignmentSet[managementv1.WarehouseAssignment](
		[]string{"describe"}, permissions.PrincipalSet{Users: []string{permUserID}, Roles: []string{permRoleID}})
	require.NoError(t, err)

	err = c.Permissions.Warehouses.Grant(t.Context(), permWarehouseID, set)

	require.NoError(t, err)
	r := reqs.Last(t)
	assert.Equal(t, "/management/v1/permissions/warehouse/"+permWarehouseID+"/assignments", r.Path)
	assert.JSONEq(t, `{"writes":[{"type":"describe","user":"oidc~alice"},{"type":"describe","role":"22222222-2222-2222-2222-222222222222"}]}`, string(r.Body))
}

func TestWarehousePermissions_Revoke_SendsDeletesOnly(t *testing.T) {
	t.Parallel()
	mux, c := testutil.ServerMux(t)
	reqs := testutil.Respond(t, mux, "POST /management/v1/permissions/warehouse/{id}/assignments", http.StatusNoContent, nil)
	a, err := permissions.BuildAssignment[managementv1.WarehouseAssignment]("ownership", permissions.PrincipalUser, permUserID)
	require.NoError(t, err)

	err = c.Permissions.Warehouses.Revoke(t.Context(), permWarehouseID, []managementv1.WarehouseAssignment{a})

	require.NoError(t, err)
	assert.JSONEq(t, `{"deletes":[{"type":"ownership","user":"oidc~alice"}]}`, string(reqs.Last(t).Body))
}

func TestWarehousePermissions_EmptyAssignmentsRejected(t *testing.T) {
	t.Parallel()
	mux, c := testutil.ServerMux(t)
	reqs := testutil.Respond(t, mux, "POST /management/v1/permissions/warehouse/{id}/assignments", http.StatusNoContent, nil)

	require.ErrorContains(t, c.Permissions.Warehouses.Grant(t.Context(), permWarehouseID, nil), "grant warehouse assignments: no assignments")
	require.ErrorContains(t, c.Permissions.Warehouses.Revoke(t.Context(), permWarehouseID, nil), "revoke warehouse assignments: no assignments")
	assert.Zero(t, reqs.Len(), "no request must be sent")
}

func TestRolePermissions_Access_SendsPrincipalFilter(t *testing.T) {
	t.Parallel()
	mux, c := testutil.ServerMux(t)
	reqs := testutil.Respond(t, mux, "GET /management/v1/permissions/role/{id}/authorizer-actions", http.StatusOK,
		`{"allowed-actions":["assume","read_assignments"]}`)

	got, err := c.Permissions.Roles.Access(t.Context(), permRoleID,
		&client.GetAuthorizerRoleActionsOptions{PrincipalRole: permRoleID})

	require.NoError(t, err)
	assert.Equal(t, []managementv1.OpenFGARoleAction{
		managementv1.OPENFGAROLEACTION_ASSUME, managementv1.OPENFGAROLEACTION_READ_ASSIGNMENTS,
	}, got.GetAllowedActions())
	r := reqs.Last(t)
	assert.Equal(t, "/management/v1/permissions/role/"+permRoleID+"/authorizer-actions", r.Path)
	assert.Equal(t, []string{permRoleID}, r.Query["principalRole"])
	assert.False(t, r.Query.Has("principalUser"))
}

func TestRolePermissions_Assignments_SendsRelationsAsRepeatedKeys(t *testing.T) {
	t.Parallel()
	mux, c := testutil.ServerMux(t)
	reqs := testutil.Respond(t, mux, "GET /management/v1/permissions/role/{id}/assignments", http.StatusOK,
		`{"assignments":[{"type":"assignee","user":"oidc~alice"}]}`)

	got, err := c.Permissions.Roles.Assignments(t.Context(), permRoleID,
		&client.GetRoleAssignmentsByIDOptions{Relations: []managementv1.RoleRelation{
			managementv1.ROLERELATION_OWNERSHIP, managementv1.ROLERELATION_ASSIGNEE,
		}})

	require.NoError(t, err)
	require.Len(t, got.GetAssignments(), 1)
	r := reqs.Last(t)
	assert.Equal(t, "/management/v1/permissions/role/"+permRoleID+"/assignments", r.Path)
	assert.Equal(t, []string{"ownership", "assignee"}, r.Query["relations"])
	assert.False(t, r.Query.Has("relations[0]"))
}

func TestRolePermissions_GrantAndRevoke_SendWritesOrDeletes(t *testing.T) {
	t.Parallel()
	mux, c := testutil.ServerMux(t)
	reqs := testutil.Respond(t, mux, "POST /management/v1/permissions/role/{id}/assignments", http.StatusNoContent, nil)
	set, err := permissions.BuildAssignmentSet[managementv1.RoleAssignment](
		[]string{"assignee"}, permissions.PrincipalSet{Users: []string{permUserID}})
	require.NoError(t, err)

	require.NoError(t, c.Permissions.Roles.Grant(t.Context(), permRoleID, set))
	r := reqs.Last(t)
	assert.Equal(t, "/management/v1/permissions/role/"+permRoleID+"/assignments", r.Path)
	assert.JSONEq(t, `{"writes":[{"type":"assignee","user":"oidc~alice"}]}`, string(r.Body))

	require.NoError(t, c.Permissions.Roles.Revoke(t.Context(), permRoleID, set))
	assert.JSONEq(t, `{"deletes":[{"type":"assignee","user":"oidc~alice"}]}`, string(reqs.Last(t).Body))
}

// permissionCalls drives every warehouse and role permission operation with
// minimal valid input, for the tests that check what all of them share.
func permissionCalls(t *testing.T) []struct {
	name, pattern, op string
	call              func(c *client.Client) error
} {
	t.Helper()
	whSet, err := permissions.BuildAssignmentSet[managementv1.WarehouseAssignment](
		[]string{"describe"}, permissions.PrincipalSet{Users: []string{permUserID}})
	require.NoError(t, err)
	roleSet, err := permissions.BuildAssignmentSet[managementv1.RoleAssignment](
		[]string{"assignee"}, permissions.PrincipalSet{Users: []string{permUserID}})
	require.NoError(t, err)

	return []struct {
		name, pattern, op string
		call              func(c *client.Client) error
	}{
		{"warehouse access", "GET /management/v1/permissions/warehouse/{id}/access", "get warehouse access", func(c *client.Client) error {
			_, err := c.Permissions.Warehouses.Access(t.Context(), permWarehouseID, nil)
			return err
		}},
		{"warehouse assignments", "GET /management/v1/permissions/warehouse/{id}/assignments", "get warehouse assignments", func(c *client.Client) error {
			_, err := c.Permissions.Warehouses.Assignments(t.Context(), permWarehouseID, nil)
			return err
		}},
		{"warehouse grant", "POST /management/v1/permissions/warehouse/{id}/assignments", "grant warehouse assignments", func(c *client.Client) error {
			return c.Permissions.Warehouses.Grant(t.Context(), permWarehouseID, whSet)
		}},
		{"warehouse revoke", "POST /management/v1/permissions/warehouse/{id}/assignments", "revoke warehouse assignments", func(c *client.Client) error {
			return c.Permissions.Warehouses.Revoke(t.Context(), permWarehouseID, whSet)
		}},
		{"role access", "GET /management/v1/permissions/role/{id}/authorizer-actions", "get role access", func(c *client.Client) error {
			_, err := c.Permissions.Roles.Access(t.Context(), permRoleID, nil)
			return err
		}},
		{"role assignments", "GET /management/v1/permissions/role/{id}/assignments", "get role assignments", func(c *client.Client) error {
			_, err := c.Permissions.Roles.Assignments(t.Context(), permRoleID, nil)
			return err
		}},
		{"role grant", "POST /management/v1/permissions/role/{id}/assignments", "grant role assignments", func(c *client.Client) error {
			return c.Permissions.Roles.Grant(t.Context(), permRoleID, roleSet)
		}},
		{"role revoke", "POST /management/v1/permissions/role/{id}/assignments", "revoke role assignments", func(c *client.Client) error {
			return c.Permissions.Roles.Revoke(t.Context(), permRoleID, roleSet)
		}},
	}
}

func TestPermissions_ServerErrorIsAPIError(t *testing.T) {
	t.Parallel()
	for _, tc := range permissionCalls(t) {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			mux, c := testutil.ServerMux(t)
			testutil.Respond(t, mux, tc.pattern, http.StatusForbidden,
				`{"error":{"code":403,"message":"Forbidden action","type":"Unauthorized","stack":[]}}`)

			err := tc.call(c)

			require.ErrorContains(t, err, tc.op+": 403 Forbidden")
			var apiErr *client.APIError
			require.ErrorAs(t, err, &apiErr)
			assert.Equal(t, http.StatusForbidden, apiErr.StatusCode)
			require.NotNil(t, apiErr.Model)
			assert.Equal(t, "Unauthorized", apiErr.Model.GetType())
			var genErr *managementv1.GenericOpenAPIError
			require.ErrorAs(t, err, &genErr, "errors.As must reach the generated error")
		})
	}
}

// No permission endpoint of a warehouse or role is project-scoped: the
// resource id selects the project. Nothing may send one.
func TestPermissions_SendNoProjectAndNoQueryByDefault(t *testing.T) {
	t.Parallel()
	for _, tc := range permissionCalls(t) {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			mux, c := testutil.ServerMux(t)
			reqs := testutil.Respond(t, mux, tc.pattern, http.StatusOK, `{"allowed-actions":[],"assignments":[]}`)

			require.NoError(t, tc.call(c))

			got := reqs.Last(t)
			assert.NotContains(t, got.Header, "X-Project-Id")
			assert.Empty(t, got.Query, "nil options must send no query parameters")
		})
	}
}

func TestRolePermissions_EmptyAssignmentsRejected(t *testing.T) {
	t.Parallel()
	mux, c := testutil.ServerMux(t)
	reqs := testutil.Respond(t, mux, "POST /management/v1/permissions/role/{id}/assignments", http.StatusNoContent, nil)

	require.ErrorContains(t, c.Permissions.Roles.Grant(t.Context(), permRoleID, nil), "grant role assignments: no assignments")
	require.ErrorContains(t, c.Permissions.Roles.Revoke(t.Context(), permRoleID, nil), "revoke role assignments: no assignments")
	assert.Zero(t, reqs.Len(), "no request must be sent")
}

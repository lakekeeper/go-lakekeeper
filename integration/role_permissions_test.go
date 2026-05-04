//go:build integration

// Most tests here scope their writes to a freshly-created role, but
// TestPermissions_Role_GetAccess and TestPermissions_Role_GetAssignments
// assert exact assignment sets on roles created under the default project.
// Keep this file serial unless that changes.
package integration

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	managementv1 "github.com/lakekeeper/go-lakekeeper/pkg/apis/management/v1"
	"github.com/lakekeeper/go-lakekeeper/pkg/permissions"
)

func TestPermissions_Role_GetAccess(t *testing.T) {
	c := sharedClient

	role := MustCreateRole(t, c, defaultProjectID)

	// The plan favors GetAuthorizerRoleActions over the deprecated
	// GetRoleAccessById because the access surface is the only one the new
	// endpoint covers cleanly.
	resp, r, err := c.PermissionsOpenfgaAPI.GetAuthorizerRoleActions(t.Context(), role.Id).Execute()
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, r.StatusCode)

	// GetAuthorizerRoleActions reports the OpenFGA-level action set, which is
	// narrower than the deprecated GetRoleAccessById's RoleAction set —
	// no delete/update/read since those are entity-level CRUD that fan out
	// to other action types in the new model. (For the assert.Subset
	// convention shared by all *_GetAccess tests, see integration_test.go.)
	want := []managementv1.OpenFGARoleAction{
		managementv1.OPENFGAROLEACTION_ASSUME,
		managementv1.OPENFGAROLEACTION_CAN_GRANT_ASSIGNEE,
		managementv1.OPENFGAROLEACTION_CAN_CHANGE_OWNERSHIP,
		managementv1.OPENFGAROLEACTION_READ_ASSIGNMENTS,
	}
	assert.Subset(t, want, resp.AllowedActions)
}

func TestPermissions_Role_GetAssignments(t *testing.T) {
	c := sharedClient

	role := MustCreateRole(t, c, defaultProjectID)

	resp, r, err := c.PermissionsOpenfgaAPI.GetRoleAssignmentsById(t.Context(), role.Id).Execute()
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, r.StatusCode)

	assert.ElementsMatch(t,
		[]permissions.AssignmentRow{{PrincipalType: "user", PrincipalID: adminID, Relation: "ownership"}},
		describeAssignments(t, resp.Assignments),
	)
}

func TestPermissions_Role_Update(t *testing.T) {
	c := sharedClient

	projectID := MustCreateProject(t, c)
	role := MustCreateRole(t, c, projectID)
	user := MustProvisionUser(t, c)

	resp, _, err := c.PermissionsOpenfgaAPI.GetRoleAssignmentsById(t.Context(), role.Id).Execute()
	require.NoError(t, err)
	assert.ElementsMatch(t,
		[]permissions.AssignmentRow{{PrincipalType: "user", PrincipalID: adminID, Relation: "ownership"}},
		describeAssignments(t, resp.Assignments),
	)

	addReq := managementv1.NewUpdateRoleAssignmentsRequest()
	addReq.Writes = append(addReq.Writes, userAssignment[managementv1.RoleAssignment](t, "assignee", user.Id))

	r, err := c.PermissionsOpenfgaAPI.UpdateRoleAssignmentsById(t.Context(), role.Id).UpdateRoleAssignmentsRequest(*addReq).Execute()
	require.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, r.StatusCode)

	resp, _, err = c.PermissionsOpenfgaAPI.GetRoleAssignmentsById(t.Context(), role.Id).Execute()
	require.NoError(t, err)
	assert.ElementsMatch(t,
		[]permissions.AssignmentRow{
			{PrincipalType: "user", PrincipalID: adminID, Relation: "ownership"},
			{PrincipalType: "user", PrincipalID: user.Id, Relation: "assignee"},
		},
		describeAssignments(t, resp.Assignments),
	)

	delReq := managementv1.NewUpdateRoleAssignmentsRequest()
	delReq.Deletes = append(delReq.Deletes, userAssignment[managementv1.RoleAssignment](t, "assignee", user.Id))

	r, err = c.PermissionsOpenfgaAPI.UpdateRoleAssignmentsById(t.Context(), role.Id).UpdateRoleAssignmentsRequest(*delReq).Execute()
	require.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, r.StatusCode)

	resp, _, err = c.PermissionsOpenfgaAPI.GetRoleAssignmentsById(t.Context(), role.Id).Execute()
	require.NoError(t, err)
	assert.ElementsMatch(t,
		[]permissions.AssignmentRow{{PrincipalType: "user", PrincipalID: adminID, Relation: "ownership"}},
		describeAssignments(t, resp.Assignments),
	)
}

func TestPermissions_Role_SameAdd(t *testing.T) {
	c := sharedClient

	user := MustProvisionUser(t, c)
	role := MustCreateRole(t, c, defaultProjectID)

	req := managementv1.NewUpdateRoleAssignmentsRequest()
	req.Writes = append(req.Writes, userAssignment[managementv1.RoleAssignment](t, "assignee", user.Id))

	r, err := c.PermissionsOpenfgaAPI.UpdateRoleAssignmentsById(t.Context(), role.Id).UpdateRoleAssignmentsRequest(*req).Execute()
	require.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, r.StatusCode)

	r, err = c.PermissionsOpenfgaAPI.UpdateRoleAssignmentsById(t.Context(), role.Id).UpdateRoleAssignmentsRequest(*req).Execute()
	require.Error(t, err)
	require.NotNil(t, r)
	assert.Equal(t, http.StatusConflict, r.StatusCode)
	assert.Contains(t, errorBody(err), "TupleAlreadyExistsError")
}

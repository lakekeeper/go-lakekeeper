//go:build integration

// Tests in this file write to assignment graphs on the *default project*
// (a shared process-wide resource) and assert exact assignment sets. They
// are deliberately serial — adding t.Parallel() would race the ElementsMatch
// checks. New tests that target a freshly-created project are safe to
// parallelize; ones that touch defaultProjectID are not.
package integration

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	managementv1 "github.com/lakekeeper/go-lakekeeper/pkg/apis/management/v1"
	"github.com/lakekeeper/go-lakekeeper/pkg/permissions"
)

// projectAdminRow is the AssignmentRow shape for the auto-granted admin
// assignment that bootstrap puts on the default project.
func projectAdminRow() permissions.AssignmentRow {
	return permissions.AssignmentRow{PrincipalType: "user", PrincipalID: adminID, Relation: "project_admin"}
}

func TestPermissions_Project_GetAccess(t *testing.T) {
	c := sharedClient

	resp, r, err := c.PermissionsOpenfgaAPI.GetProjectAccessById(t.Context(), defaultProjectID).Execute()
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, r.StatusCode)

	want := []managementv1.ProjectAction{
		managementv1.ProjectActionCreateWarehouse,
		managementv1.ProjectActionDelete,
		managementv1.ProjectActionRename,
		managementv1.ProjectActionListWarehouses,
		managementv1.ProjectActionCreateRole,
		managementv1.ProjectActionListRoles,
		managementv1.ProjectActionSearchRoles,
		managementv1.ProjectActionReadAssignments,
		managementv1.ProjectActionGrantRoleCreator,
		managementv1.ProjectActionGrantCreate,
		managementv1.ProjectActionGrantDescribe,
		managementv1.ProjectActionGrantModify,
		managementv1.ProjectActionGrantSelect,
		managementv1.ProjectActionGrantProjectAdmin,
		managementv1.ProjectActionGrantSecurityAdmin,
		managementv1.ProjectActionGrantDataAdmin,
		managementv1.ProjectActionGetEndpointStatistics,
	}
	assert.Subset(t, want, resp.AllowedActions)
}

func TestPermissions_Project_GetAssignments(t *testing.T) {
	c := sharedClient

	resp, r, err := c.PermissionsOpenfgaAPI.GetProjectAssignmentsById(t.Context(), defaultProjectID).Execute()
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, r.StatusCode)

	assert.Equal(t, defaultProjectID, resp.ProjectId)
	assert.ElementsMatch(t,
		[]permissions.AssignmentRow{projectAdminRow()},
		describeAssignments(t, resp.Assignments),
	)
}

func TestPermissions_Project_Update(t *testing.T) {
	c := sharedClient

	user := MustProvisionUser(t, c)

	resp, _, err := c.PermissionsOpenfgaAPI.GetProjectAssignmentsById(t.Context(), defaultProjectID).Execute()
	require.NoError(t, err)
	assert.ElementsMatch(t,
		[]permissions.AssignmentRow{projectAdminRow()},
		describeAssignments(t, resp.Assignments),
	)

	addReq := managementv1.NewUpdateProjectAssignmentsRequest()
	addReq.Writes = append(addReq.Writes, userAssignment[managementv1.ProjectAssignment](t, "select", user.Id))

	r, err := c.PermissionsOpenfgaAPI.UpdateProjectAssignmentsById(t.Context(), defaultProjectID).UpdateProjectAssignmentsRequest(*addReq).Execute()
	require.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, r.StatusCode)

	resp, _, err = c.PermissionsOpenfgaAPI.GetProjectAssignmentsById(t.Context(), defaultProjectID).Execute()
	require.NoError(t, err)
	assert.ElementsMatch(t,
		[]permissions.AssignmentRow{
			projectAdminRow(),
			{PrincipalType: "user", PrincipalID: user.Id, Relation: "select"},
		},
		describeAssignments(t, resp.Assignments),
	)

	delReq := managementv1.NewUpdateProjectAssignmentsRequest()
	delReq.Deletes = append(delReq.Deletes, userAssignment[managementv1.ProjectAssignment](t, "select", user.Id))

	r, err = c.PermissionsOpenfgaAPI.UpdateProjectAssignmentsById(t.Context(), defaultProjectID).UpdateProjectAssignmentsRequest(*delReq).Execute()
	require.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, r.StatusCode)

	resp, _, err = c.PermissionsOpenfgaAPI.GetProjectAssignmentsById(t.Context(), defaultProjectID).Execute()
	require.NoError(t, err)
	assert.ElementsMatch(t,
		[]permissions.AssignmentRow{projectAdminRow()},
		describeAssignments(t, resp.Assignments),
	)
}

func TestPermissions_Project_SameAdd(t *testing.T) {
	c := sharedClient

	user := MustProvisionUser(t, c)

	req := managementv1.NewUpdateProjectAssignmentsRequest()
	req.Writes = append(req.Writes, userAssignment[managementv1.ProjectAssignment](t, "modify", user.Id))

	r, err := c.PermissionsOpenfgaAPI.UpdateProjectAssignmentsById(t.Context(), defaultProjectID).UpdateProjectAssignmentsRequest(*req).Execute()
	require.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, r.StatusCode)
	// defaultProjectID is a shared resource; undo the write so other tests
	// asserting exact assignment sets don't see a leftover tuple. Cleanups
	// run LIFO, so this fires before MustProvisionUser's user-delete cleanup.
	t.Cleanup(func() {
		delReq := managementv1.NewUpdateProjectAssignmentsRequest()
		delReq.Deletes = append(delReq.Deletes, userAssignment[managementv1.ProjectAssignment](t, "modify", user.Id))
		if _, err := c.PermissionsOpenfgaAPI.UpdateProjectAssignmentsById(context.Background(), defaultProjectID).UpdateProjectAssignmentsRequest(*delReq).Execute(); err != nil {
			t.Errorf("undo project assignment: %v", err)
		}
	})

	r, err = c.PermissionsOpenfgaAPI.UpdateProjectAssignmentsById(t.Context(), defaultProjectID).UpdateProjectAssignmentsRequest(*req).Execute()
	require.Error(t, err)
	require.NotNil(t, r)
	assert.Equal(t, http.StatusConflict, r.StatusCode)
	assert.Contains(t, errorBody(err), "TupleAlreadyExistsError")
}

func TestPermissions_Project_Add_NewProject(t *testing.T) {
	c := sharedClient

	user := MustProvisionUser(t, c)
	projectID := MustCreateProject(t, c)

	resp, r, err := c.PermissionsOpenfgaAPI.GetProjectAssignmentsById(t.Context(), projectID).Execute()
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, r.StatusCode)
	assert.Equal(t, projectID, resp.ProjectId)
	assert.ElementsMatch(t,
		[]permissions.AssignmentRow{{PrincipalType: "user", PrincipalID: adminID, Relation: "project_admin"}},
		describeAssignments(t, resp.Assignments),
	)

	addReq := managementv1.NewUpdateProjectAssignmentsRequest()
	addReq.Writes = append(addReq.Writes, userAssignment[managementv1.ProjectAssignment](t, "modify", user.Id))

	r, err = c.PermissionsOpenfgaAPI.UpdateProjectAssignmentsById(t.Context(), projectID).UpdateProjectAssignmentsRequest(*addReq).Execute()
	require.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, r.StatusCode)

	resp, _, err = c.PermissionsOpenfgaAPI.GetProjectAssignmentsById(t.Context(), projectID).Execute()
	require.NoError(t, err)
	assert.ElementsMatch(t,
		[]permissions.AssignmentRow{
			{PrincipalType: "user", PrincipalID: adminID, Relation: "project_admin"},
			{PrincipalType: "user", PrincipalID: user.Id, Relation: "modify"},
		},
		describeAssignments(t, resp.Assignments),
	)
}

func TestPermissions_Project_Add_Role(t *testing.T) {
	c := sharedClient

	projectID := MustCreateProject(t, c)
	role := MustCreateRole(t, c, projectID)

	addReq := managementv1.NewUpdateProjectAssignmentsRequest()
	addReq.Writes = append(addReq.Writes, roleAssignment[managementv1.ProjectAssignment](t, "describe", role.Id))

	r, err := c.PermissionsOpenfgaAPI.UpdateProjectAssignmentsById(t.Context(), projectID).UpdateProjectAssignmentsRequest(*addReq).Execute()
	require.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, r.StatusCode)

	resp, _, err := c.PermissionsOpenfgaAPI.GetProjectAssignmentsById(t.Context(), projectID).Execute()
	require.NoError(t, err)
	assert.ElementsMatch(t,
		[]permissions.AssignmentRow{
			{PrincipalType: "user", PrincipalID: adminID, Relation: "project_admin"},
			{PrincipalType: "role", PrincipalID: role.Id, Relation: "describe"},
		},
		describeAssignments(t, resp.Assignments),
	)
}

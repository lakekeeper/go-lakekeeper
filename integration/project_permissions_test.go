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
	"github.com/lakekeeper/go-lakekeeper/pkg/client"
	"github.com/lakekeeper/go-lakekeeper/pkg/permissions"
)

// projectAdminRow is the AssignmentRow shape for the auto-granted admin
// assignment that bootstrap puts on the default project.
func projectAdminRow() permissions.AssignmentRow {
	return permissions.AssignmentRow{PrincipalType: "user", PrincipalID: adminID, Relation: "project_admin"}
}

func TestPermissions_Project_GetAccess(t *testing.T) {
	c := sharedClient

	resp, err := c.Permissions.Projects.Access(t.Context(), nil, client.WithProject(defaultProjectID))
	require.NoError(t, err)

	want := []managementv1.ProjectAction{
		managementv1.PROJECTACTION_CREATE_WAREHOUSE,
		managementv1.PROJECTACTION_DELETE,
		managementv1.PROJECTACTION_RENAME,
		managementv1.PROJECTACTION_LIST_WAREHOUSES,
		managementv1.PROJECTACTION_CREATE_ROLE,
		managementv1.PROJECTACTION_LIST_ROLES,
		managementv1.PROJECTACTION_SEARCH_ROLES,
		managementv1.PROJECTACTION_READ_ASSIGNMENTS,
		managementv1.PROJECTACTION_GRANT_ROLE_CREATOR,
		managementv1.PROJECTACTION_GRANT_CREATE,
		managementv1.PROJECTACTION_GRANT_DESCRIBE,
		managementv1.PROJECTACTION_GRANT_MODIFY,
		managementv1.PROJECTACTION_GRANT_SELECT,
		managementv1.PROJECTACTION_GRANT_PROJECT_ADMIN,
		managementv1.PROJECTACTION_GRANT_SECURITY_ADMIN,
		managementv1.PROJECTACTION_GRANT_DATA_ADMIN,
		managementv1.PROJECTACTION_GET_ENDPOINT_STATISTICS,
		managementv1.PROJECTACTION_GRANT_TAG_CREATOR,
		managementv1.PROJECTACTION_GRANT_WRITE,
	}
	assertAllowedActionsSubset(t, want, resp.GetAllowedActions(), managementv1.AllowedProjectActionEnumValues)
}

func TestPermissions_Project_GetAssignments(t *testing.T) {
	c := sharedClient

	resp, err := c.Permissions.Projects.Assignments(t.Context(), nil, client.WithProject(defaultProjectID))
	require.NoError(t, err)

	assert.Equal(t, defaultProjectID, resp.GetProjectId())
	assert.ElementsMatch(t,
		[]permissions.AssignmentRow{projectAdminRow()},
		describeAssignments(t, resp.GetAssignments()),
	)
}

// Without WithProject the façade sends no project; the compose stack keeps
// Lakekeeper's default configuration, so the server resolves the nil-UUID
// default project.
func TestPermissions_Project_GetAssignments_NoProjectUsesServerDefault(t *testing.T) {
	c := sharedClient

	resp, err := c.Permissions.Projects.Assignments(t.Context(), nil)
	require.NoError(t, err)

	assert.Equal(t, defaultProjectID, resp.GetProjectId())
}

func TestPermissions_Project_Update(t *testing.T) {
	c := sharedClient

	user := MustProvisionUser(t, c)

	resp, err := c.Permissions.Projects.Assignments(t.Context(), nil, client.WithProject(defaultProjectID))
	require.NoError(t, err)
	assert.ElementsMatch(t,
		[]permissions.AssignmentRow{projectAdminRow()},
		describeAssignments(t, resp.GetAssignments()),
	)

	sel := []managementv1.ProjectAssignment{userAssignment[managementv1.ProjectAssignment](t, "select", user.GetId())}
	require.NoError(t, c.Permissions.Projects.Grant(t.Context(), sel, client.WithProject(defaultProjectID)))

	resp, err = c.Permissions.Projects.Assignments(t.Context(), nil, client.WithProject(defaultProjectID))
	require.NoError(t, err)
	assert.ElementsMatch(t,
		[]permissions.AssignmentRow{
			projectAdminRow(),
			{PrincipalType: "user", PrincipalID: user.GetId(), Relation: "select"},
		},
		describeAssignments(t, resp.GetAssignments()),
	)

	require.NoError(t, c.Permissions.Projects.Revoke(t.Context(), sel, client.WithProject(defaultProjectID)))

	resp, err = c.Permissions.Projects.Assignments(t.Context(), nil, client.WithProject(defaultProjectID))
	require.NoError(t, err)
	assert.ElementsMatch(t,
		[]permissions.AssignmentRow{projectAdminRow()},
		describeAssignments(t, resp.GetAssignments()),
	)
}

func TestPermissions_Project_SameAdd(t *testing.T) {
	c := sharedClient

	user := MustProvisionUser(t, c)

	modify := []managementv1.ProjectAssignment{userAssignment[managementv1.ProjectAssignment](t, "modify", user.GetId())}
	require.NoError(t, c.Permissions.Projects.Grant(t.Context(), modify, client.WithProject(defaultProjectID)))
	// defaultProjectID is a shared resource; undo the write so other tests
	// asserting exact assignment sets don't see a leftover tuple. Cleanups
	// run LIFO, so this fires before MustProvisionUser's user-delete cleanup.
	t.Cleanup(func() {
		if err := c.Permissions.Projects.Revoke(context.Background(), modify, client.WithProject(defaultProjectID)); err != nil {
			t.Errorf("undo project assignment: %v", err)
		}
	})

	err := c.Permissions.Projects.Grant(t.Context(), modify, client.WithProject(defaultProjectID))
	var apiErr *client.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, http.StatusConflict, apiErr.StatusCode)
	assert.Contains(t, errorBody(err), "TupleAlreadyExistsError")
}

func TestPermissions_Project_Add_NewProject(t *testing.T) {
	c := sharedClient

	user := MustProvisionUser(t, c)
	projectID := MustCreateProject(t, c)

	resp, err := c.Permissions.Projects.Assignments(t.Context(), nil, client.WithProject(projectID))
	require.NoError(t, err)
	assert.Equal(t, projectID, resp.GetProjectId())
	assert.ElementsMatch(t,
		[]permissions.AssignmentRow{{PrincipalType: "user", PrincipalID: adminID, Relation: "project_admin"}},
		describeAssignments(t, resp.GetAssignments()),
	)

	modify := []managementv1.ProjectAssignment{userAssignment[managementv1.ProjectAssignment](t, "modify", user.GetId())}
	require.NoError(t, c.Permissions.Projects.Grant(t.Context(), modify, client.WithProject(projectID)))

	resp, err = c.Permissions.Projects.Assignments(t.Context(), nil, client.WithProject(projectID))
	require.NoError(t, err)
	assert.ElementsMatch(t,
		[]permissions.AssignmentRow{
			{PrincipalType: "user", PrincipalID: adminID, Relation: "project_admin"},
			{PrincipalType: "user", PrincipalID: user.GetId(), Relation: "modify"},
		},
		describeAssignments(t, resp.GetAssignments()),
	)
}

func TestPermissions_Project_Add_Role(t *testing.T) {
	c := sharedClient

	projectID := MustCreateProject(t, c)
	role := MustCreateRole(t, c, projectID)

	describe := []managementv1.ProjectAssignment{roleAssignment[managementv1.ProjectAssignment](t, "describe", role.GetId())}
	require.NoError(t, c.Permissions.Projects.Grant(t.Context(), describe, client.WithProject(projectID)))

	resp, err := c.Permissions.Projects.Assignments(t.Context(), nil, client.WithProject(projectID))
	require.NoError(t, err)
	assert.ElementsMatch(t,
		[]permissions.AssignmentRow{
			{PrincipalType: "user", PrincipalID: adminID, Relation: "project_admin"},
			{PrincipalType: "role", PrincipalID: role.GetId(), Relation: "describe"},
		},
		describeAssignments(t, resp.GetAssignments()),
	)
}

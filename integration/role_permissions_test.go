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
	"github.com/lakekeeper/go-lakekeeper/pkg/client"
	"github.com/lakekeeper/go-lakekeeper/pkg/permissions"
)

func TestPermissions_Role_GetAccess(t *testing.T) {
	c := sharedClient

	role := MustCreateRole(t, c, defaultProjectID)

	// Permissions.Roles.Access reads GetAuthorizerRoleActions, not the
	// deprecated GetRoleAccessById, because the access surface is the only
	// one the new endpoint covers cleanly.
	resp, err := c.Permissions.Roles.Access(t.Context(), role.Id, nil)
	require.NoError(t, err)

	// GetAuthorizerRoleActions reports the OpenFGA-level action set, which is
	// narrower than the deprecated GetRoleAccessById's RoleAction set —
	// no delete/update/read since those are entity-level CRUD that fan out
	// to other action types in the new model. (For the assertAllowedActionsSubset
	// convention shared by all *_GetAccess tests, see integration_test.go.)
	want := []managementv1.OpenFGARoleAction{
		managementv1.OPENFGAROLEACTION_ASSUME,
		managementv1.OPENFGAROLEACTION_CAN_GRANT_ASSIGNEE,
		managementv1.OPENFGAROLEACTION_CAN_CHANGE_OWNERSHIP,
		managementv1.OPENFGAROLEACTION_READ_ASSIGNMENTS,
	}
	assertAllowedActionsSubset(t, want, resp.AllowedActions, managementv1.AllowedOpenFGARoleActionEnumValues)
}

func TestPermissions_Role_GetAssignments(t *testing.T) {
	c := sharedClient

	role := MustCreateRole(t, c, defaultProjectID)

	resp, err := c.Permissions.Roles.Assignments(t.Context(), role.Id, nil)
	require.NoError(t, err)

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

	resp, err := c.Permissions.Roles.Assignments(t.Context(), role.Id, nil)
	require.NoError(t, err)
	assert.ElementsMatch(t,
		[]permissions.AssignmentRow{{PrincipalType: "user", PrincipalID: adminID, Relation: "ownership"}},
		describeAssignments(t, resp.Assignments),
	)

	assignee := []managementv1.RoleAssignment{userAssignment[managementv1.RoleAssignment](t, "assignee", user.Id)}
	require.NoError(t, c.Permissions.Roles.Grant(t.Context(), role.Id, assignee))

	resp, err = c.Permissions.Roles.Assignments(t.Context(), role.Id, nil)
	require.NoError(t, err)
	assert.ElementsMatch(t,
		[]permissions.AssignmentRow{
			{PrincipalType: "user", PrincipalID: adminID, Relation: "ownership"},
			{PrincipalType: "user", PrincipalID: user.Id, Relation: "assignee"},
		},
		describeAssignments(t, resp.Assignments),
	)

	require.NoError(t, c.Permissions.Roles.Revoke(t.Context(), role.Id, assignee))

	resp, err = c.Permissions.Roles.Assignments(t.Context(), role.Id, nil)
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

	assignee := []managementv1.RoleAssignment{userAssignment[managementv1.RoleAssignment](t, "assignee", user.Id)}
	require.NoError(t, c.Permissions.Roles.Grant(t.Context(), role.Id, assignee))

	err := c.Permissions.Roles.Grant(t.Context(), role.Id, assignee)
	var apiErr *client.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, http.StatusConflict, apiErr.StatusCode)
	assert.Contains(t, errorBody(err), "TupleAlreadyExistsError")
}

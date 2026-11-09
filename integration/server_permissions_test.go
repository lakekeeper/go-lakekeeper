//go:build integration

// Tests in this file mutate the *server-level* assignment graph, which is a
// process-wide shared resource. They are deliberately serial: they assert
// exact assignment sets, so two parallel writers would race and flake the
// ElementsMatch checks. Do not add t.Parallel() here without first
// rewriting the assertions to be relative-set-aware.
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

// serverAdminActions is the server action set an admin assignment grants.
var serverAdminActions = []managementv1.ServerAction{
	managementv1.SERVERACTION_CREATE_PROJECT,
	managementv1.SERVERACTION_UPDATE_USERS,
	managementv1.SERVERACTION_DELETE_USERS,
	managementv1.SERVERACTION_LIST_USERS,
	managementv1.SERVERACTION_GRANT_ADMIN,
	managementv1.SERVERACTION_PROVISION_USERS,
	managementv1.SERVERACTION_READ_ASSIGNMENTS,
}

// revokeOnCleanup removes server assignments when the test ends. The server
// is shared, so a leftover tuple would break later tests that assert exact
// assignment sets. Cleanups run LIFO, so this fires before an earlier
// MustProvisionUser's user-delete cleanup.
func revokeOnCleanup(t *testing.T, c *client.Client, assignments []managementv1.ServerAssignment) {
	t.Helper()
	t.Cleanup(func() {
		if err := c.Permissions.Server.Revoke(context.Background(), assignments); err != nil {
			t.Errorf("undo server assignment: %v", err)
		}
	})
}

func TestPermissions_Server_GetAccess(t *testing.T) {
	c := sharedClient

	resp, err := c.Permissions.Server.Access(t.Context(), nil)
	require.NoError(t, err)

	assertAllowedActionsSubset(t, serverAdminActions, resp.GetAllowedActions(), managementv1.AllowedServerActionEnumValues)
}

func TestPermissions_Server_GetAssignments(t *testing.T) {
	c := sharedClient

	resp, err := c.Permissions.Server.Assignments(t.Context(), nil)
	require.NoError(t, err)

	want := []permissions.AssignmentRow{{
		PrincipalType: "user",
		PrincipalID:   adminID,
		Relation:      "operator",
	}}
	assert.ElementsMatch(t, want, describeAssignments(t, resp.GetAssignments()))
}

func TestPermissions_Server_GetAssignments_RelationsFilter(t *testing.T) {
	c := sharedClient

	// The only server assignment is the admin's `operator`, so filtering on
	// `admin` must return nothing. A server that ignores the parameter's
	// encoding returns the unfiltered set instead of an error, which only a
	// negative case like this one catches.
	resp, err := c.Permissions.Server.Assignments(t.Context(), &client.GetServerAssignmentsOptions{
		Relations: []managementv1.ServerRelation{managementv1.SERVERRELATION_ADMIN},
	})
	require.NoError(t, err)
	assert.Empty(t, describeAssignments(t, resp.GetAssignments()))

	resp, err = c.Permissions.Server.Assignments(t.Context(), &client.GetServerAssignmentsOptions{
		Relations: []managementv1.ServerRelation{managementv1.SERVERRELATION_ADMIN, managementv1.SERVERRELATION_OPERATOR},
	})
	require.NoError(t, err)
	assert.ElementsMatch(t,
		[]permissions.AssignmentRow{{PrincipalType: "user", PrincipalID: adminID, Relation: "operator"}},
		describeAssignments(t, resp.GetAssignments()),
	)
}

func TestPermissions_Server_Update(t *testing.T) {
	c := sharedClient

	user := MustProvisionUser(t, c)

	resp, err := c.Permissions.Server.Assignments(t.Context(), nil)
	require.NoError(t, err)
	assert.ElementsMatch(t,
		[]permissions.AssignmentRow{{PrincipalType: "user", PrincipalID: adminID, Relation: "operator"}},
		describeAssignments(t, resp.GetAssignments()),
	)

	// add admin assignment for the new user
	admin := []managementv1.ServerAssignment{userAssignment[managementv1.ServerAssignment](t, "admin", user.GetId())}
	require.NoError(t, c.Permissions.Server.Grant(t.Context(), admin))
	// If the test stops before its own revoke below, remove the tuple anyway.
	removed := false
	t.Cleanup(func() {
		if removed {
			return
		}
		if err := c.Permissions.Server.Revoke(context.Background(), admin); err != nil {
			t.Errorf("undo server admin assignment: %v", err)
		}
	})

	resp, err = c.Permissions.Server.Assignments(t.Context(), nil)
	require.NoError(t, err)
	assert.ElementsMatch(t,
		[]permissions.AssignmentRow{
			{PrincipalType: "user", PrincipalID: adminID, Relation: "operator"},
			{PrincipalType: "user", PrincipalID: user.GetId(), Relation: "admin"},
		},
		describeAssignments(t, resp.GetAssignments()),
	)

	// remove the admin assignment
	require.NoError(t, c.Permissions.Server.Revoke(t.Context(), admin))
	removed = true

	resp, err = c.Permissions.Server.Assignments(t.Context(), nil)
	require.NoError(t, err)
	assert.ElementsMatch(t,
		[]permissions.AssignmentRow{{PrincipalType: "user", PrincipalID: adminID, Relation: "operator"}},
		describeAssignments(t, resp.GetAssignments()),
	)
}

func TestPermissions_Server_SameAdd(t *testing.T) {
	c := sharedClient

	user := MustProvisionUser(t, c)

	operator := []managementv1.ServerAssignment{userAssignment[managementv1.ServerAssignment](t, "operator", user.GetId())}
	require.NoError(t, c.Permissions.Server.Grant(t.Context(), operator))
	revokeOnCleanup(t, c, operator)

	// re-applying the same write fails on the server
	err := c.Permissions.Server.Grant(t.Context(), operator)
	var apiErr *client.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, http.StatusConflict, apiErr.StatusCode)
	assert.Contains(t, errorBody(err), "TupleAlreadyExistsError")
}

func TestPermissions_Server_GetAccess_UserFilter(t *testing.T) {
	c := sharedClient

	user := MustProvisionUser(t, c)
	byUser := &client.GetServerAccessOptions{PrincipalUser: user.GetId()}

	// fresh user has no allowed actions on the server
	resp, err := c.Permissions.Server.Access(t.Context(), byUser)
	require.NoError(t, err)
	assert.Empty(t, resp.GetAllowedActions())

	// granting admin should expand the allowed-actions set
	admin := []managementv1.ServerAssignment{userAssignment[managementv1.ServerAssignment](t, "admin", user.GetId())}
	require.NoError(t, c.Permissions.Server.Grant(t.Context(), admin))
	revokeOnCleanup(t, c, admin)

	resp, err = c.Permissions.Server.Access(t.Context(), byUser)
	require.NoError(t, err)
	assertAllowedActionsSubset(t, serverAdminActions, resp.GetAllowedActions(), managementv1.AllowedServerActionEnumValues)
}

func TestPermissions_Server_GetAccess_RoleFilter(t *testing.T) {
	c := sharedClient

	role := MustCreateRole(t, c, defaultProjectID)
	byRole := &client.GetServerAccessOptions{PrincipalRole: role.GetId()}

	// fresh role has no allowed actions on the server
	resp, err := c.Permissions.Server.Access(t.Context(), byRole)
	require.NoError(t, err)
	assert.Empty(t, resp.GetAllowedActions())

	// granting admin should expand the allowed-actions set
	admin := []managementv1.ServerAssignment{roleAssignment[managementv1.ServerAssignment](t, "admin", role.GetId())}
	require.NoError(t, c.Permissions.Server.Grant(t.Context(), admin))
	revokeOnCleanup(t, c, admin)

	resp, err = c.Permissions.Server.Access(t.Context(), byRole)
	require.NoError(t, err)
	assertAllowedActionsSubset(t, serverAdminActions, resp.GetAllowedActions(), managementv1.AllowedServerActionEnumValues)
}

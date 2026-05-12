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
	"github.com/lakekeeper/go-lakekeeper/pkg/permissions"
)

func TestPermissions_Server_GetAccess(t *testing.T) {
	c := sharedClient

	resp, r, err := c.PermissionsOpenfgaAPI.GetServerAccess(t.Context()).Execute()
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, r.StatusCode)

	want := []managementv1.ServerAction{
		managementv1.ServerActionCreateProject,
		managementv1.ServerActionUpdateUsers,
		managementv1.ServerActionDeleteUsers,
		managementv1.ServerActionListUsers,
		managementv1.ServerActionGrantAdmin,
		managementv1.ServerActionProvisionUsers,
		managementv1.ServerActionReadAssignments,
	}
	assert.Subset(t, want, resp.AllowedActions)
}

func TestPermissions_Server_GetAssignments(t *testing.T) {
	c := sharedClient

	resp, r, err := c.PermissionsOpenfgaAPI.GetServerAssignments(t.Context()).Execute()
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, r.StatusCode)

	want := []permissions.AssignmentRow{{
		PrincipalType: "user",
		PrincipalID:   adminID,
		Relation:      "operator",
	}}
	assert.ElementsMatch(t, want, describeAssignments(t, resp.Assignments))
}

func TestPermissions_Server_Update(t *testing.T) {
	c := sharedClient

	user := MustProvisionUser(t, c)

	resp, _, err := c.PermissionsOpenfgaAPI.GetServerAssignments(t.Context()).Execute()
	require.NoError(t, err)
	assert.ElementsMatch(t,
		[]permissions.AssignmentRow{{PrincipalType: "user", PrincipalID: adminID, Relation: "operator"}},
		describeAssignments(t, resp.Assignments),
	)

	// add admin assignment for the new user
	addReq := managementv1.NewUpdateServerAssignmentsRequest()
	addReq.Writes = append(addReq.Writes, userAssignment[managementv1.ServerAssignment](t, "admin", user.Id))

	r, err := c.PermissionsOpenfgaAPI.UpdateServerAssignments(t.Context()).UpdateServerAssignmentsRequest(*addReq).Execute()
	require.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, r.StatusCode)

	resp, _, err = c.PermissionsOpenfgaAPI.GetServerAssignments(t.Context()).Execute()
	require.NoError(t, err)
	assert.ElementsMatch(t,
		[]permissions.AssignmentRow{
			{PrincipalType: "user", PrincipalID: adminID, Relation: "operator"},
			{PrincipalType: "user", PrincipalID: user.Id, Relation: "admin"},
		},
		describeAssignments(t, resp.Assignments),
	)

	// remove the admin assignment
	delReq := managementv1.NewUpdateServerAssignmentsRequest()
	delReq.Deletes = append(delReq.Deletes, userAssignment[managementv1.ServerAssignment](t, "admin", user.Id))

	r, err = c.PermissionsOpenfgaAPI.UpdateServerAssignments(t.Context()).UpdateServerAssignmentsRequest(*delReq).Execute()
	require.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, r.StatusCode)

	resp, _, err = c.PermissionsOpenfgaAPI.GetServerAssignments(t.Context()).Execute()
	require.NoError(t, err)
	assert.ElementsMatch(t,
		[]permissions.AssignmentRow{{PrincipalType: "user", PrincipalID: adminID, Relation: "operator"}},
		describeAssignments(t, resp.Assignments),
	)
}

func TestPermissions_Server_SameAdd(t *testing.T) {
	c := sharedClient

	user := MustProvisionUser(t, c)

	req := managementv1.NewUpdateServerAssignmentsRequest()
	req.Writes = append(req.Writes, userAssignment[managementv1.ServerAssignment](t, "operator", user.Id))

	r, err := c.PermissionsOpenfgaAPI.UpdateServerAssignments(t.Context()).UpdateServerAssignmentsRequest(*req).Execute()
	require.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, r.StatusCode)
	// Server is a shared resource; undo the write so other tests asserting
	// exact assignment sets don't see a leftover tuple. Cleanups run LIFO,
	// so this fires before MustProvisionUser's user-delete cleanup.
	t.Cleanup(func() {
		delReq := managementv1.NewUpdateServerAssignmentsRequest()
		delReq.Deletes = append(delReq.Deletes, userAssignment[managementv1.ServerAssignment](t, "operator", user.Id))
		if _, err := c.PermissionsOpenfgaAPI.UpdateServerAssignments(context.Background()).UpdateServerAssignmentsRequest(*delReq).Execute(); err != nil {
			t.Errorf("undo server assignment: %v", err)
		}
	})

	// re-applying the same write fails on the server
	r, err = c.PermissionsOpenfgaAPI.UpdateServerAssignments(t.Context()).UpdateServerAssignmentsRequest(*req).Execute()
	require.Error(t, err)
	require.NotNil(t, r)
	assert.Equal(t, http.StatusConflict, r.StatusCode)
	assert.Contains(t, errorBody(err), "TupleAlreadyExistsError")
}

func TestPermissions_Server_GetAccess_UserFilter(t *testing.T) {
	c := sharedClient

	user := MustProvisionUser(t, c)

	// fresh user has no allowed actions on the server
	resp, _, err := c.PermissionsOpenfgaAPI.GetServerAccess(t.Context()).PrincipalUser(user.Id).Execute()
	require.NoError(t, err)
	assert.Empty(t, resp.AllowedActions)

	// granting admin should expand the allowed-actions set
	addReq := managementv1.NewUpdateServerAssignmentsRequest()
	addReq.Writes = append(addReq.Writes, userAssignment[managementv1.ServerAssignment](t, "admin", user.Id))
	_, err = c.PermissionsOpenfgaAPI.UpdateServerAssignments(t.Context()).UpdateServerAssignmentsRequest(*addReq).Execute()
	require.NoError(t, err)
	// Server is shared; remove the tuple so this suite is re-runnable
	// against an already-up stack. Cleanups run LIFO, so this fires before
	// MustProvisionUser's user-delete cleanup — the tuple's principal is
	// still valid at delete time.
	t.Cleanup(func() {
		delReq := managementv1.NewUpdateServerAssignmentsRequest()
		delReq.Deletes = append(delReq.Deletes, userAssignment[managementv1.ServerAssignment](t, "admin", user.Id))
		if _, err := c.PermissionsOpenfgaAPI.UpdateServerAssignments(context.Background()).UpdateServerAssignmentsRequest(*delReq).Execute(); err != nil {
			t.Errorf("undo server admin assignment: %v", err)
		}
	})

	resp, _, err = c.PermissionsOpenfgaAPI.GetServerAccess(t.Context()).PrincipalUser(user.Id).Execute()
	require.NoError(t, err)

	want := []managementv1.ServerAction{
		managementv1.ServerActionCreateProject,
		managementv1.ServerActionUpdateUsers,
		managementv1.ServerActionDeleteUsers,
		managementv1.ServerActionListUsers,
		managementv1.ServerActionGrantAdmin,
		managementv1.ServerActionProvisionUsers,
		managementv1.ServerActionReadAssignments,
	}
	assert.Subset(t, want, resp.AllowedActions)
}

func TestPermissions_Server_GetAccess_RoleFilter(t *testing.T) {
	c := sharedClient

	role := MustCreateRole(t, c, defaultProjectID)

	// fresh role has no allowed actions on the server
	resp, _, err := c.PermissionsOpenfgaAPI.GetServerAccess(t.Context()).PrincipalRole(role.Id).Execute()
	require.NoError(t, err)
	assert.Empty(t, resp.AllowedActions)

	// granting admin should expand the allowed-actions set
	addReq := managementv1.NewUpdateServerAssignmentsRequest()
	addReq.Writes = append(addReq.Writes, roleAssignment[managementv1.ServerAssignment](t, "admin", role.Id))
	_, err = c.PermissionsOpenfgaAPI.UpdateServerAssignments(t.Context()).UpdateServerAssignmentsRequest(*addReq).Execute()
	require.NoError(t, err)
	// Server is shared; see UserFilter test for cleanup rationale.
	t.Cleanup(func() {
		delReq := managementv1.NewUpdateServerAssignmentsRequest()
		delReq.Deletes = append(delReq.Deletes, roleAssignment[managementv1.ServerAssignment](t, "admin", role.Id))
		if _, err := c.PermissionsOpenfgaAPI.UpdateServerAssignments(context.Background()).UpdateServerAssignmentsRequest(*delReq).Execute(); err != nil {
			t.Errorf("undo server admin assignment: %v", err)
		}
	})

	resp, _, err = c.PermissionsOpenfgaAPI.GetServerAccess(t.Context()).PrincipalRole(role.Id).Execute()
	require.NoError(t, err)

	want := []managementv1.ServerAction{
		managementv1.ServerActionCreateProject,
		managementv1.ServerActionUpdateUsers,
		managementv1.ServerActionDeleteUsers,
		managementv1.ServerActionListUsers,
		managementv1.ServerActionGrantAdmin,
		managementv1.ServerActionProvisionUsers,
		managementv1.ServerActionReadAssignments,
	}
	assert.Subset(t, want, resp.AllowedActions)
}

//go:build integration

// Tests here all scope their writes to freshly-created warehouses, so they
// are nominally safe to parallelize. They are kept serial for now to match
// the rest of the *_permissions_test.go files (see e.g.
// server_permissions_test.go for the shared-resource cases that *cannot* be
// parallelized) — flip this once the helper invariants are tightened.
package integration

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	managementv1 "github.com/lakekeeper/go-lakekeeper/pkg/apis/management/v1"
	"github.com/lakekeeper/go-lakekeeper/pkg/permissions"
)

func TestPermissions_Warehouse_GetAuthzProps(t *testing.T) {
	c := sharedClient

	project := MustCreateProject(t, c)
	wh, _ := MustCreateWarehouse(t, c, project)

	resp, r, err := c.PermissionsOpenfgaAPI.GetWarehouseById(t.Context(), wh).Execute()
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, r.StatusCode)
	assert.False(t, resp.ManagedAccess)
}

func TestPermissions_Warehouse_SetManagedAccess(t *testing.T) {
	c := sharedClient

	project := MustCreateProject(t, c)
	wh, _ := MustCreateWarehouse(t, c, project)

	resp, r, err := c.PermissionsOpenfgaAPI.GetWarehouseById(t.Context(), wh).Execute()
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, r.StatusCode)
	assert.False(t, resp.ManagedAccess)

	setReq := managementv1.NewSetManagedAccessRequest(true)
	r, err = c.PermissionsOpenfgaAPI.SetWarehouseManagedAccess(t.Context(), wh).SetManagedAccessRequest(*setReq).Execute()
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, r.StatusCode)

	resp, _, err = c.PermissionsOpenfgaAPI.GetWarehouseById(t.Context(), wh).Execute()
	require.NoError(t, err)
	assert.True(t, resp.ManagedAccess)
}

func TestPermissions_Warehouse_GetAccess(t *testing.T) {
	c := sharedClient

	project := MustCreateProject(t, c)
	wh, _ := MustCreateWarehouse(t, c, project)

	resp, r, err := c.PermissionsOpenfgaAPI.GetWarehouseAccessById(t.Context(), wh).Execute()
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, r.StatusCode)

	want := []managementv1.WarehouseAction{
		managementv1.WarehouseActionCreateNamespace,
		managementv1.WarehouseActionDelete,
		managementv1.WarehouseActionModifyStorage,
		managementv1.WarehouseActionModifyStorageCredential,
		managementv1.WarehouseActionGetConfig,
		managementv1.WarehouseActionGetMetadata,
		managementv1.WarehouseActionListNamespaces,
		managementv1.WarehouseActionIncludeInList,
		managementv1.WarehouseActionDeactivate,
		managementv1.WarehouseActionActivate,
		managementv1.WarehouseActionRename,
		managementv1.WarehouseActionListDeletedTabulars,
		managementv1.WarehouseActionReadAssignments,
		managementv1.WarehouseActionGrantCreate,
		managementv1.WarehouseActionGrantDescribe,
		managementv1.WarehouseActionGrantModify,
		managementv1.WarehouseActionGrantSelect,
		managementv1.WarehouseActionGrantPassGrants,
		managementv1.WarehouseActionGrantManageGrants,
		managementv1.WarehouseActionChangeOwnership,
		managementv1.WarehouseActionGetAllTasks,
		managementv1.WarehouseActionControlAllTasks,
		managementv1.WarehouseActionSetProtection,
		managementv1.WarehouseActionGetEndpointStatistics,
	}
	assert.Subset(t, want, resp.AllowedActions)
}

func TestPermissions_Warehouse_GetAssignments(t *testing.T) {
	c := sharedClient

	project := MustCreateProject(t, c)
	wh, _ := MustCreateWarehouse(t, c, project)

	resp, r, err := c.PermissionsOpenfgaAPI.GetWarehouseAssignmentsById(t.Context(), wh).Execute()
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, r.StatusCode)

	assert.ElementsMatch(t,
		[]permissions.AssignmentRow{{PrincipalType: "user", PrincipalID: adminID, Relation: "ownership"}},
		describeAssignments(t, resp.Assignments),
	)
}

func TestPermissions_Warehouse_Update(t *testing.T) {
	c := sharedClient

	project := MustCreateProject(t, c)
	wh, _ := MustCreateWarehouse(t, c, project)
	user := MustProvisionUser(t, c)

	resp, _, err := c.PermissionsOpenfgaAPI.GetWarehouseAssignmentsById(t.Context(), wh).Execute()
	require.NoError(t, err)
	assert.ElementsMatch(t,
		[]permissions.AssignmentRow{{PrincipalType: "user", PrincipalID: adminID, Relation: "ownership"}},
		describeAssignments(t, resp.Assignments),
	)

	addReq := managementv1.NewUpdateWarehouseAssignmentsRequest()
	addReq.Writes = append(addReq.Writes, userAssignment[managementv1.WarehouseAssignment](t, "describe", user.Id))

	r, err := c.PermissionsOpenfgaAPI.UpdateWarehouseAssignmentsById(t.Context(), wh).UpdateWarehouseAssignmentsRequest(*addReq).Execute()
	require.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, r.StatusCode)

	resp, _, err = c.PermissionsOpenfgaAPI.GetWarehouseAssignmentsById(t.Context(), wh).Execute()
	require.NoError(t, err)
	assert.ElementsMatch(t,
		[]permissions.AssignmentRow{
			{PrincipalType: "user", PrincipalID: adminID, Relation: "ownership"},
			{PrincipalType: "user", PrincipalID: user.Id, Relation: "describe"},
		},
		describeAssignments(t, resp.Assignments),
	)

	delReq := managementv1.NewUpdateWarehouseAssignmentsRequest()
	delReq.Deletes = append(delReq.Deletes, userAssignment[managementv1.WarehouseAssignment](t, "describe", user.Id))

	r, err = c.PermissionsOpenfgaAPI.UpdateWarehouseAssignmentsById(t.Context(), wh).UpdateWarehouseAssignmentsRequest(*delReq).Execute()
	require.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, r.StatusCode)

	resp, _, err = c.PermissionsOpenfgaAPI.GetWarehouseAssignmentsById(t.Context(), wh).Execute()
	require.NoError(t, err)
	assert.ElementsMatch(t,
		[]permissions.AssignmentRow{{PrincipalType: "user", PrincipalID: adminID, Relation: "ownership"}},
		describeAssignments(t, resp.Assignments),
	)
}

func TestPermissions_Warehouse_SameAdd(t *testing.T) {
	c := sharedClient

	user := MustProvisionUser(t, c)
	project := MustCreateProject(t, c)
	wh, _ := MustCreateWarehouse(t, c, project)

	req := managementv1.NewUpdateWarehouseAssignmentsRequest()
	req.Writes = append(req.Writes, userAssignment[managementv1.WarehouseAssignment](t, "modify", user.Id))

	r, err := c.PermissionsOpenfgaAPI.UpdateWarehouseAssignmentsById(t.Context(), wh).UpdateWarehouseAssignmentsRequest(*req).Execute()
	require.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, r.StatusCode)

	r, err = c.PermissionsOpenfgaAPI.UpdateWarehouseAssignmentsById(t.Context(), wh).UpdateWarehouseAssignmentsRequest(*req).Execute()
	require.Error(t, err)
	require.NotNil(t, r)
	assert.Equal(t, http.StatusConflict, r.StatusCode)
	assert.Contains(t, errorBody(err), "TupleAlreadyExistsError")
}

func TestPermissions_Warehouse_Add_Role(t *testing.T) {
	c := sharedClient

	project := MustCreateProject(t, c)
	wh, _ := MustCreateWarehouse(t, c, project)
	role := MustCreateRole(t, c, project)

	addReq := managementv1.NewUpdateWarehouseAssignmentsRequest()
	addReq.Writes = append(addReq.Writes, roleAssignment[managementv1.WarehouseAssignment](t, "describe", role.Id))

	r, err := c.PermissionsOpenfgaAPI.UpdateWarehouseAssignmentsById(t.Context(), wh).UpdateWarehouseAssignmentsRequest(*addReq).Execute()
	require.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, r.StatusCode)

	resp, _, err := c.PermissionsOpenfgaAPI.GetWarehouseAssignmentsById(t.Context(), wh).Execute()
	require.NoError(t, err)
	assert.ElementsMatch(t,
		[]permissions.AssignmentRow{
			{PrincipalType: "user", PrincipalID: adminID, Relation: "ownership"},
			{PrincipalType: "role", PrincipalID: role.Id, Relation: "describe"},
		},
		describeAssignments(t, resp.Assignments),
	)
}

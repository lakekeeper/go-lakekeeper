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
	"github.com/lakekeeper/go-lakekeeper/pkg/client"
	"github.com/lakekeeper/go-lakekeeper/pkg/permissions"
)

func TestPermissions_Warehouse_GetAuthzProps(t *testing.T) {
	c := sharedClient

	project := MustCreateProject(t, c)
	wh, _ := MustCreateWarehouse(t, c, project)

	resp, r, err := c.PermissionsOpenfgaAPI.GetWarehouseById(t.Context(), wh).Execute()
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, r.StatusCode)
	assert.False(t, resp.GetManagedAccess())
}

func TestPermissions_Warehouse_SetManagedAccess(t *testing.T) {
	c := sharedClient

	project := MustCreateProject(t, c)
	wh, _ := MustCreateWarehouse(t, c, project)

	resp, r, err := c.PermissionsOpenfgaAPI.GetWarehouseById(t.Context(), wh).Execute()
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, r.StatusCode)
	assert.False(t, resp.GetManagedAccess())

	setReq := managementv1.NewSetManagedAccessRequest(true)
	r, err = c.PermissionsOpenfgaAPI.SetWarehouseManagedAccess(t.Context(), wh).SetManagedAccessRequest(*setReq).Execute()
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, r.StatusCode)

	resp, _, err = c.PermissionsOpenfgaAPI.GetWarehouseById(t.Context(), wh).Execute()
	require.NoError(t, err)
	assert.True(t, resp.GetManagedAccess())
}

func TestPermissions_Warehouse_GetAccess(t *testing.T) {
	c := sharedClient

	project := MustCreateProject(t, c)
	wh, _ := MustCreateWarehouse(t, c, project)

	resp, err := c.Permissions.Warehouses.Access(t.Context(), wh, nil)
	require.NoError(t, err)

	want := []managementv1.WarehouseAction{
		managementv1.WAREHOUSEACTION_CREATE_NAMESPACE,
		managementv1.WAREHOUSEACTION_DELETE,
		managementv1.WAREHOUSEACTION_MODIFY_STORAGE,
		managementv1.WAREHOUSEACTION_MODIFY_STORAGE_CREDENTIAL,
		managementv1.WAREHOUSEACTION_GET_CONFIG,
		managementv1.WAREHOUSEACTION_GET_METADATA,
		managementv1.WAREHOUSEACTION_LIST_NAMESPACES,
		managementv1.WAREHOUSEACTION_INCLUDE_IN_LIST,
		managementv1.WAREHOUSEACTION_DEACTIVATE,
		managementv1.WAREHOUSEACTION_ACTIVATE,
		managementv1.WAREHOUSEACTION_RENAME,
		managementv1.WAREHOUSEACTION_LIST_DELETED_TABULARS,
		managementv1.WAREHOUSEACTION_READ_ASSIGNMENTS,
		managementv1.WAREHOUSEACTION_GRANT_CREATE,
		managementv1.WAREHOUSEACTION_GRANT_DESCRIBE,
		managementv1.WAREHOUSEACTION_GRANT_MODIFY,
		managementv1.WAREHOUSEACTION_GRANT_SELECT,
		managementv1.WAREHOUSEACTION_GRANT_PASS_GRANTS,
		managementv1.WAREHOUSEACTION_GRANT_MANAGE_GRANTS,
		managementv1.WAREHOUSEACTION_CHANGE_OWNERSHIP,
		managementv1.WAREHOUSEACTION_GET_ALL_TASKS,
		managementv1.WAREHOUSEACTION_CONTROL_ALL_TASKS,
		managementv1.WAREHOUSEACTION_SET_PROTECTION,
		managementv1.WAREHOUSEACTION_GET_ENDPOINT_STATISTICS,
		managementv1.WAREHOUSEACTION_ACCEPT_MOVED_NAMESPACE,
		managementv1.WAREHOUSEACTION_GRANT_MANAGE_TAGS,
		managementv1.WAREHOUSEACTION_SET_FORMAT_VERSION_POLICY,
		managementv1.WAREHOUSEACTION_GRANT_WRITE,
	}
	assertAllowedActionsSubset(t, want, resp.GetAllowedActions(), managementv1.AllowedWarehouseActionEnumValues)
}

func TestPermissions_Warehouse_GetAssignments(t *testing.T) {
	c := sharedClient

	project := MustCreateProject(t, c)
	wh, _ := MustCreateWarehouse(t, c, project)

	resp, err := c.Permissions.Warehouses.Assignments(t.Context(), wh, nil)
	require.NoError(t, err)

	assert.ElementsMatch(t,
		[]permissions.AssignmentRow{{PrincipalType: "user", PrincipalID: adminID, Relation: "ownership"}},
		describeAssignments(t, resp.GetAssignments()),
	)
}

func TestPermissions_Warehouse_Update(t *testing.T) {
	c := sharedClient

	project := MustCreateProject(t, c)
	wh, _ := MustCreateWarehouse(t, c, project)
	user := MustProvisionUser(t, c)

	resp, err := c.Permissions.Warehouses.Assignments(t.Context(), wh, nil)
	require.NoError(t, err)
	assert.ElementsMatch(t,
		[]permissions.AssignmentRow{{PrincipalType: "user", PrincipalID: adminID, Relation: "ownership"}},
		describeAssignments(t, resp.GetAssignments()),
	)

	describe := []managementv1.WarehouseAssignment{userAssignment[managementv1.WarehouseAssignment](t, "describe", user.GetId())}
	require.NoError(t, c.Permissions.Warehouses.Grant(t.Context(), wh, describe))

	resp, err = c.Permissions.Warehouses.Assignments(t.Context(), wh, nil)
	require.NoError(t, err)
	assert.ElementsMatch(t,
		[]permissions.AssignmentRow{
			{PrincipalType: "user", PrincipalID: adminID, Relation: "ownership"},
			{PrincipalType: "user", PrincipalID: user.GetId(), Relation: "describe"},
		},
		describeAssignments(t, resp.GetAssignments()),
	)

	require.NoError(t, c.Permissions.Warehouses.Revoke(t.Context(), wh, describe))

	resp, err = c.Permissions.Warehouses.Assignments(t.Context(), wh, nil)
	require.NoError(t, err)
	assert.ElementsMatch(t,
		[]permissions.AssignmentRow{{PrincipalType: "user", PrincipalID: adminID, Relation: "ownership"}},
		describeAssignments(t, resp.GetAssignments()),
	)
}

func TestPermissions_Warehouse_SameAdd(t *testing.T) {
	c := sharedClient

	user := MustProvisionUser(t, c)
	project := MustCreateProject(t, c)
	wh, _ := MustCreateWarehouse(t, c, project)

	modify := []managementv1.WarehouseAssignment{userAssignment[managementv1.WarehouseAssignment](t, "modify", user.GetId())}
	require.NoError(t, c.Permissions.Warehouses.Grant(t.Context(), wh, modify))

	err := c.Permissions.Warehouses.Grant(t.Context(), wh, modify)
	var apiErr *client.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, http.StatusConflict, apiErr.StatusCode)
	assert.Contains(t, errorBody(err), "TupleAlreadyExistsError")
}

func TestPermissions_Warehouse_Add_Role(t *testing.T) {
	c := sharedClient

	project := MustCreateProject(t, c)
	wh, _ := MustCreateWarehouse(t, c, project)
	role := MustCreateRole(t, c, project)

	require.NoError(t, c.Permissions.Warehouses.Grant(t.Context(), wh,
		[]managementv1.WarehouseAssignment{roleAssignment[managementv1.WarehouseAssignment](t, "describe", role.GetId())}))

	resp, err := c.Permissions.Warehouses.Assignments(t.Context(), wh, nil)
	require.NoError(t, err)
	assert.ElementsMatch(t,
		[]permissions.AssignmentRow{
			{PrincipalType: "user", PrincipalID: adminID, Relation: "ownership"},
			{PrincipalType: "role", PrincipalID: role.GetId(), Relation: "describe"},
		},
		describeAssignments(t, resp.GetAssignments()),
	)
}

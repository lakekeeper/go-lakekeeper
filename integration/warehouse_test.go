//go:build integration

package integration

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	managementv1 "github.com/lakekeeper/go-lakekeeper/pkg/apis/management/v1"
	"github.com/lakekeeper/go-lakekeeper/pkg/client"
	"github.com/lakekeeper/go-lakekeeper/pkg/storage/credential"
	"github.com/lakekeeper/go-lakekeeper/pkg/storage/profile"
)

func TestWarehouse_Create_Default(t *testing.T) {
	t.Parallel()
	c := sharedClient

	sp := profile.NewS3Profile(
		"testacc", "eu-local-1",
		profile.WithS3Endpoint("http://minio:9000/"),
		profile.WithS3PathStyleAccess(),
	)
	sc := credential.NewS3AccessKey("minio-root-user", "minio-root-password")

	name := randomName("test-wh-default")
	req := managementv1.NewCreateWarehouseRequest(sp, name)
	req.SetStorageCredential(sc)

	wh, err := c.Warehouses.Create(t.Context(), req, client.WithProject(defaultProjectID))
	require.NoError(t, err)
	require.NotNil(t, wh)

	t.Cleanup(func() {
		if err := c.Warehouses.Delete(context.Background(), wh.GetWarehouseId()); err != nil {
			t.Errorf("delete warehouse: %v", err)
		}
	})

	got, err := c.Warehouses.Get(t.Context(), wh.GetWarehouseId())
	require.NoError(t, err)
	require.NotNil(t, got)

	assert.Equal(t, wh.GetWarehouseId(), got.GetWarehouseId())
	assert.Equal(t, name, got.GetName())
	assert.Equal(t, defaultProjectID, got.GetProjectId())
	assert.Equal(t, managementv1.WAREHOUSESTATUS_ACTIVE, got.GetStatus())
	assert.False(t, got.GetProtected())
	s3, ok := profile.AsS3(got.GetStorageProfile())
	require.True(t, ok, "the storage profile must be S3")
	assert.Equal(t, "testacc", s3.GetBucket())
	assert.Equal(t, "eu-local-1", s3.GetRegion())
}

// Not parallel: TestWarehouse_Create_Default also creates a warehouse on the
// same bucket and prefix in the default project, and the server rejects
// overlapping storage profiles within a project.
func TestWarehouse_Create_WithoutProjectUsesDefaultProject(t *testing.T) {
	c := sharedClient
	whID, _ := MustCreateWarehouse(t, c, "")

	got, err := c.Warehouses.Get(t.Context(), whID)
	require.NoError(t, err)
	assert.Equal(t, defaultProjectID, got.GetProjectId(), "a create without a project must land in the default project")
}

func TestWarehouse_Get_NotFoundIsAPIError(t *testing.T) {
	t.Parallel()
	c := sharedClient

	_, err := c.Warehouses.Get(t.Context(), uuid.NewString())

	var apiErr *client.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, http.StatusNotFound, apiErr.StatusCode)
}

func warehouseIDs(resp *managementv1.ListWarehousesResponse) []string {
	ids := make([]string, 0, len(resp.GetWarehouses()))
	for _, wh := range resp.GetWarehouses() {
		ids = append(ids, wh.GetWarehouseId())
	}
	return ids
}

func TestWarehouse_List_ProjectScoping(t *testing.T) {
	t.Parallel()
	c := sharedClient
	project := MustCreateProject(t, c)
	whID, _ := MustCreateWarehouse(t, c, project)

	scoped, err := c.Warehouses.List(t.Context(), nil, client.WithProject(project))
	require.NoError(t, err)
	assert.Contains(t, warehouseIDs(scoped), whID, "a list scoped to the project must include its warehouse")

	unscoped, err := c.Warehouses.List(t.Context(), nil)
	require.NoError(t, err)
	assert.NotContains(t, warehouseIDs(unscoped), whID, "a list without a project must use the default project")
}

func TestWarehouse_List_StatusFilter(t *testing.T) {
	t.Parallel()
	c := sharedClient
	project := MustCreateProject(t, c)
	whID, _ := MustCreateWarehouse(t, c, project)

	active, err := c.Warehouses.List(t.Context(), &client.ListWarehousesOptions{
		Status: []managementv1.WarehouseStatus{managementv1.WAREHOUSESTATUS_ACTIVE},
	}, client.WithProject(project))
	require.NoError(t, err)
	assert.Contains(t, warehouseIDs(active), whID)

	inactive, err := c.Warehouses.List(t.Context(), &client.ListWarehousesOptions{
		Status: []managementv1.WarehouseStatus{managementv1.WAREHOUSESTATUS_INACTIVE},
	}, client.WithProject(project))
	require.NoError(t, err)
	assert.NotContains(t, warehouseIDs(inactive), whID, "an active warehouse must not match the inactive filter")
}

func TestWarehouse_Create_NewProject(t *testing.T) {
	t.Parallel()
	c := sharedClient

	p, err := c.Projects.Create(t.Context(), managementv1.NewCreateProjectRequest("test-project-warehouse-create"))
	require.NoError(t, err)
	require.NotNil(t, p)

	sp := profile.NewS3Profile(
		"testacc", "eu-local-1",
		profile.WithS3Endpoint("http://minio:9000/"),
		profile.WithS3PathStyleAccess(),
	)
	sc := credential.NewS3AccessKey("minio-root-user", "minio-root-password")

	req := managementv1.NewCreateWarehouseRequest(sp, "test")
	req.SetStorageCredential(sc)

	wh, err := c.Warehouses.Create(t.Context(), req, client.WithProject(p.GetProjectId()))
	require.NoError(t, err)
	require.NotNil(t, wh)
	assert.Equal(t, p.GetProjectId(), wh.GetProjectId())

	t.Cleanup(func() {
		if err := c.Warehouses.Delete(context.Background(), wh.GetWarehouseId()); err != nil {
			t.Errorf("delete warehouse: %v", err)
		}

		if err := c.Projects.Delete(context.Background(), client.WithProject(p.GetProjectId())); err != nil {
			t.Errorf("delete project: %v", err)
		}
	})
}

func TestWarehouse_Statistics(t *testing.T) {
	t.Parallel()
	c := sharedClient

	project := MustCreateProject(t, c)
	whID, _ := MustCreateWarehouse(t, c, project)

	resp, err := c.Warehouses.Statistics(t.Context(), whID, &client.GetWarehouseStatisticsOptions{PageSize: 10})
	require.NoError(t, err)

	// Computed timestamps and counts are non-deterministic; assert presence
	// and identity-of-warehouse only.
	assert.NotEmpty(t, resp.GetStats())
	assert.LessOrEqual(t, len(resp.GetStats()), 10)
	assert.Equal(t, whID, resp.GetWarehouseIdent())
}

func TestWarehouse_Rename(t *testing.T) {
	t.Parallel()
	c := sharedClient

	project := MustCreateProject(t, c)
	whID, _ := MustCreateWarehouse(t, c, project)

	newName := randomName("renamed-wh")
	resp, err := c.Warehouses.Rename(t.Context(), whID, managementv1.NewRenameWarehouseRequest(newName))
	require.NoError(t, err)
	assert.Equal(t, newName, resp.GetName())

	got, err := c.Warehouses.Get(t.Context(), whID)
	require.NoError(t, err)
	assert.Equal(t, newName, got.GetName())
}

func TestWarehouse_UpdateDeleteProfile(t *testing.T) {
	t.Parallel()
	c := sharedClient

	project := MustCreateProject(t, c)
	whID, _ := MustCreateWarehouse(t, c, project)

	resp, err := c.Warehouses.UpdateDeleteProfile(t.Context(), whID,
		managementv1.NewUpdateWarehouseDeleteProfileRequest(profile.NewSoftDeleteProfile(2*time.Hour)))
	require.NoError(t, err)
	soft, ok := profile.AsSoftDeleteProfile(resp.GetDeleteProfile())
	require.True(t, ok, "the response must carry the soft profile, got %+v", resp.GetDeleteProfile())
	assert.Equal(t, int64(7200), soft.GetExpirationSeconds())

	got, err := c.Warehouses.Get(t.Context(), whID)
	require.NoError(t, err)
	soft, ok = profile.AsSoftDeleteProfile(got.GetDeleteProfile())
	require.True(t, ok, "the soft profile must be persisted")
	assert.Equal(t, int64(7200), soft.GetExpirationSeconds())

	resp, err = c.Warehouses.UpdateDeleteProfile(t.Context(), whID,
		managementv1.NewUpdateWarehouseDeleteProfileRequest(profile.NewHardDeleteProfile()))
	require.NoError(t, err)
	_, ok = profile.AsHardDeleteProfile(resp.GetDeleteProfile())
	assert.True(t, ok, "the response must carry the hard profile, got %+v", resp.GetDeleteProfile())
}

func TestWarehouse_DeactivateActivate(t *testing.T) {
	t.Parallel()
	c := sharedClient

	project := MustCreateProject(t, c)
	whID, _ := MustCreateWarehouse(t, c, project)

	require.NoError(t, c.Warehouses.Deactivate(t.Context(), whID))

	got, err := c.Warehouses.Get(t.Context(), whID)
	require.NoError(t, err)
	assert.Equal(t, managementv1.WAREHOUSESTATUS_INACTIVE, got.GetStatus())

	require.NoError(t, c.Warehouses.Activate(t.Context(), whID))

	got, err = c.Warehouses.Get(t.Context(), whID)
	require.NoError(t, err)
	assert.Equal(t, managementv1.WAREHOUSESTATUS_ACTIVE, got.GetStatus())
}

func TestWarehouse_SetProtection(t *testing.T) {
	t.Parallel()
	c := sharedClient

	project := MustCreateProject(t, c)
	whID, _ := MustCreateWarehouse(t, c, project)

	resp, err := c.Warehouses.SetProtection(t.Context(), whID, managementv1.NewSetProtectionRequest(true))
	require.NoError(t, err)
	assert.True(t, resp.GetProtected())
	assert.False(t, resp.GetUpdatedAt().IsZero())

	// Unprotect before MustCreateWarehouse's delete cleanup runs (cleanups
	// are LIFO, so this one fires first). Without it, deletion 409s on the
	// protected warehouse and the project cleanup also fails.
	t.Cleanup(func() {
		if _, err := c.Warehouses.SetProtection(context.Background(), whID, managementv1.NewSetProtectionRequest(false)); err != nil {
			t.Errorf("unprotect warehouse: %v", err)
		}
	})

	err = c.Warehouses.Delete(t.Context(), whID)
	var apiErr *client.APIError
	require.ErrorAs(t, err, &apiErr, "deleting a protected warehouse must fail")
	assert.Equal(t, http.StatusConflict, apiErr.StatusCode)
}

//go:build integration

package integration

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	managementv1 "github.com/lakekeeper/go-lakekeeper/pkg/apis/management/v1"
	"github.com/lakekeeper/go-lakekeeper/pkg/storage/credential"
	"github.com/lakekeeper/go-lakekeeper/pkg/storage/profile"
)

func TestWarehouse_Create_Default(t *testing.T) {
	t.Parallel()
	c := sharedClient

	sp := managementv1.StorageProfileS3AsStorageProfile(profile.NewS3Profile(
		"testacc", "eu-local-1",
		profile.WithS3Endpoint("http://minio:9000/"),
		profile.WithS3PathStyleAccess(),
	))
	sc := credential.NewS3AccessKey("minio-root-user", "minio-root-password")

	name := randomName("test-wh-default")
	req := managementv1.NewCreateWarehouseRequest(sp, name)
	req.SetProjectId(defaultProjectID)
	req.SetStorageCredential(sc)

	wh, r, err := c.WarehouseAPI.CreateWarehouse(t.Context()).CreateWarehouseRequest(*req).Execute()
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, r.StatusCode)
	require.NotNil(t, wh)

	t.Cleanup(func() {
		r, err := c.WarehouseAPI.DeleteWarehouse(context.Background(), wh.WarehouseId).Execute()
		if err != nil {
			t.Errorf("delete warehouse: %v", err)
			return
		}
		assert.Equal(t, http.StatusNoContent, r.StatusCode)
	})

	got, _, err := c.WarehouseAPI.GetWarehouse(t.Context(), wh.WarehouseId).Execute()
	require.NoError(t, err)
	require.NotNil(t, got)

	assert.Equal(t, wh.WarehouseId, got.WarehouseId)
	assert.Equal(t, name, got.Name)
	assert.Equal(t, defaultProjectID, got.ProjectId)
	assert.Equal(t, managementv1.WAREHOUSESTATUS_ACTIVE, got.Status)
	assert.False(t, got.Protected)
	require.NotNil(t, got.StorageProfile.StorageProfileS3)
	assert.Equal(t, "testacc", got.StorageProfile.StorageProfileS3.Bucket)
	assert.Equal(t, "eu-local-1", got.StorageProfile.StorageProfileS3.Region)
}

func TestWarehouse_Create_NewProject(t *testing.T) {
	t.Parallel()
	c := sharedClient

	pReq := managementv1.NewCreateProjectRequest("test-project-warehouse-create")
	p, _, err := c.ProjectAPI.CreateProject(t.Context()).CreateProjectRequest(*pReq).Execute()
	require.NoError(t, err)
	require.NotNil(t, p)

	sp := managementv1.StorageProfileS3AsStorageProfile(profile.NewS3Profile(
		"testacc", "eu-local-1",
		profile.WithS3Endpoint("http://minio:9000/"),
		profile.WithS3PathStyleAccess(),
	))
	sc := credential.NewS3AccessKey("minio-root-user", "minio-root-password")

	req := managementv1.NewCreateWarehouseRequest(sp, "test")
	req.SetProjectId(p.ProjectId)
	req.SetStorageCredential(sc)

	wh, r, err := c.WarehouseAPI.CreateWarehouse(t.Context()).CreateWarehouseRequest(*req).Execute()
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, r.StatusCode)
	require.NotNil(t, wh)

	t.Cleanup(func() {
		r, err := c.WarehouseAPI.DeleteWarehouse(context.Background(), wh.WarehouseId).Execute()
		if err != nil {
			t.Errorf("delete warehouse: %v", err)
		} else {
			assert.Equal(t, http.StatusNoContent, r.StatusCode)
		}

		r, err = c.ProjectAPI.DeleteProject(context.Background()).XProjectId(p.ProjectId).Execute()
		if err != nil {
			t.Errorf("delete project: %v", err)
			return
		}
		assert.Equal(t, http.StatusNoContent, r.StatusCode)
	})
}

func TestWarehouse_Statistics(t *testing.T) {
	t.Parallel()
	c := sharedClient

	project := MustCreateProject(t, c)
	whID, _ := MustCreateWarehouse(t, c, project)

	resp, r, err := c.WarehouseAPI.GetWarehouseStatistics(t.Context(), whID).Execute()
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, r.StatusCode)

	// Computed timestamps and counts are non-deterministic; assert presence
	// and identity-of-warehouse only.
	assert.NotEmpty(t, resp.Stats)
	assert.Equal(t, whID, resp.WarehouseIdent)
}

func TestWarehouse_Rename(t *testing.T) {
	t.Parallel()
	c := sharedClient

	project := MustCreateProject(t, c)
	whID, _ := MustCreateWarehouse(t, c, project)

	newName := randomName("renamed-wh")
	req := managementv1.NewRenameWarehouseRequest(newName)
	resp, r, err := c.WarehouseAPI.RenameWarehouse(t.Context(), whID).RenameWarehouseRequest(*req).Execute()
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, r.StatusCode)
	assert.Equal(t, newName, resp.Name)

	got, _, err := c.WarehouseAPI.GetWarehouse(t.Context(), whID).Execute()
	require.NoError(t, err)
	assert.Equal(t, newName, got.Name)
}

func TestWarehouse_DeactivateActivate(t *testing.T) {
	t.Parallel()
	c := sharedClient

	project := MustCreateProject(t, c)
	whID, _ := MustCreateWarehouse(t, c, project)

	r, err := c.WarehouseAPI.DeactivateWarehouse(t.Context(), whID).Execute()
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, r.StatusCode)

	got, _, err := c.WarehouseAPI.GetWarehouse(t.Context(), whID).Execute()
	require.NoError(t, err)
	assert.Equal(t, managementv1.WAREHOUSESTATUS_INACTIVE, got.Status)

	r, err = c.WarehouseAPI.ActivateWarehouse(t.Context(), whID).Execute()
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, r.StatusCode)

	got, _, err = c.WarehouseAPI.GetWarehouse(t.Context(), whID).Execute()
	require.NoError(t, err)
	assert.Equal(t, managementv1.WAREHOUSESTATUS_ACTIVE, got.Status)
}

func TestWarehouse_SetProtection(t *testing.T) {
	t.Parallel()
	c := sharedClient

	project := MustCreateProject(t, c)
	whID, _ := MustCreateWarehouse(t, c, project)

	req := managementv1.NewSetProtectionRequest(true)
	resp, r, err := c.WarehouseAPI.SetWarehouseProtection(t.Context(), whID).SetProtectionRequest(*req).Execute()
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, r.StatusCode)
	assert.True(t, resp.Protected)
	assert.True(t, resp.UpdatedAt.IsSet())

	// Unprotect before MustCreateWarehouse's delete cleanup runs (cleanups
	// are LIFO, so this one fires first). Without it, deletion 409s on the
	// protected warehouse and the project cleanup also fails.
	t.Cleanup(func() {
		unprotect := managementv1.NewSetProtectionRequest(false)
		if _, _, err := c.WarehouseAPI.SetWarehouseProtection(context.Background(), whID).SetProtectionRequest(*unprotect).Execute(); err != nil {
			t.Errorf("unprotect warehouse: %v", err)
		}
	})
}

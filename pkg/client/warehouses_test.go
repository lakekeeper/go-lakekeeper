package client_test

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	managementv1 "github.com/lakekeeper/go-lakekeeper/pkg/apis/management/v1"
	"github.com/lakekeeper/go-lakekeeper/pkg/client"
	"github.com/lakekeeper/go-lakekeeper/pkg/storage/profile"
	"github.com/lakekeeper/go-lakekeeper/pkg/testutil"
)

const projectA = "11111111-1111-1111-1111-111111111111"

func TestWarehouses_NilRequestsRejected(t *testing.T) {
	t.Parallel()
	_, c := testutil.ServerMux(t)

	_, err := c.Warehouses.Create(t.Context(), nil)
	require.ErrorContains(t, err, "request must not be nil")

	_, err = c.Warehouses.Rename(t.Context(), "id", nil)
	require.ErrorContains(t, err, "request must not be nil")

	_, err = c.Warehouses.SetProtection(t.Context(), "id", nil)
	require.ErrorContains(t, err, "request must not be nil")

	_, err = c.Warehouses.UpdateDeleteProfile(t.Context(), "id", nil)
	require.ErrorContains(t, err, "request must not be nil")
}

var emptyWarehouseList = map[string]any{"warehouses": []any{}}

func TestWarehouses_List_WithProjectSendsHeaderOnly(t *testing.T) {
	t.Parallel()
	mux, c := testutil.ServerMux(t)
	reqs := testutil.Respond(t, mux, "GET /management/v1/warehouse", http.StatusOK, emptyWarehouseList)

	_, err := c.Warehouses.List(t.Context(), nil, client.WithProject(projectA))

	require.NoError(t, err)
	got := reqs.Last(t)
	assert.Equal(t, projectA, got.Header.Get("x-project-id"))
	assert.False(t, got.Query.Has("projectId"), "the deprecated projectId query parameter must not be sent")
}

func TestWarehouses_List_WithoutProjectSendsNoProject(t *testing.T) {
	t.Parallel()
	mux, c := testutil.ServerMux(t)
	reqs := testutil.Respond(t, mux, "GET /management/v1/warehouse", http.StatusOK, emptyWarehouseList)

	_, err := c.Warehouses.List(t.Context(), nil)

	require.NoError(t, err)
	got := reqs.Last(t)
	assert.NotContains(t, got.Header, "X-Project-Id", "no project must be sent")
	assert.False(t, got.Query.Has("projectId"), "no project must be sent")
}

func TestWarehouses_List_StatusFilterReachesQuery(t *testing.T) {
	t.Parallel()
	mux, c := testutil.ServerMux(t)
	reqs := testutil.Respond(t, mux, "GET /management/v1/warehouse", http.StatusOK, emptyWarehouseList)

	_, err := c.Warehouses.List(t.Context(), &client.ListWarehousesOptions{
		Status: []managementv1.WarehouseStatus{managementv1.WAREHOUSESTATUS_ACTIVE, managementv1.WAREHOUSESTATUS_INACTIVE},
	})

	require.NoError(t, err)
	assert.Equal(t, []string{"active", "inactive"}, reqs.Last(t).Query["warehouseStatus"])
}

func TestWarehouses_List_EmptyOptionsSendNoFilter(t *testing.T) {
	t.Parallel()
	mux, c := testutil.ServerMux(t)
	reqs := testutil.Respond(t, mux, "GET /management/v1/warehouse", http.StatusOK, emptyWarehouseList)

	_, err := c.Warehouses.List(t.Context(), &client.ListWarehousesOptions{})

	require.NoError(t, err)
	assert.Empty(t, reqs.Last(t).Query, "an empty options struct must send no query parameters")
}

const listWarehousesBody = `{"warehouses":[{
  "id": "9d6651da-5f19-11f1-ad69-1b7104fe3cf7",
  "warehouse-id": "9d6651da-5f19-11f1-ad69-1b7104fe3cf7",
  "name": "test-wh",
  "project-id": "11111111-1111-1111-1111-111111111111",
  "storage-profile": {"type": "s3", "bucket": "testacc", "region": "eu-local-1", "flavor": "aws", "sts-enabled": false},
  "storage-credential-type": {"type": "s3", "credential-type": "access-key"},
  "delete-profile": {"type": "hard"},
  "status": "active",
  "protected": false,
  "allowed-format-versions": [1, 2, 3],
  "updated-at": null
}]}`

func TestWarehouses_List_ReturnsGeneratedResponse(t *testing.T) {
	t.Parallel()
	mux, c := testutil.ServerMux(t)
	testutil.Respond(t, mux, "GET /management/v1/warehouse", http.StatusOK, listWarehousesBody)

	got, err := c.Warehouses.List(t.Context(), nil, client.WithProject(projectA))

	require.NoError(t, err)
	require.Len(t, got.GetWarehouses(), 1)
	wh := got.GetWarehouses()[0]
	assert.Equal(t, "test-wh", wh.GetName())
	assert.Equal(t, projectA, wh.GetProjectId())
	assert.Equal(t, managementv1.WAREHOUSESTATUS_ACTIVE, wh.GetStatus())
}

func TestWarehouses_List_ServerErrorIsAPIError(t *testing.T) {
	t.Parallel()
	mux, c := testutil.ServerMux(t)
	testutil.Respond(t, mux, "GET /management/v1/warehouse", http.StatusBadRequest,
		`{"error":{"code":400,"message":"No project id provided","type":"ProjectIdMissing","stack":[]}}`)

	_, err := c.Warehouses.List(t.Context(), nil)

	require.ErrorContains(t, err, "list warehouses: 400 Bad Request")
	var apiErr *client.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, http.StatusBadRequest, apiErr.StatusCode)
	require.NotNil(t, apiErr.Model)
	assert.Equal(t, "ProjectIdMissing", apiErr.Model.GetType())
	var genErr *managementv1.GenericOpenAPIError
	require.ErrorAs(t, err, &genErr, "errors.As must reach the generated error")
}

const warehouseID = "9d6651da-5f19-11f1-ad69-1b7104fe3cf7"

const warehouseBody = `{
  "id": "9d6651da-5f19-11f1-ad69-1b7104fe3cf7",
  "warehouse-id": "9d6651da-5f19-11f1-ad69-1b7104fe3cf7",
  "name": "test-wh",
  "project-id": "11111111-1111-1111-1111-111111111111",
  "storage-profile": {"type": "s3", "bucket": "testacc", "region": "eu-local-1", "flavor": "aws", "sts-enabled": false},
  "delete-profile": {"type": "hard"},
  "status": "active",
  "protected": false,
  "allowed-format-versions": [1, 2, 3]
}`

func newCreateWarehouseRequest() *managementv1.CreateWarehouseRequest {
	sp := managementv1.StorageProfileS3AsStorageProfile(managementv1.NewStorageProfileS3("testacc", "eu-local-1", false, "s3"))
	return managementv1.NewCreateWarehouseRequest(sp, "test-wh")
}

func decodeWarehouseRequestBody(t *testing.T, got testutil.CapturedRequest) map[string]any {
	t.Helper()
	var body map[string]any
	require.NoError(t, json.Unmarshal(got.Body, &body))
	return body
}

func TestWarehouses_Create_WithProjectSendsHeaderOnly(t *testing.T) {
	t.Parallel()
	mux, c := testutil.ServerMux(t)
	reqs := testutil.Respond(t, mux, "POST /management/v1/warehouse", http.StatusCreated, warehouseBody)

	got, err := c.Warehouses.Create(t.Context(), newCreateWarehouseRequest(), client.WithProject(projectA))

	require.NoError(t, err)
	assert.Equal(t, warehouseID, got.GetWarehouseId())
	sent := reqs.Last(t)
	assert.Equal(t, projectA, sent.Header.Get("x-project-id"))
	body := decodeWarehouseRequestBody(t, sent)
	assert.Equal(t, "test-wh", body["warehouse-name"])
	assert.NotContains(t, body, "project-id", "the deprecated project-id body field must not be sent")
}

func TestWarehouses_Create_WithoutProjectSendsNoProject(t *testing.T) {
	t.Parallel()
	mux, c := testutil.ServerMux(t)
	reqs := testutil.Respond(t, mux, "POST /management/v1/warehouse", http.StatusCreated, warehouseBody)

	_, err := c.Warehouses.Create(t.Context(), newCreateWarehouseRequest())

	require.NoError(t, err)
	sent := reqs.Last(t)
	assert.NotContains(t, sent.Header, "X-Project-Id")
	assert.NotContains(t, decodeWarehouseRequestBody(t, sent), "project-id")
}

func TestWarehouses_Get_ReturnsWarehouse(t *testing.T) {
	t.Parallel()
	mux, c := testutil.ServerMux(t)
	reqs := testutil.Respond(t, mux, "GET /management/v1/warehouse/{id}", http.StatusOK, warehouseBody)

	got, err := c.Warehouses.Get(t.Context(), warehouseID)

	require.NoError(t, err)
	assert.Equal(t, "/management/v1/warehouse/"+warehouseID, reqs.Last(t).Path)
	assert.Equal(t, "test-wh", got.GetName())
	assert.Equal(t, projectA, got.GetProjectId())
	assert.Equal(t, managementv1.WAREHOUSESTATUS_ACTIVE, got.GetStatus())
}

func TestWarehouses_Get_NotFoundIsAPIError(t *testing.T) {
	t.Parallel()
	mux, c := testutil.ServerMux(t)
	testutil.Respond(t, mux, "GET /management/v1/warehouse/{id}", http.StatusNotFound,
		`{"error":{"code":404,"message":"Warehouse not found","type":"WarehouseNotFound","stack":[]}}`)

	_, err := c.Warehouses.Get(t.Context(), warehouseID)

	require.ErrorContains(t, err, "get warehouse: 404 Not Found")
	var apiErr *client.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, http.StatusNotFound, apiErr.StatusCode)
	require.NotNil(t, apiErr.Model)
	assert.Equal(t, "WarehouseNotFound", apiErr.Model.GetType())
}

func TestWarehouses_Rename_SendsNewName(t *testing.T) {
	t.Parallel()
	mux, c := testutil.ServerMux(t)
	reqs := testutil.Respond(t, mux, "POST /management/v1/warehouse/{id}/rename", http.StatusOK, warehouseBody)

	got, err := c.Warehouses.Rename(t.Context(), warehouseID, managementv1.NewRenameWarehouseRequest("test-wh"))

	require.NoError(t, err)
	sent := reqs.Last(t)
	assert.Equal(t, "/management/v1/warehouse/"+warehouseID+"/rename", sent.Path)
	assert.Equal(t, map[string]any{"new-name": "test-wh"}, decodeWarehouseRequestBody(t, sent))
	assert.Equal(t, "test-wh", got.GetName())
}

func TestWarehouses_ActivateDeactivate_PostToStatusPaths(t *testing.T) {
	t.Parallel()
	mux, c := testutil.ServerMux(t)
	activate := testutil.Respond(t, mux, "POST /management/v1/warehouse/{id}/activate", http.StatusOK, nil)
	deactivate := testutil.Respond(t, mux, "POST /management/v1/warehouse/{id}/deactivate", http.StatusOK, nil)

	require.NoError(t, c.Warehouses.Deactivate(t.Context(), warehouseID))
	require.NoError(t, c.Warehouses.Activate(t.Context(), warehouseID))

	assert.Equal(t, "/management/v1/warehouse/"+warehouseID+"/deactivate", deactivate.Last(t).Path)
	assert.Equal(t, "/management/v1/warehouse/"+warehouseID+"/activate", activate.Last(t).Path)
}

func TestWarehouses_Activate_ServerErrorIsAPIError(t *testing.T) {
	t.Parallel()
	mux, c := testutil.ServerMux(t)
	testutil.Respond(t, mux, "POST /management/v1/warehouse/{id}/activate", http.StatusForbidden,
		`{"error":{"code":403,"message":"Forbidden","type":"Forbidden","stack":[]}}`)

	err := c.Warehouses.Activate(t.Context(), warehouseID)

	var apiErr *client.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, http.StatusForbidden, apiErr.StatusCode)
	assert.ErrorContains(t, err, "activate warehouse: 403 Forbidden")
}

func TestWarehouses_SetProtection_SendsFlagAndReturnsState(t *testing.T) {
	t.Parallel()
	mux, c := testutil.ServerMux(t)
	reqs := testutil.Respond(t, mux, "POST /management/v1/warehouse/{id}/protection", http.StatusOK,
		`{"protected": true, "updated_at": "2026-05-04T10:00:00Z"}`)

	got, err := c.Warehouses.SetProtection(t.Context(), warehouseID, managementv1.NewSetProtectionRequest(true))

	require.NoError(t, err)
	sent := reqs.Last(t)
	assert.Equal(t, "/management/v1/warehouse/"+warehouseID+"/protection", sent.Path)
	assert.Equal(t, map[string]any{"protected": true}, decodeWarehouseRequestBody(t, sent))
	assert.True(t, got.GetProtected())
	assert.Equal(t, "2026-05-04T10:00:00Z", got.GetUpdatedAt().UTC().Format(time.RFC3339))
}

func TestWarehouses_UpdateDeleteProfile_SendsProfileAndReturnsWarehouse(t *testing.T) {
	t.Parallel()
	mux, c := testutil.ServerMux(t)
	reqs := testutil.Respond(t, mux, "POST /management/v1/warehouse/{id}/delete-profile", http.StatusOK, warehouseBody)

	got, err := c.Warehouses.UpdateDeleteProfile(t.Context(), warehouseID,
		managementv1.NewUpdateWarehouseDeleteProfileRequest(profile.NewSoftDeleteProfile(time.Hour)))

	require.NoError(t, err)
	sent := reqs.Last(t)
	assert.Equal(t, "/management/v1/warehouse/"+warehouseID+"/delete-profile", sent.Path)
	assert.Equal(t, map[string]any{"delete-profile": map[string]any{"type": "soft", "expiration-seconds": float64(3600)}},
		decodeWarehouseRequestBody(t, sent))
	assert.Equal(t, "test-wh", got.GetName())
}

func TestWarehouses_UpdateDeleteProfile_ServerErrorIsAPIError(t *testing.T) {
	t.Parallel()
	mux, c := testutil.ServerMux(t)
	testutil.Respond(t, mux, "POST /management/v1/warehouse/{id}/delete-profile", http.StatusBadRequest,
		`{"error":{"code":400,"message":"invalid expiration","type":"BadRequest","stack":[]}}`)

	_, err := c.Warehouses.UpdateDeleteProfile(t.Context(), warehouseID,
		managementv1.NewUpdateWarehouseDeleteProfileRequest(profile.NewHardDeleteProfile()))

	require.ErrorContains(t, err, "update warehouse delete profile: 400 Bad Request")
	var apiErr *client.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, http.StatusBadRequest, apiErr.StatusCode)
}

const warehouseStatisticsBody = `{
  "warehouse-ident": "9d6651da-5f19-11f1-ad69-1b7104fe3cf7",
  "next-page-token": "next-1",
  "stats": [{"number-of-tables": 7, "number-of-views": 3, "timestamp": "2026-05-04T10:00:00Z", "updated-at": "2026-05-04T10:00:00Z"}]
}`

func TestWarehouses_Statistics_PagingOptionsReachQuery(t *testing.T) {
	t.Parallel()
	mux, c := testutil.ServerMux(t)
	reqs := testutil.Respond(t, mux, "GET /management/v1/warehouse/{id}/statistics", http.StatusOK, warehouseStatisticsBody)

	got, err := c.Warehouses.Statistics(t.Context(), warehouseID, &client.GetWarehouseStatisticsOptions{
		PageSize:  25,
		PageToken: "page-2",
	})

	require.NoError(t, err)
	sent := reqs.Last(t)
	assert.Equal(t, "/management/v1/warehouse/"+warehouseID+"/statistics", sent.Path)
	assert.Equal(t, "25", sent.Query.Get("page_size"))
	assert.Equal(t, "page-2", sent.Query.Get("page_token"))
	assert.Equal(t, "next-1", got.GetNextPageToken())
	assert.Equal(t, warehouseID, got.GetWarehouseIdent())
	require.Len(t, got.GetStats(), 1)
	assert.Equal(t, int64(7), got.GetStats()[0].GetNumberOfTables())
}

func TestWarehouses_Statistics_NilOptionsSendNoQuery(t *testing.T) {
	t.Parallel()
	mux, c := testutil.ServerMux(t)
	reqs := testutil.Respond(t, mux, "GET /management/v1/warehouse/{id}/statistics", http.StatusOK, warehouseStatisticsBody)

	_, err := c.Warehouses.Statistics(t.Context(), warehouseID, nil)
	require.NoError(t, err)
	assert.Empty(t, reqs.Last(t).Query, "nil options must send no query parameters")

	_, err = c.Warehouses.Statistics(t.Context(), warehouseID, &client.GetWarehouseStatisticsOptions{})
	require.NoError(t, err)
	assert.Empty(t, reqs.Last(t).Query, "zero options must send no query parameters")
}

func TestWarehouses_Delete_SendsDelete(t *testing.T) {
	t.Parallel()
	mux, c := testutil.ServerMux(t)
	reqs := testutil.Respond(t, mux, "DELETE /management/v1/warehouse/{id}", http.StatusNoContent, nil)

	require.NoError(t, c.Warehouses.Delete(t.Context(), warehouseID))

	sent := reqs.Last(t)
	assert.Equal(t, "/management/v1/warehouse/"+warehouseID, sent.Path)
	assert.Empty(t, sent.Query)
}

func TestWarehouses_Delete_ConflictIsAPIError(t *testing.T) {
	t.Parallel()
	mux, c := testutil.ServerMux(t)
	testutil.Respond(t, mux, "DELETE /management/v1/warehouse/{id}", http.StatusConflict,
		`{"error":{"code":409,"message":"Warehouse is protected","type":"Protected","stack":[]}}`)

	err := c.Warehouses.Delete(t.Context(), warehouseID)

	var apiErr *client.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, http.StatusConflict, apiErr.StatusCode)
	assert.ErrorContains(t, err, "delete warehouse: 409 Conflict")
}

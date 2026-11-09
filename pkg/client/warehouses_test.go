package client_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	managementv1 "github.com/lakekeeper/go-lakekeeper/pkg/apis/management/v1"
	"github.com/lakekeeper/go-lakekeeper/pkg/client"
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

func TestWarehouses_List_EmptyProjectSendsNoProject(t *testing.T) {
	t.Parallel()
	mux, c := testutil.ServerMux(t)
	reqs := testutil.Respond(t, mux, "GET /management/v1/warehouse", http.StatusOK, emptyWarehouseList)

	_, err := c.Warehouses.List(t.Context(), nil, client.WithProject(""))

	require.NoError(t, err)
	assert.NotContains(t, reqs.Last(t).Header, "X-Project-Id", "an empty project must not be sent")
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

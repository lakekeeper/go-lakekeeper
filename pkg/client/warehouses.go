package client

import (
	"context"
	"errors"

	managementv1 "github.com/lakekeeper/go-lakekeeper/pkg/apis/management/v1"
)

// Warehouses is a one-call façade over WarehouseAPIService. The generator's
// fluent shape (`c.WarehouseAPI.X(ctx).Body(*req).Execute()`) stays available
// for cases that need fine-grained control — extra request options, the raw
// *http.Response, or query params not exposed here.
type Warehouses struct {
	api *managementv1.WarehouseAPIService
}

// Create sends a CreateWarehouse request and returns the created warehouse.
func (w *Warehouses) Create(ctx context.Context, req *managementv1.CreateWarehouseRequest) (*managementv1.GetWarehouseResponse, error) {
	if req == nil {
		return nil, errors.New("create warehouse: request must not be nil")
	}
	out, _, err := w.api.CreateWarehouse(ctx).CreateWarehouseRequest(*req).Execute()
	return out, wrapAPIError("create warehouse", err)
}

// Get retrieves a warehouse by id.
func (w *Warehouses) Get(ctx context.Context, id string) (*managementv1.GetWarehouseResponse, error) {
	out, _, err := w.api.GetWarehouse(ctx, id).Execute()
	return out, wrapAPIError("get warehouse", err)
}

// Delete removes a warehouse by id. The server enforces protection — to
// delete a protected warehouse, unprotect it first via SetProtection.
func (w *Warehouses) Delete(ctx context.Context, id string) error {
	_, err := w.api.DeleteWarehouse(ctx, id).Execute()
	return wrapAPIError("delete warehouse", err)
}

// List returns warehouses scoped to the given project.
func (w *Warehouses) List(ctx context.Context, projectID string) (*managementv1.ListWarehousesResponse, error) {
	out, _, err := w.api.ListWarehouses(ctx).ProjectId(projectID).Execute()
	return out, wrapAPIError("list warehouses", err)
}

// Rename changes a warehouse's display name.
func (w *Warehouses) Rename(ctx context.Context, id string, req *managementv1.RenameWarehouseRequest) (*managementv1.GetWarehouseResponse, error) {
	if req == nil {
		return nil, errors.New("rename warehouse: request must not be nil")
	}
	out, _, err := w.api.RenameWarehouse(ctx, id).RenameWarehouseRequest(*req).Execute()
	return out, wrapAPIError("rename warehouse", err)
}

// Activate transitions a warehouse from inactive to active.
func (w *Warehouses) Activate(ctx context.Context, id string) error {
	_, err := w.api.ActivateWarehouse(ctx, id).Execute()
	return wrapAPIError("activate warehouse", err)
}

// Deactivate transitions a warehouse from active to inactive. Inactive
// warehouses reject catalog operations until reactivated.
func (w *Warehouses) Deactivate(ctx context.Context, id string) error {
	_, err := w.api.DeactivateWarehouse(ctx, id).Execute()
	return wrapAPIError("deactivate warehouse", err)
}

// SetProtection toggles the protection flag on a warehouse. While protected,
// the warehouse cannot be deleted.
func (w *Warehouses) SetProtection(ctx context.Context, id string, req *managementv1.SetProtectionRequest) (*managementv1.ProtectionResponse, error) {
	if req == nil {
		return nil, errors.New("set warehouse protection: request must not be nil")
	}
	out, _, err := w.api.SetWarehouseProtection(ctx, id).SetProtectionRequest(*req).Execute()
	return out, wrapAPIError("set warehouse protection", err)
}

// Statistics fetches paged usage statistics for a warehouse.
func (w *Warehouses) Statistics(ctx context.Context, id string) (*managementv1.WarehouseStatisticsResponse, error) {
	out, _, err := w.api.GetWarehouseStatistics(ctx, id).Execute()
	return out, wrapAPIError("warehouse statistics", err)
}

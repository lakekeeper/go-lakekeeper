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

// Create creates a warehouse in a project and returns it. Name the project
// with WithProject; without it the server applies its default project.
//
// A project in req's deprecated ProjectId field (e.g. from a JSON config) is
// sent as the header, like WithProject; naming a different project in both
// fails with ErrProjectIDConflict. req itself is not modified.
func (w *Warehouses) Create(ctx context.Context, req *managementv1.CreateWarehouseRequest, callOpts ...CallOption) (*managementv1.GetWarehouseResponse, error) {
	const op = "create warehouse"
	if req == nil {
		return nil, errors.New(op + ": request must not be nil")
	}
	body := *req
	callOpts, err := projectFromBody(op, body.GetProjectId(), callOpts)
	if err != nil {
		return nil, err
	}
	body.UnsetProjectId()
	call, err := withProject(op, w.api.CreateWarehouse(ctx), callOpts)
	if err != nil {
		return nil, err
	}
	out, resp, err := call.CreateWarehouseRequest(body).Execute()
	return entityResult(op, out, resp, err)
}

// Get retrieves a warehouse by id.
func (w *Warehouses) Get(ctx context.Context, id string) (*managementv1.GetWarehouseResponse, error) {
	out, resp, err := w.api.GetWarehouse(ctx, id).Execute()
	return entityResult("get warehouse", out, resp, err)
}

// Delete removes a warehouse by id. The server enforces protection — to
// delete a protected warehouse, unprotect it first via SetProtection.
func (w *Warehouses) Delete(ctx context.Context, id string) error {
	resp, err := w.api.DeleteWarehouse(ctx, id).Execute()
	return WrapAPIError("delete warehouse", err, resp)
}

// ListWarehousesOptions holds the query parameters of Warehouses.List. A nil
// pointer or a zero field sends nothing for that parameter.
type ListWarehousesOptions struct {
	// Status keeps only warehouses in one of these states. Empty: the
	// server's default filter applies.
	Status []managementv1.WarehouseStatus
}

// List returns the warehouses of a project. Name the project with
// WithProject; without it the server applies its default project.
func (w *Warehouses) List(ctx context.Context, opts *ListWarehousesOptions, callOpts ...CallOption) (*managementv1.ListWarehousesResponse, error) {
	req, err := withProject("list warehouses", w.api.ListWarehouses(ctx), callOpts)
	if err != nil {
		return nil, err
	}
	if opts != nil && len(opts.Status) > 0 {
		req = req.WarehouseStatus(opts.Status)
	}
	out, resp, err := req.Execute()
	return entityResult("list warehouses", out, resp, err)
}

// Rename changes a warehouse's display name.
func (w *Warehouses) Rename(ctx context.Context, id string, req *managementv1.RenameWarehouseRequest) (*managementv1.GetWarehouseResponse, error) {
	if req == nil {
		return nil, errors.New("rename warehouse: request must not be nil")
	}
	out, resp, err := w.api.RenameWarehouse(ctx, id).RenameWarehouseRequest(*req).Execute()
	return entityResult("rename warehouse", out, resp, err)
}

// Activate transitions a warehouse from inactive to active.
func (w *Warehouses) Activate(ctx context.Context, id string) error {
	resp, err := w.api.ActivateWarehouse(ctx, id).Execute()
	return WrapAPIError("activate warehouse", err, resp)
}

// Deactivate transitions a warehouse from active to inactive. Inactive
// warehouses reject catalog operations until reactivated.
func (w *Warehouses) Deactivate(ctx context.Context, id string) error {
	resp, err := w.api.DeactivateWarehouse(ctx, id).Execute()
	return WrapAPIError("deactivate warehouse", err, resp)
}

// SetProtection toggles the protection flag on a warehouse. While protected,
// the warehouse cannot be deleted.
func (w *Warehouses) SetProtection(ctx context.Context, id string, req *managementv1.SetProtectionRequest) (*managementv1.ProtectionResponse, error) {
	if req == nil {
		return nil, errors.New("set warehouse protection: request must not be nil")
	}
	out, resp, err := w.api.SetWarehouseProtection(ctx, id).SetProtectionRequest(*req).Execute()
	return entityResult("set warehouse protection", out, resp, err)
}

// GetWarehouseStatisticsOptions holds the query parameters of
// Warehouses.Statistics. A nil pointer or a zero field sends nothing for that
// parameter.
type GetWarehouseStatisticsOptions struct {
	// PageSize caps the number of entries per page. Zero: the server's default.
	PageSize int64
	// PageToken resumes from the NextPageToken of a previous page.
	PageToken string
}

// Statistics returns one page of a warehouse's usage statistics (table and
// view counts per sample timestamp). Read the next page's token from the
// response's GetNextPageToken.
func (w *Warehouses) Statistics(ctx context.Context, id string, opts *GetWarehouseStatisticsOptions) (*managementv1.WarehouseStatisticsResponse, error) {
	req := w.api.GetWarehouseStatistics(ctx, id)
	if opts != nil {
		req = withPaging(req, opts.PageSize, opts.PageToken)
	}
	out, resp, err := req.Execute()
	return entityResult("get warehouse statistics", out, resp, err)
}

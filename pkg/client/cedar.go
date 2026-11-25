package client

import (
	"context"
	"errors"
	"fmt"

	managementv1 "github.com/lakekeeper/go-lakekeeper/pkg/apis/management/v1"
)

// cedarRequirement is what every Cedar operation needs: Lakekeeper+ running
// the Cedar authorization backend.
var cedarRequirement = requirement{
	feature:        "Cedar",
	lakekeeperPlus: true,
	authzBackend:   AuthzBackendCedar,
}

// Cedar is a one-call façade over PermissionsCedarAPIService: Cedar policy
// management on Lakekeeper+ with the Cedar authorization backend. On other
// servers its methods fail with an *UnsupportedError (see ErrUnsupported).
type Cedar struct {
	api  *managementv1.PermissionsCedarAPIService
	caps *capabilityCache
}

// Schema returns the active Cedar schema, as fragments keyed by file path.
func (c *Cedar) Schema(ctx context.Context) (*managementv1.GetCedarSchemaResponse, error) {
	const op = "get cedar schema"
	out, resp, err := c.api.GetCedarSchema(ctx).Execute()
	out, err = entityResult(op, out, resp, err)
	return out, c.caps.diagnose(ctx, cedarRequirement, op, err)
}

// ListProjectPolicies lists every Cedar policy stored for a project, with
// the scope's version and whether the caller can change them. Name the
// Project with WithProject, or omit it for the server's default project.
// Accepts WithBreakGlass: can-write then says whether emergency access could
// change them.
func (c *Cedar) ListProjectPolicies(ctx context.Context, callOpts ...CallOption) (*managementv1.ScopePoliciesResponse, error) {
	const op = "list project cedar policies"
	req, err := withProjectAndBreakGlass(op, c.api.ListProjectScopePolicies(ctx), callOpts)
	if err != nil {
		return nil, err
	}
	out, resp, err := req.Execute()
	out, err = entityResult(op, out, resp, err)
	return out, c.caps.diagnose(ctx, cedarRequirement, op, err)
}

// GetProjectPolicy returns one Cedar policy stored for a project, by name.
func (c *Cedar) GetProjectPolicy(ctx context.Context, name string, callOpts ...CallOption) (*managementv1.ScopePolicyEntry, error) {
	const op = "get project cedar policy"
	req, err := withProject(op, c.api.GetProjectScopePolicy(ctx, name), callOpts)
	if err != nil {
		return nil, err
	}
	out, resp, err := req.Execute()
	out, err = entityResult(op, out, resp, err)
	return out, c.caps.diagnose(ctx, cedarRequirement, op, err)
}

// ApplyProjectPolicies creates, updates and deletes a project's Cedar
// policies in one transaction, as req describes. Accepts WithProject and
// WithBreakGlass.
func (c *Cedar) ApplyProjectPolicies(ctx context.Context, req *managementv1.ApplyScopePoliciesRequest, callOpts ...CallOption) (*managementv1.ApplyScopePoliciesResponse, error) {
	const op = "apply project cedar policies"
	if req == nil {
		return nil, errors.New(op + ": request must not be nil")
	}
	call, err := withProjectAndBreakGlass(op, c.api.ApplyProjectScopePolicies(ctx), callOpts)
	if err != nil {
		return nil, err
	}
	out, resp, err := call.ApplyScopePoliciesRequest(*req).Execute()
	out, err = entityResult(op, out, resp, err)
	return out, c.caps.diagnose(ctx, cedarRequirement, op, err)
}

// ListWarehousePolicies lists every Cedar policy stored for a warehouse, with
// the scope's version, its project's scope version, and whether the caller
// can change them. Accepts WithBreakGlass.
func (c *Cedar) ListWarehousePolicies(ctx context.Context, warehouseID string, callOpts ...CallOption) (*managementv1.ScopePoliciesResponse, error) {
	const op = "list warehouse cedar policies"
	req, err := withBreakGlass(op, c.api.ListWarehouseScopePolicies(ctx, warehouseID), callOpts)
	if err != nil {
		return nil, err
	}
	out, resp, err := req.Execute()
	out, err = entityResult(op, out, resp, err)
	return out, c.caps.diagnose(ctx, cedarRequirement, op, err)
}

// GetWarehousePolicy returns one Cedar policy stored for a warehouse, by
// name.
func (c *Cedar) GetWarehousePolicy(ctx context.Context, warehouseID, name string) (*managementv1.ScopePolicyEntry, error) {
	const op = "get warehouse cedar policy"
	out, resp, err := c.api.GetWarehouseScopePolicy(ctx, warehouseID, name).Execute()
	out, err = entityResult(op, out, resp, err)
	return out, c.caps.diagnose(ctx, cedarRequirement, op, err)
}

// ApplyWarehousePolicies creates, updates and deletes a warehouse's Cedar
// policies in one transaction, as req describes. Accepts WithBreakGlass.
func (c *Cedar) ApplyWarehousePolicies(ctx context.Context, warehouseID string, req *managementv1.ApplyScopePoliciesRequest, callOpts ...CallOption) (*managementv1.ApplyScopePoliciesResponse, error) {
	const op = "apply warehouse cedar policies"
	if req == nil {
		return nil, errors.New(op + ": request must not be nil")
	}
	call, err := withBreakGlass(op, c.api.ApplyWarehouseScopePolicies(ctx, warehouseID), callOpts)
	if err != nil {
		return nil, err
	}
	out, resp, err := call.ApplyScopePoliciesRequest(*req).Execute()
	out, err = entityResult(op, out, resp, err)
	return out, c.caps.diagnose(ctx, cedarRequirement, op, err)
}

// BreakGlassStatus reports whether a break-glass request by the caller would
// be honored for a project. WithProject names the Project; the spec takes it
// as the project-id query parameter, so that is how it is sent. Without it,
// the server uses its default project.
func (c *Cedar) BreakGlassStatus(ctx context.Context, callOpts ...CallOption) (*managementv1.BreakGlassStatusResponse, error) {
	const op = "get break-glass status"
	o := resolveCallOptions(callOpts)
	req := c.api.BreakGlassStatus(ctx)
	switch {
	case o.breakGlassSet:
		return nil, notAccepted(op, "WithBreakGlass")
	case !o.projectSet:
	case o.projectID == "":
		return nil, fmt.Errorf("%s: %w", op, ErrEmptyProjectID)
	default:
		req = req.ProjectId(o.projectID)
	}
	out, resp, err := req.Execute()
	out, err = entityResult(op, out, resp, err)
	return out, c.caps.diagnose(ctx, cedarRequirement, op, err)
}

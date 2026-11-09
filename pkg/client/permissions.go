package client

import (
	"context"
	"fmt"

	managementv1 "github.com/lakekeeper/go-lakekeeper/pkg/apis/management/v1"
)

// Permissions is the façade over Lakekeeper's permissions service
// (PermissionsOpenfgaAPI). It holds one sub-façade per resource kind; each
// offers the same four verbs:
//
//   - Access reads a principal's allowed actions on the resource (the
//     current principal unless the options name another one);
//   - Assignments lists the resource's assignments;
//   - Grant writes assignments;
//   - Revoke deletes assignments.
//
// Assignments are the generated *Assignment types, e.g. what
// permissions.BuildAssignmentSet builds.
type Permissions struct {
	Warehouses *WarehousePermissions
	Roles      *RolePermissions
}

// errNoAssignments rejects a Grant or Revoke without assignments before it
// reaches the server, where it would be a silent no-op.
func errNoAssignments(op string) error {
	return fmt.Errorf("%s: no assignments", op)
}

// WarehousePermissions manages the permissions of a warehouse. The
// warehouse id selects the project, so these calls take no project option.
type WarehousePermissions struct {
	api *managementv1.PermissionsOpenfgaAPIService
}

// GetWarehouseAccessByIDOptions holds the query parameters of
// WarehousePermissions.Access. A nil pointer or a zero field sends nothing.
type GetWarehouseAccessByIDOptions struct {
	// PrincipalUser reads this user's access instead of the caller's.
	PrincipalUser string
	// PrincipalRole reads this role's access instead of the caller's.
	PrincipalRole string
}

// Access returns the allowed actions on a warehouse.
func (p *WarehousePermissions) Access(ctx context.Context, warehouseID string, opts *GetWarehouseAccessByIDOptions) (*managementv1.GetWarehouseAccessResponse, error) {
	req := p.api.GetWarehouseAccessById(ctx, warehouseID)
	if opts != nil {
		if opts.PrincipalUser != "" {
			req = req.PrincipalUser(opts.PrincipalUser)
		}
		if opts.PrincipalRole != "" {
			req = req.PrincipalRole(opts.PrincipalRole)
		}
	}
	out, resp, err := req.Execute()
	return out, WrapAPIError("get warehouse access", err, resp)
}

// GetWarehouseAssignmentsByIDOptions holds the query parameters of
// WarehousePermissions.Assignments. A nil pointer or a zero field sends
// nothing.
type GetWarehouseAssignmentsByIDOptions struct {
	// Relations keeps only assignments of these relations. Empty: all.
	Relations []managementv1.WarehouseRelation
}

// Assignments lists the assignments on a warehouse.
func (p *WarehousePermissions) Assignments(ctx context.Context, warehouseID string, opts *GetWarehouseAssignmentsByIDOptions) (*managementv1.GetWarehouseAssignmentsResponse, error) {
	req := p.api.GetWarehouseAssignmentsById(ctx, warehouseID)
	if opts != nil && len(opts.Relations) > 0 {
		req = req.Relations(opts.Relations)
	}
	out, resp, err := req.Execute()
	return out, WrapAPIError("get warehouse assignments", err, resp)
}

// Grant writes assignments on a warehouse.
func (p *WarehousePermissions) Grant(ctx context.Context, warehouseID string, assignments []managementv1.WarehouseAssignment) error {
	return p.update(ctx, "grant warehouse assignments", warehouseID, managementv1.UpdateWarehouseAssignmentsRequest{Writes: assignments})
}

// Revoke deletes assignments on a warehouse.
func (p *WarehousePermissions) Revoke(ctx context.Context, warehouseID string, assignments []managementv1.WarehouseAssignment) error {
	return p.update(ctx, "revoke warehouse assignments", warehouseID, managementv1.UpdateWarehouseAssignmentsRequest{Deletes: assignments})
}

func (p *WarehousePermissions) update(ctx context.Context, op, warehouseID string, req managementv1.UpdateWarehouseAssignmentsRequest) error {
	if len(req.Writes) == 0 && len(req.Deletes) == 0 {
		return errNoAssignments(op)
	}
	resp, err := p.api.UpdateWarehouseAssignmentsById(ctx, warehouseID).UpdateWarehouseAssignmentsRequest(req).Execute()
	return WrapAPIError(op, err, resp)
}

// RolePermissions manages the permissions of a role. The role id selects
// the project, so these calls take no project option.
type RolePermissions struct {
	api *managementv1.PermissionsOpenfgaAPIService
}

// GetAuthorizerRoleActionsOptions holds the query parameters of
// RolePermissions.Access. A nil pointer or a zero field sends nothing.
type GetAuthorizerRoleActionsOptions struct {
	// PrincipalUser reads this user's access instead of the caller's.
	PrincipalUser string
	// PrincipalRole reads this role's access instead of the caller's.
	PrincipalRole string
}

// Access returns the allowed authorizer actions on a role.
func (p *RolePermissions) Access(ctx context.Context, roleID string, opts *GetAuthorizerRoleActionsOptions) (*managementv1.GetOpenFGARoleActionsResponse, error) {
	req := p.api.GetAuthorizerRoleActions(ctx, roleID)
	if opts != nil {
		if opts.PrincipalUser != "" {
			req = req.PrincipalUser(opts.PrincipalUser)
		}
		if opts.PrincipalRole != "" {
			req = req.PrincipalRole(opts.PrincipalRole)
		}
	}
	out, resp, err := req.Execute()
	return out, WrapAPIError("get role access", err, resp)
}

// GetRoleAssignmentsByIDOptions holds the query parameters of
// RolePermissions.Assignments. A nil pointer or a zero field sends nothing.
type GetRoleAssignmentsByIDOptions struct {
	// Relations keeps only assignments of these relations. Empty: all.
	Relations []managementv1.RoleRelation
}

// Assignments lists the assignments on a role.
func (p *RolePermissions) Assignments(ctx context.Context, roleID string, opts *GetRoleAssignmentsByIDOptions) (*managementv1.GetRoleAssignmentsResponse, error) {
	req := p.api.GetRoleAssignmentsById(ctx, roleID)
	if opts != nil && len(opts.Relations) > 0 {
		req = req.Relations(opts.Relations)
	}
	out, resp, err := req.Execute()
	return out, WrapAPIError("get role assignments", err, resp)
}

// Grant writes assignments on a role.
func (p *RolePermissions) Grant(ctx context.Context, roleID string, assignments []managementv1.RoleAssignment) error {
	return p.update(ctx, "grant role assignments", roleID, managementv1.UpdateRoleAssignmentsRequest{Writes: assignments})
}

// Revoke deletes assignments on a role.
func (p *RolePermissions) Revoke(ctx context.Context, roleID string, assignments []managementv1.RoleAssignment) error {
	return p.update(ctx, "revoke role assignments", roleID, managementv1.UpdateRoleAssignmentsRequest{Deletes: assignments})
}

func (p *RolePermissions) update(ctx context.Context, op, roleID string, req managementv1.UpdateRoleAssignmentsRequest) error {
	if len(req.Writes) == 0 && len(req.Deletes) == 0 {
		return errNoAssignments(op)
	}
	resp, err := p.api.UpdateRoleAssignmentsById(ctx, roleID).UpdateRoleAssignmentsRequest(req).Execute()
	return WrapAPIError(op, err, resp)
}

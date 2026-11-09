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
	Projects   *ProjectPermissions
	Server     *ServerPermissions
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

// ProjectPermissions manages the permissions of a project. The project is
// the WithProject option; without it the server applies its default project.
type ProjectPermissions struct {
	api *managementv1.PermissionsOpenfgaAPIService
}

// GetProjectAccessOptions holds the query parameters of
// ProjectPermissions.Access. A nil pointer or a zero field sends nothing.
type GetProjectAccessOptions struct {
	// PrincipalUser reads this user's access instead of the caller's.
	PrincipalUser string
	// PrincipalRole reads this role's access instead of the caller's.
	PrincipalRole string
}

// Access returns the allowed actions on a project.
func (p *ProjectPermissions) Access(ctx context.Context, opts *GetProjectAccessOptions, callOpts ...CallOption) (*managementv1.GetProjectAccessResponse, error) {
	req := withProject(p.api.GetProjectAccess(ctx), callOpts)
	if opts != nil {
		if opts.PrincipalUser != "" {
			req = req.PrincipalUser(opts.PrincipalUser)
		}
		if opts.PrincipalRole != "" {
			req = req.PrincipalRole(opts.PrincipalRole)
		}
	}
	out, resp, err := req.Execute()
	return out, WrapAPIError("get project access", err, resp)
}

// GetProjectAssignmentsOptions holds the query parameters of
// ProjectPermissions.Assignments. A nil pointer or a zero field sends
// nothing.
type GetProjectAssignmentsOptions struct {
	// Relations keeps only assignments of these relations. Empty: all.
	Relations []managementv1.ProjectRelation
}

// Assignments lists the assignments on a project.
func (p *ProjectPermissions) Assignments(ctx context.Context, opts *GetProjectAssignmentsOptions, callOpts ...CallOption) (*managementv1.GetProjectAssignmentsResponse, error) {
	req := withProject(p.api.GetProjectAssignments(ctx), callOpts)
	if opts != nil && len(opts.Relations) > 0 {
		req = req.Relations(opts.Relations)
	}
	out, resp, err := req.Execute()
	return out, WrapAPIError("get project assignments", err, resp)
}

// Grant writes assignments on a project.
func (p *ProjectPermissions) Grant(ctx context.Context, assignments []managementv1.ProjectAssignment, callOpts ...CallOption) error {
	return p.update(ctx, "grant project assignments", managementv1.UpdateProjectAssignmentsRequest{Writes: assignments}, callOpts)
}

// Revoke deletes assignments on a project.
func (p *ProjectPermissions) Revoke(ctx context.Context, assignments []managementv1.ProjectAssignment, callOpts ...CallOption) error {
	return p.update(ctx, "revoke project assignments", managementv1.UpdateProjectAssignmentsRequest{Deletes: assignments}, callOpts)
}

func (p *ProjectPermissions) update(ctx context.Context, op string, req managementv1.UpdateProjectAssignmentsRequest, callOpts []CallOption) error {
	if len(req.Writes) == 0 && len(req.Deletes) == 0 {
		return errNoAssignments(op)
	}
	resp, err := withProject(p.api.UpdateProjectAssignments(ctx), callOpts).UpdateProjectAssignmentsRequest(req).Execute()
	return WrapAPIError(op, err, resp)
}

// ServerPermissions manages the permissions of the server. The server is
// a single resource, so these calls take neither an id nor a project
// option.
type ServerPermissions struct {
	api *managementv1.PermissionsOpenfgaAPIService
}

// GetServerAccessOptions holds the query parameters of
// ServerPermissions.Access. A nil pointer or a zero field sends nothing.
type GetServerAccessOptions struct {
	// PrincipalUser reads this user's access instead of the caller's.
	PrincipalUser string
	// PrincipalRole reads this role's access instead of the caller's.
	PrincipalRole string
}

// Access returns the allowed actions on the server.
func (p *ServerPermissions) Access(ctx context.Context, opts *GetServerAccessOptions) (*managementv1.GetServerAccessResponse, error) {
	req := p.api.GetServerAccess(ctx)
	if opts != nil {
		if opts.PrincipalUser != "" {
			req = req.PrincipalUser(opts.PrincipalUser)
		}
		if opts.PrincipalRole != "" {
			req = req.PrincipalRole(opts.PrincipalRole)
		}
	}
	out, resp, err := req.Execute()
	return out, WrapAPIError("get server access", err, resp)
}

// GetServerAssignmentsOptions holds the query parameters of
// ServerPermissions.Assignments. A nil pointer or a zero field sends
// nothing.
type GetServerAssignmentsOptions struct {
	// Relations keeps only assignments of these relations. Empty: all.
	Relations []managementv1.ServerRelation
}

// Assignments lists the assignments on the server.
func (p *ServerPermissions) Assignments(ctx context.Context, opts *GetServerAssignmentsOptions) (*managementv1.GetServerAssignmentsResponse, error) {
	req := p.api.GetServerAssignments(ctx)
	if opts != nil && len(opts.Relations) > 0 {
		req = req.Relations(opts.Relations)
	}
	out, resp, err := req.Execute()
	return out, WrapAPIError("get server assignments", err, resp)
}

// Grant writes assignments on the server.
func (p *ServerPermissions) Grant(ctx context.Context, assignments []managementv1.ServerAssignment) error {
	return p.update(ctx, "grant server assignments", managementv1.UpdateServerAssignmentsRequest{Writes: assignments})
}

// Revoke deletes assignments on the server.
func (p *ServerPermissions) Revoke(ctx context.Context, assignments []managementv1.ServerAssignment) error {
	return p.update(ctx, "revoke server assignments", managementv1.UpdateServerAssignmentsRequest{Deletes: assignments})
}

func (p *ServerPermissions) update(ctx context.Context, op string, req managementv1.UpdateServerAssignmentsRequest) error {
	if len(req.Writes) == 0 && len(req.Deletes) == 0 {
		return errNoAssignments(op)
	}
	resp, err := p.api.UpdateServerAssignments(ctx).UpdateServerAssignmentsRequest(req).Execute()
	return WrapAPIError(op, err, resp)
}

package client

import (
	"context"
	"errors"

	managementv1 "github.com/lakekeeper/go-lakekeeper/pkg/apis/management/v1"
)

// Roles is a one-call façade over RoleAPIService. All operations require
// a project id, threaded through as the X-Project-Id header. For request
// options (paging, custom headers), reach for c.RoleAPI.
type Roles struct {
	api *managementv1.RoleAPIService
}

// Create creates a role in the given project.
func (r *Roles) Create(ctx context.Context, projectID string, req *managementv1.CreateRoleRequest) (*managementv1.Role, error) {
	if req == nil {
		return nil, errors.New("create role: request must not be nil")
	}
	out, _, err := r.api.CreateRole(ctx).XProjectId(projectID).CreateRoleRequest(*req).Execute()
	return out, err
}

// Get retrieves a role by id, scoped to the given project.
func (r *Roles) Get(ctx context.Context, projectID, roleID string) (*managementv1.Role, error) {
	out, _, err := r.api.GetRole(ctx, roleID).XProjectId(projectID).Execute()
	return out, err
}

// Delete removes a role.
func (r *Roles) Delete(ctx context.Context, projectID, roleID string) error {
	_, err := r.api.DeleteRole(ctx, roleID).XProjectId(projectID).Execute()
	return err
}

// List returns roles in the given project.
func (r *Roles) List(ctx context.Context, projectID string) (*managementv1.ListRolesResponse, error) {
	out, _, err := r.api.ListRoles(ctx).XProjectId(projectID).Execute()
	return out, err
}

// Update mutates a role's name/description.
func (r *Roles) Update(ctx context.Context, projectID, roleID string, req *managementv1.UpdateRoleRequest) (*managementv1.Role, error) {
	if req == nil {
		return nil, errors.New("update role: request must not be nil")
	}
	out, _, err := r.api.UpdateRole(ctx, roleID).XProjectId(projectID).UpdateRoleRequest(*req).Execute()
	return out, err
}

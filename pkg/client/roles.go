package client

import (
	"context"
	"errors"

	managementv1 "github.com/lakekeeper/go-lakekeeper/pkg/apis/management/v1"
)

// Roles is a one-call façade over RoleAPIService. Every operation is
// project-scoped: name the Project with WithProject, or omit it and the server
// applies its default project. Role permissions live on c.Permissions.Roles.
// Role membership has no façade; reach for the embedded c.RoleAPI.
type Roles struct {
	api *managementv1.RoleAPIService
}

// Create creates a role.
func (r *Roles) Create(ctx context.Context, req *managementv1.CreateRoleRequest, callOpts ...CallOption) (*managementv1.Role, error) {
	if req == nil {
		return nil, errors.New("create role: request must not be nil")
	}
	out, resp, err := withProject(r.api.CreateRole(ctx), callOpts).CreateRoleRequest(*req).Execute()
	return out, WrapAPIError("create role", err, resp)
}

// Get retrieves a role by id.
func (r *Roles) Get(ctx context.Context, roleID string, callOpts ...CallOption) (*managementv1.Role, error) {
	out, resp, err := withProject(r.api.GetRole(ctx, roleID), callOpts).Execute()
	return out, WrapAPIError("get role", err, resp)
}

// Delete removes a role.
func (r *Roles) Delete(ctx context.Context, roleID string, callOpts ...CallOption) error {
	resp, err := withProject(r.api.DeleteRole(ctx, roleID), callOpts).Execute()
	return WrapAPIError("delete role", err, resp)
}

// ListRolesOptions holds the query parameters of Roles.List. A nil pointer
// or a zero field sends nothing.
type ListRolesOptions struct {
	// PageSize caps the number of roles in one page; zero uses the server's
	// default page size.
	PageSize int64
	// PageToken continues a listing from a previous page's NextPageToken.
	PageToken string
}

// List returns one page of roles. Page through the project with the
// response's NextPageToken.
func (r *Roles) List(ctx context.Context, opts *ListRolesOptions, callOpts ...CallOption) (*managementv1.ListRolesResponse, error) {
	req := withProject(r.api.ListRoles(ctx), callOpts)
	if opts != nil {
		if opts.PageSize != 0 {
			req = req.PageSize(opts.PageSize)
		}
		if opts.PageToken != "" {
			req = req.PageToken(opts.PageToken)
		}
	}
	out, resp, err := req.Execute()
	return out, WrapAPIError("list roles", err, resp)
}

// Update mutates a role's name/description.
func (r *Roles) Update(ctx context.Context, roleID string, req *managementv1.UpdateRoleRequest, callOpts ...CallOption) (*managementv1.Role, error) {
	if req == nil {
		return nil, errors.New("update role: request must not be nil")
	}
	out, resp, err := withProject(r.api.UpdateRole(ctx, roleID), callOpts).UpdateRoleRequest(*req).Execute()
	return out, WrapAPIError("update role", err, resp)
}

// Search runs the server's fuzzy role search over req.Search and returns the
// best matches. The server truncates the search string to 64 characters and
// returns a fixed number of roles (currently 10), ranked by
// trigram distance; a role with exactly the searched name is usually, but not
// always, among them. To prove a name is absent, page through List.
func (r *Roles) Search(ctx context.Context, req *managementv1.SearchRoleRequest, callOpts ...CallOption) (*managementv1.SearchRoleResponse, error) {
	if req == nil {
		return nil, errors.New("search roles: request must not be nil")
	}
	out, resp, err := withProject(r.api.SearchRole(ctx), callOpts).SearchRoleRequest(*req).Execute()
	return out, WrapAPIError("search roles", err, resp)
}

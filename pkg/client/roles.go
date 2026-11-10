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

// Create creates a role. A project in req's deprecated ProjectId field is
// sent as the header, like WithProject; naming a different project in both
// fails with ErrProjectIDConflict. req itself is not modified.
func (r *Roles) Create(ctx context.Context, req *managementv1.CreateRoleRequest, callOpts ...CallOption) (*managementv1.Role, error) {
	const op = "create role"
	if req == nil {
		return nil, errors.New(op + ": request must not be nil")
	}
	body := *req
	callOpts, err := projectFromBody(op, body.GetProjectId(), callOpts)
	if err != nil {
		return nil, err
	}
	body.UnsetProjectId()
	call, err := withProject(op, r.api.CreateRole(ctx), callOpts)
	if err != nil {
		return nil, err
	}
	out, resp, err := call.CreateRoleRequest(body).Execute()
	return entityResult(op, out, resp, err)
}

// Get retrieves a role by id.
func (r *Roles) Get(ctx context.Context, roleID string, callOpts ...CallOption) (*managementv1.Role, error) {
	req, err := withProject("get role", r.api.GetRole(ctx, roleID), callOpts)
	if err != nil {
		return nil, err
	}
	out, resp, err := req.Execute()
	return entityResult("get role", out, resp, err)
}

// Delete removes a role.
func (r *Roles) Delete(ctx context.Context, roleID string, callOpts ...CallOption) error {
	req, err := withProject("delete role", r.api.DeleteRole(ctx, roleID), callOpts)
	if err != nil {
		return err
	}
	resp, err := req.Execute()
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
	req, err := withProject("list roles", r.api.ListRoles(ctx), callOpts)
	if err != nil {
		return nil, err
	}
	if opts != nil {
		req = withPaging(req, opts.PageSize, opts.PageToken)
	}
	out, resp, err := req.Execute()
	return entityResult("list roles", out, resp, err)
}

// Update mutates a role's name/description.
func (r *Roles) Update(ctx context.Context, roleID string, req *managementv1.UpdateRoleRequest, callOpts ...CallOption) (*managementv1.Role, error) {
	if req == nil {
		return nil, errors.New("update role: request must not be nil")
	}
	call, err := withProject("update role", r.api.UpdateRole(ctx, roleID), callOpts)
	if err != nil {
		return nil, err
	}
	out, resp, err := call.UpdateRoleRequest(*req).Execute()
	return entityResult("update role", out, resp, err)
}

// Search runs the server's fuzzy role search over req.Search and returns the
// best matches. The server truncates the search string to 64 characters and
// returns a fixed number of roles (currently 10), ranked by
// trigram distance; a role with exactly the searched name is usually, but not
// always, among them. To prove a name is absent, page through List.
//
// A project in req's deprecated ProjectId field is handled as in Create.
func (r *Roles) Search(ctx context.Context, req *managementv1.SearchRoleRequest, callOpts ...CallOption) (*managementv1.SearchRoleResponse, error) {
	const op = "search roles"
	if req == nil {
		return nil, errors.New(op + ": request must not be nil")
	}
	body := *req
	callOpts, err := projectFromBody(op, body.GetProjectId(), callOpts)
	if err != nil {
		return nil, err
	}
	body.UnsetProjectId()
	call, err := withProject(op, r.api.SearchRole(ctx), callOpts)
	if err != nil {
		return nil, err
	}
	out, resp, err := call.SearchRoleRequest(body).Execute()
	return entityResult(op, out, resp, err)
}

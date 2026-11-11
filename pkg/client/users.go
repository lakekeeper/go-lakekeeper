package client

import (
	"context"
	"errors"

	managementv1 "github.com/lakekeeper/go-lakekeeper/pkg/apis/management/v1"
)

// Users is a one-call façade over UserAPIService. Users are not
// project-scoped, but Roles and TransitiveRoles list role assignments, which
// are: they take WithProject.
type Users struct {
	api *managementv1.UserAPIService
}

// Create provisions a user.
func (u *Users) Create(ctx context.Context, req *managementv1.CreateUserRequest) (*managementv1.User, error) {
	const op = "create user"
	if req == nil {
		return nil, errors.New(op + ": request must not be nil")
	}
	out, resp, err := u.api.CreateUser(ctx).CreateUserRequest(*req).Execute()
	return entityResult(op, out, resp, err)
}

// Get fetches a user by id.
func (u *Users) Get(ctx context.Context, userID string) (*managementv1.User, error) {
	out, resp, err := u.api.GetUser(ctx, userID).Execute()
	return entityResult("get user", out, resp, err)
}

// Delete removes a user.
func (u *Users) Delete(ctx context.Context, userID string) error {
	resp, err := u.api.DeleteUser(ctx, userID).Execute()
	return WrapAPIError("delete user", err, resp)
}

// ListUserOptions holds the query parameters of Users.List. A zero field
// sends nothing, so the server's default applies.
type ListUserOptions struct {
	// Name filters users by name on the server (across all pages).
	Name string
	// PageSize caps the number of users in one response.
	PageSize int64
	// PageToken continues a previous listing; take it from
	// ListUsersResponse.GetNextPageToken().
	PageToken string
}

// List returns one page of users. opts may be nil.
func (u *Users) List(ctx context.Context, opts *ListUserOptions) (*managementv1.ListUsersResponse, error) {
	req := u.api.ListUser(ctx)
	if opts != nil {
		if opts.Name != "" {
			req = req.Name(opts.Name)
		}
		req = withPaging(req, opts.PageSize, opts.PageToken)
	}
	out, resp, err := req.Execute()
	return entityResult("list users", out, resp, err)
}

// Update mutates a user's profile fields.
func (u *Users) Update(ctx context.Context, userID string, req *managementv1.UpdateUserRequest) error {
	const op = "update user"
	if req == nil {
		return errors.New(op + ": request must not be nil")
	}
	resp, err := u.api.UpdateUser(ctx, userID).UpdateUserRequest(*req).Execute()
	return WrapAPIError(op, err, resp)
}

// Whoami returns the calling principal's user record.
func (u *Users) Whoami(ctx context.Context) (*managementv1.WhoamiResponse, error) {
	out, resp, err := u.api.Whoami(ctx).Execute()
	return entityResult("whoami", out, resp, err)
}

// Search runs the server's fuzzy user search over req.Search and returns the
// best matches. The server truncates the search string to 64 characters and
// returns a fixed number of users, ranked by similarity; a user with exactly
// the searched name is usually, but not always, among them. To prove a name
// is absent, page through List.
func (u *Users) Search(ctx context.Context, req *managementv1.SearchUserRequest) (*managementv1.SearchUserResponse, error) {
	const op = "search users"
	if req == nil {
		return nil, errors.New(op + ": request must not be nil")
	}
	out, resp, err := u.api.SearchUser(ctx).SearchUserRequest(*req).Execute()
	return entityResult(op, out, resp, err)
}

// GetUserActionsOptions holds the query parameters of Users.Actions. A nil
// pointer or an empty field sends nothing.
type GetUserActionsOptions struct {
	// PrincipalUser reads this user's actions instead of the caller's.
	PrincipalUser string
	// PrincipalRole reads this role's actions instead of the caller's.
	PrincipalRole string
}

// Actions returns the actions a principal may perform on the user userID:
// the caller unless opts names another principal. opts may be nil.
func (u *Users) Actions(ctx context.Context, userID string, opts *GetUserActionsOptions) (*managementv1.GetLakekeeperUserActionsResponse, error) {
	out, resp, err := withPrincipal(u.api.GetUserActions(ctx, userID), opts).Execute()
	return entityResult("get user actions", out, resp, err)
}

// ListUserRolesOptions holds the query parameters of Users.Roles and
// Users.TransitiveRoles. A nil pointer or a zero field sends nothing.
type ListUserRolesOptions struct {
	// PageSize caps the number of roles in one page; zero uses the server's
	// default page size.
	PageSize int64
	// PageToken continues a listing from a previous page's NextPageToken.
	PageToken string
}

// Roles returns one page of the roles the user is directly assigned to, in
// the WithProject project (else the server's default project). Page through
// them with the response's NextPageToken.
func (u *Users) Roles(ctx context.Context, userID string, opts *ListUserRolesOptions, callOpts ...CallOption) (*managementv1.ListRoleMembershipsResponse, error) {
	const op = "list user roles"
	req, err := withProject(op, u.api.ListUserRoles(ctx, userID), callOpts)
	if err != nil {
		return nil, err
	}
	if opts != nil {
		req = withPaging(req, opts.PageSize, opts.PageToken)
	}
	out, resp, err := req.Execute()
	return entityResult(op, out, resp, err)
}

// TransitiveRoles is Roles over the user's effective role set: the direct
// assignments plus every role reachable upward through role membership. The
// server supports it only when assignments are catalog-managed; with an
// assignment-managing authorizer such as OpenFGA it answers 501 Not
// Implemented.
func (u *Users) TransitiveRoles(ctx context.Context, userID string, opts *ListUserRolesOptions, callOpts ...CallOption) (*managementv1.ListRoleMembershipsResponse, error) {
	const op = "list user transitive roles"
	req, err := withProject(op, u.api.ListUserTransitiveRoles(ctx, userID), callOpts)
	if err != nil {
		return nil, err
	}
	if opts != nil {
		req = withPaging(req, opts.PageSize, opts.PageToken)
	}
	out, resp, err := req.Execute()
	return entityResult(op, out, resp, err)
}

package client

import (
	"context"
	"errors"

	managementv1 "github.com/lakekeeper/go-lakekeeper/pkg/apis/management/v1"
)

// Users is a one-call façade over UserAPIService.
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

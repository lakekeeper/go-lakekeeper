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
	if req == nil {
		return nil, errors.New("create user: request must not be nil")
	}
	out, resp, err := u.api.CreateUser(ctx).CreateUserRequest(*req).Execute()
	return out, WrapAPIError("create user", err, resp)
}

// Get fetches a user by id.
func (u *Users) Get(ctx context.Context, userID string) (*managementv1.User, error) {
	out, resp, err := u.api.GetUser(ctx, userID).Execute()
	return out, WrapAPIError("get user", err, resp)
}

// Delete removes a user.
func (u *Users) Delete(ctx context.Context, userID string) error {
	resp, err := u.api.DeleteUser(ctx, userID).Execute()
	return WrapAPIError("delete user", err, resp)
}

// List returns users with optional paging. Use c.UserAPI.ListUser directly
// for filter parameters not exposed here.
func (u *Users) List(ctx context.Context) (*managementv1.ListUsersResponse, error) {
	out, resp, err := u.api.ListUser(ctx).Execute()
	return out, WrapAPIError("list users", err, resp)
}

// Update mutates a user's profile fields.
func (u *Users) Update(ctx context.Context, userID string, req *managementv1.UpdateUserRequest) error {
	if req == nil {
		return errors.New("update user: request must not be nil")
	}
	resp, err := u.api.UpdateUser(ctx, userID).UpdateUserRequest(*req).Execute()
	return WrapAPIError("update user", err, resp)
}

// Whoami returns the calling principal's user record.
func (u *Users) Whoami(ctx context.Context) (*managementv1.WhoamiResponse, error) {
	out, resp, err := u.api.Whoami(ctx).Execute()
	return out, WrapAPIError("whoami", err, resp)
}

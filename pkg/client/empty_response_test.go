package client_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	managementv1 "github.com/lakekeeper/go-lakekeeper/pkg/apis/management/v1"
	"github.com/lakekeeper/go-lakekeeper/pkg/client"
	"github.com/lakekeeper/go-lakekeeper/pkg/storage/profile"
	"github.com/lakekeeper/go-lakekeeper/pkg/testutil"
)

// A façade method that returns an entity never returns (nil, nil): the
// generated client decodes an empty 2xx body to a nil result without an
// error, and callers dereferencing it would panic.
func TestEntityMethods_EmptySuccessBodyIsAnError(t *testing.T) {
	t.Parallel()

	calls := map[string]func(context.Context, *client.Client) (any, error){
		"Warehouses.Create": func(ctx context.Context, c *client.Client) (any, error) {
			return c.Warehouses.Create(ctx, newCreateWarehouseRequest())
		},
		"Warehouses.Get": func(ctx context.Context, c *client.Client) (any, error) {
			return c.Warehouses.Get(ctx, warehouseID)
		},
		"Warehouses.List": func(ctx context.Context, c *client.Client) (any, error) {
			return c.Warehouses.List(ctx, nil)
		},
		"Warehouses.Rename": func(ctx context.Context, c *client.Client) (any, error) {
			return c.Warehouses.Rename(ctx, warehouseID, managementv1.NewRenameWarehouseRequest("x"))
		},
		"Warehouses.SetProtection": func(ctx context.Context, c *client.Client) (any, error) {
			return c.Warehouses.SetProtection(ctx, warehouseID, managementv1.NewSetProtectionRequest(true))
		},
		"Warehouses.Statistics": func(ctx context.Context, c *client.Client) (any, error) {
			return c.Warehouses.Statistics(ctx, warehouseID, nil)
		},
		"Warehouses.UpdateDeleteProfile": func(ctx context.Context, c *client.Client) (any, error) {
			return c.Warehouses.UpdateDeleteProfile(ctx, warehouseID,
				managementv1.NewUpdateWarehouseDeleteProfileRequest(profile.NewHardDeleteProfile()))
		},
		"Projects.Create": func(ctx context.Context, c *client.Client) (any, error) {
			return c.Projects.Create(ctx, managementv1.NewCreateProjectRequest("p"))
		},
		"Projects.Get": func(ctx context.Context, c *client.Client) (any, error) {
			return c.Projects.Get(ctx)
		},
		"Projects.List": func(ctx context.Context, c *client.Client) (any, error) {
			return c.Projects.List(ctx)
		},
		"Roles.Create": func(ctx context.Context, c *client.Client) (any, error) {
			return c.Roles.Create(ctx, managementv1.NewCreateRoleRequest("r"))
		},
		"Roles.Get": func(ctx context.Context, c *client.Client) (any, error) {
			return c.Roles.Get(ctx, roleID)
		},
		"Roles.List": func(ctx context.Context, c *client.Client) (any, error) {
			return c.Roles.List(ctx, nil)
		},
		"Roles.Update": func(ctx context.Context, c *client.Client) (any, error) {
			return c.Roles.Update(ctx, roleID, managementv1.NewUpdateRoleRequest("r"))
		},
		"Roles.Search": func(ctx context.Context, c *client.Client) (any, error) {
			return c.Roles.Search(ctx, managementv1.NewSearchRoleRequest("r"))
		},
		"Users.Create": func(ctx context.Context, c *client.Client) (any, error) {
			return c.Users.Create(ctx, managementv1.NewCreateUserRequestWithDefaults())
		},
		"Users.Get": func(ctx context.Context, c *client.Client) (any, error) {
			return c.Users.Get(ctx, "oidc~alice")
		},
		"Users.List": func(ctx context.Context, c *client.Client) (any, error) {
			return c.Users.List(ctx, nil)
		},
		"Users.Whoami": func(ctx context.Context, c *client.Client) (any, error) {
			return c.Users.Whoami(ctx)
		},
		"Server.Info": func(ctx context.Context, c *client.Client) (any, error) {
			return c.Server.Info(ctx)
		},
		"Permissions.Warehouses.Access": func(ctx context.Context, c *client.Client) (any, error) {
			return c.Permissions.Warehouses.Access(ctx, warehouseID, nil)
		},
		"Permissions.Warehouses.Assignments": func(ctx context.Context, c *client.Client) (any, error) {
			return c.Permissions.Warehouses.Assignments(ctx, warehouseID, nil)
		},
		"Permissions.Roles.Access": func(ctx context.Context, c *client.Client) (any, error) {
			return c.Permissions.Roles.Access(ctx, roleID, nil)
		},
		"Permissions.Roles.Assignments": func(ctx context.Context, c *client.Client) (any, error) {
			return c.Permissions.Roles.Assignments(ctx, roleID, nil)
		},
		"Permissions.Projects.Access": func(ctx context.Context, c *client.Client) (any, error) {
			return c.Permissions.Projects.Access(ctx, nil)
		},
		"Permissions.Projects.Assignments": func(ctx context.Context, c *client.Client) (any, error) {
			return c.Permissions.Projects.Assignments(ctx, nil)
		},
		"Permissions.Server.Access": func(ctx context.Context, c *client.Client) (any, error) {
			return c.Permissions.Server.Access(ctx, nil)
		},
		"Permissions.Server.Assignments": func(ctx context.Context, c *client.Client) (any, error) {
			return c.Permissions.Server.Assignments(ctx, nil)
		},
	}

	for name, call := range calls {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			mux, c := testutil.ServerMux(t)
			reqs := testutil.Respond(t, mux, "/", http.StatusOK, nil)

			_, err := call(t.Context(), c)

			require.ErrorIs(t, err, client.ErrEmptyResponse)
			assert.Equal(t, 1, reqs.Len(), "the request must have been sent")
		})
	}
}

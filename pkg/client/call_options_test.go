package client_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	managementv1 "github.com/lakekeeper/go-lakekeeper/pkg/apis/management/v1"
	"github.com/lakekeeper/go-lakekeeper/pkg/client"
	"github.com/lakekeeper/go-lakekeeper/pkg/permissions"
	"github.com/lakekeeper/go-lakekeeper/pkg/testutil"
)

// An explicit WithProject("") is a caller bug, typically an unset variable.
// Sending nothing would silently target the server's default project, which
// for Projects.Delete deletes it. Every project-scoped method must reject it
// before any request leaves.
func TestWithProject_EmptyIDIsRejectedBeforeAnyRequest(t *testing.T) {
	t.Parallel()
	empty := client.WithProject("")
	assignments, err := permissions.BuildAssignmentSet[managementv1.ProjectAssignment](
		[]string{"select"}, permissions.PrincipalSet{Users: []string{"oidc~alice"}})
	require.NoError(t, err)

	calls := map[string]func(context.Context, *client.Client) error{
		"Warehouses.Create": func(ctx context.Context, c *client.Client) error {
			_, err := c.Warehouses.Create(ctx, managementv1.NewCreateWarehouseRequest(managementv1.StorageProfile{}, "wh"), empty)
			return err
		},
		"Warehouses.List": func(ctx context.Context, c *client.Client) error {
			_, err := c.Warehouses.List(ctx, nil, empty)
			return err
		},
		"Projects.Get": func(ctx context.Context, c *client.Client) error {
			_, err := c.Projects.Get(ctx, empty)
			return err
		},
		"Projects.Rename": func(ctx context.Context, c *client.Client) error {
			return c.Projects.Rename(ctx, managementv1.NewRenameProjectRequest("x"), empty)
		},
		"Projects.Delete": func(ctx context.Context, c *client.Client) error {
			return c.Projects.Delete(ctx, empty)
		},
		"Roles.Create": func(ctx context.Context, c *client.Client) error {
			_, err := c.Roles.Create(ctx, managementv1.NewCreateRoleRequest("r"), empty)
			return err
		},
		"Roles.Get": func(ctx context.Context, c *client.Client) error {
			_, err := c.Roles.Get(ctx, "id", empty)
			return err
		},
		"Roles.List": func(ctx context.Context, c *client.Client) error {
			_, err := c.Roles.List(ctx, nil, empty)
			return err
		},
		"Roles.Update": func(ctx context.Context, c *client.Client) error {
			_, err := c.Roles.Update(ctx, "id", managementv1.NewUpdateRoleRequest("r"), empty)
			return err
		},
		"Roles.Delete": func(ctx context.Context, c *client.Client) error {
			return c.Roles.Delete(ctx, "id", empty)
		},
		"Roles.Search": func(ctx context.Context, c *client.Client) error {
			_, err := c.Roles.Search(ctx, managementv1.NewSearchRoleRequest("r"), empty)
			return err
		},
		"Users.Roles": func(ctx context.Context, c *client.Client) error {
			_, err := c.Users.Roles(ctx, "id", nil, empty)
			return err
		},
		"Users.TransitiveRoles": func(ctx context.Context, c *client.Client) error {
			_, err := c.Users.TransitiveRoles(ctx, "id", nil, empty)
			return err
		},
		"Permissions.Projects.Access": func(ctx context.Context, c *client.Client) error {
			_, err := c.Permissions.Projects.Access(ctx, nil, empty)
			return err
		},
		"Permissions.Projects.Assignments": func(ctx context.Context, c *client.Client) error {
			_, err := c.Permissions.Projects.Assignments(ctx, nil, empty)
			return err
		},
		"Permissions.Projects.Grant": func(ctx context.Context, c *client.Client) error {
			return c.Permissions.Projects.Grant(ctx, assignments, empty)
		},
		"Permissions.Projects.Revoke": func(ctx context.Context, c *client.Client) error {
			return c.Permissions.Projects.Revoke(ctx, assignments, empty)
		},
	}

	for name, call := range calls {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			mux, c := testutil.ServerMux(t)
			reqs := testutil.Respond(t, mux, "/", http.StatusOK, `{}`)

			err := call(t.Context(), c)

			require.ErrorIs(t, err, client.ErrEmptyProjectID)
			assert.Zero(t, reqs.Len(), "no request must be sent")
		})
	}
}

// Destructive project-level calls never fall back to the server's default
// project: without WithProject they would delete it, or change its
// permissions. They require the option, unlike every other project-scoped
// method, and send nothing without it.
func TestProjectRequiredMethods_WithoutProjectAreRejected(t *testing.T) {
	t.Parallel()
	assignments, err := permissions.BuildAssignmentSet[managementv1.ProjectAssignment](
		[]string{"select"}, permissions.PrincipalSet{Users: []string{"oidc~alice"}})
	require.NoError(t, err)

	calls := map[string]func(context.Context, *client.Client) error{
		"Projects.Delete": func(ctx context.Context, c *client.Client) error {
			return c.Projects.Delete(ctx)
		},
		"Permissions.Projects.Grant": func(ctx context.Context, c *client.Client) error {
			return c.Permissions.Projects.Grant(ctx, assignments)
		},
		"Permissions.Projects.Revoke": func(ctx context.Context, c *client.Client) error {
			return c.Permissions.Projects.Revoke(ctx, assignments)
		},
	}

	for name, call := range calls {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			mux, c := testutil.ServerMux(t)
			reqs := testutil.Respond(t, mux, "/", http.StatusNoContent, nil)

			err := call(t.Context(), c)

			require.ErrorIs(t, err, client.ErrProjectRequired)
			assert.Zero(t, reqs.Len(), "no request must be sent")
		})
	}
}

// bodyProjectCall is a façade method whose request model still has the
// deprecated project-id body field. The façade must never send that field:
// it moves the project to the x-project-id header instead.
type bodyProjectCall struct {
	pattern  string
	response string
	// call sends a request whose body field is bodyProject and reports what
	// the caller's request holds afterwards.
	call func(ctx context.Context, c *client.Client, bodyProject string, opts ...client.CallOption) (callerBodyProject string, err error)
}

var bodyProjectCalls = map[string]bodyProjectCall{
	"Warehouses.Create": {
		pattern:  "POST /management/v1/warehouse",
		response: warehouseBody,
		call: func(ctx context.Context, c *client.Client, bodyProject string, opts ...client.CallOption) (string, error) {
			req := newCreateWarehouseRequest()
			req.SetProjectId(bodyProject)
			_, err := c.Warehouses.Create(ctx, req, opts...)
			return req.GetProjectId(), err
		},
	},
	"Roles.Create": {
		pattern:  "POST /management/v1/role",
		response: roleBody,
		call: func(ctx context.Context, c *client.Client, bodyProject string, opts ...client.CallOption) (string, error) {
			req := managementv1.NewCreateRoleRequest("r")
			req.SetProjectId(bodyProject)
			_, err := c.Roles.Create(ctx, req, opts...)
			return req.GetProjectId(), err
		},
	},
	"Roles.Search": {
		pattern:  "POST /management/v1/search/role",
		response: `{"roles":[]}`,
		call: func(ctx context.Context, c *client.Client, bodyProject string, opts ...client.CallOption) (string, error) {
			req := managementv1.NewSearchRoleRequest("r")
			req.SetProjectId(bodyProject)
			_, err := c.Roles.Search(ctx, req, opts...)
			return req.GetProjectId(), err
		},
	},
}

const projectB = "22222222-2222-2222-2222-222222222222"

func TestBodyProject_IsSentAsHeaderOnly(t *testing.T) {
	t.Parallel()
	for name, tc := range bodyProjectCalls {
		for variant, opts := range map[string][]client.CallOption{
			"body only":         nil,
			"body equal to opt": {client.WithProject(projectB)},
		} {
			t.Run(name+"/"+variant, func(t *testing.T) {
				t.Parallel()
				mux, c := testutil.ServerMux(t)
				reqs := testutil.Respond(t, mux, tc.pattern, http.StatusOK, tc.response)

				callerBody, err := tc.call(t.Context(), c, projectB, opts...)

				require.NoError(t, err)
				sent := reqs.Last(t)
				assert.Equal(t, projectB, sent.Header.Get("x-project-id"))
				assert.NotContains(t, string(sent.Body), "project-id", "the deprecated body field must not be sent")
				assert.Equal(t, projectB, callerBody, "the caller's request must not be modified")
			})
		}
	}
}

func TestBodyProject_ConflictWithWithProjectIsRejected(t *testing.T) {
	t.Parallel()
	for name, tc := range bodyProjectCalls {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			mux, c := testutil.ServerMux(t)
			reqs := testutil.Respond(t, mux, tc.pattern, http.StatusOK, tc.response)

			_, err := tc.call(t.Context(), c, projectB, client.WithProject(projectA))

			require.ErrorIs(t, err, client.ErrProjectIDConflict)
			assert.Zero(t, reqs.Len(), "no request must be sent")
		})
	}
}

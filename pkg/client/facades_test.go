package client

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	managementv1 "github.com/lakekeeper/go-lakekeeper/pkg/apis/management/v1"
	"github.com/lakekeeper/go-lakekeeper/pkg/core"
)

// The façades are thin wrappers around generated fluent calls. The
// generated layer is exercised by integration tests against the live
// server; here we only assert the construction wiring (façades exist and
// reach the same APIClient) and the nil-request guards.

func newTestClient(t *testing.T) *Client {
	t.Helper()
	c, err := New(t.Context(), "https://example.invalid", "token", WithoutRetries())
	require.NoError(t, err)
	require.NotNil(t, c)
	return c
}

func TestFacades_AreInitialised(t *testing.T) {
	t.Parallel()
	c := newTestClient(t)

	// Each façade points at the same APIService instance the embedded
	// APIClient exposes. Pointer equality confirms the façade is just a
	// wrapper, not a parallel construction.
	assert.Same(t, c.WarehouseAPI, c.Warehouses.api)
	assert.Same(t, c.ProjectAPI, c.Projects.api)
	assert.Same(t, c.RoleAPI, c.Roles.api)
	assert.Same(t, c.UserAPI, c.Users.api)
	assert.Same(t, c.ServerAPI, c.Server.api)
}

func TestWarehouses_NilRequestsRejected(t *testing.T) {
	t.Parallel()
	c := newTestClient(t)
	ctx := context.Background()

	_, err := c.Warehouses.Create(ctx, nil)
	require.ErrorContains(t, err, "request must not be nil")

	_, err = c.Warehouses.Rename(ctx, "id", nil)
	require.ErrorContains(t, err, "request must not be nil")

	_, err = c.Warehouses.SetProtection(ctx, "id", nil)
	require.ErrorContains(t, err, "request must not be nil")
}

func TestProjects_NilRequestsRejected(t *testing.T) {
	t.Parallel()
	c := newTestClient(t)
	ctx := context.Background()

	_, err := c.Projects.Create(ctx, nil)
	require.ErrorContains(t, err, "request must not be nil")

	err = c.Projects.Rename(ctx, "p", nil)
	require.ErrorContains(t, err, "request must not be nil")
}

// Confirms the New / NewWithAuthSource symmetry covers the façade init.
func TestNewWithAuthSource_FaçadesInitialised(t *testing.T) {
	t.Parallel()
	c, err := NewWithAuthSource(t.Context(), "https://example.invalid",
		&core.AccessTokenAuthSource{Token: "tok"}, WithoutRetries())
	require.NoError(t, err)
	require.NotNil(t, c.Warehouses)
	require.NotNil(t, c.Projects)
	require.NotNil(t, c.Roles)
	require.NotNil(t, c.Users)
	require.NotNil(t, c.Server)

	// Build a request value (not sent) to confirm the type plumbing on the
	// embedded APIClient still works alongside the façades.
	req := managementv1.NewCreateProjectRequest("does-not-matter")
	require.NotNil(t, req)
}

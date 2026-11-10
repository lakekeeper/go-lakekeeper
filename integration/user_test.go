//go:build integration

package integration

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lakekeeper/go-lakekeeper/pkg/client"
)

func TestUser_Get(t *testing.T) {
	t.Parallel()
	c := sharedClient
	user := MustProvisionUser(t, c)

	got, err := c.Users.Get(t.Context(), user.GetId())

	require.NoError(t, err)
	assert.Equal(t, user.GetName(), got.GetName())
	assert.Equal(t, user.GetEmail(), got.GetEmail())
}

func TestUser_Get_NotFoundIsAPIError(t *testing.T) {
	t.Parallel()
	c := sharedClient

	_, err := c.Users.Get(t.Context(), fmt.Sprintf("oidc~%s", uuid.New()))

	var apiErr *client.APIError
	require.True(t, errors.As(err, &apiErr), "want *client.APIError, got %v", err)
	assert.Equal(t, http.StatusNotFound, apiErr.StatusCode)
}

// TestUser_List_NameFilter checks that the name filter is applied by the
// server: a page of one finds the user even though other users exist.
func TestUser_List_NameFilter(t *testing.T) {
	t.Parallel()
	c := sharedClient
	user := MustProvisionUser(t, c)
	MustProvisionUser(t, c)

	got, err := c.Users.List(t.Context(), &client.ListUserOptions{Name: user.GetName(), PageSize: 1})

	require.NoError(t, err)
	require.Len(t, got.GetUsers(), 1)
	assert.Equal(t, user.GetId(), got.GetUsers()[0].GetId())
}

func TestUser_Whoami(t *testing.T) {
	t.Parallel()
	c := sharedClient

	got, err := c.Users.Whoami(t.Context())

	require.NoError(t, err)
	assert.NotEmpty(t, got.GetId())
	assert.NotEmpty(t, got.GetUserType())
}

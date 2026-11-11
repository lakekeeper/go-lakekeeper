package client_test

import (
	"context"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	managementv1 "github.com/lakekeeper/go-lakekeeper/pkg/apis/management/v1"
	"github.com/lakekeeper/go-lakekeeper/pkg/client"
	"github.com/lakekeeper/go-lakekeeper/pkg/testutil"
)

var userEmptyList = map[string]any{"users": []any{}}

const (
	userID   = "oidc~d223d88c-85b6-4859-b5c5-27f3825e47f6"
	userBody = `{
  "id": "oidc~d223d88c-85b6-4859-b5c5-27f3825e47f6",
  "name": "Peter Cold",
  "email": "peter@example.com",
  "user-type": "human",
  "created-at": "2026-01-02T03:04:05Z",
  "updated-at": null,
  "last-updated-with": "create-endpoint"
}`
)

func TestUsers_NilRequestsRejected(t *testing.T) {
	t.Parallel()
	_, c := testutil.ServerMux(t)

	_, err := c.Users.Create(t.Context(), nil)
	require.ErrorContains(t, err, "request must not be nil")

	err = c.Users.Update(t.Context(), userID, nil)
	require.ErrorContains(t, err, "request must not be nil")
}

func TestUsers_List_ReturnsGeneratedResponse(t *testing.T) {
	t.Parallel()
	mux, c := testutil.ServerMux(t)
	testutil.Respond(t, mux, "GET /management/v1/user", http.StatusOK,
		`{"users":[`+userBody+`],"next-page-token":"tok-2"}`)

	got, err := c.Users.List(t.Context(), nil)

	require.NoError(t, err)
	assert.Equal(t, "tok-2", got.GetNextPageToken())
	require.Len(t, got.GetUsers(), 1)
	u := got.GetUsers()[0]
	assert.Equal(t, userID, u.GetId())
	assert.Equal(t, "peter@example.com", u.GetEmail())
}

func TestUsers_Get_ReturnsUser(t *testing.T) {
	t.Parallel()
	mux, c := testutil.ServerMux(t)
	reqs := testutil.Respond(t, mux, "GET /management/v1/user/{id}", http.StatusOK, userBody)

	got, err := c.Users.Get(t.Context(), userID)

	require.NoError(t, err)
	assert.Equal(t, "/management/v1/user/"+userID, reqs.Last(t).Path)
	assert.Equal(t, "Peter Cold", got.GetName())
	assert.Equal(t, managementv1.USERTYPE_HUMAN, got.GetUserType())
	assert.Equal(t, time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC), got.GetCreatedAt())
	assert.True(t, got.GetUpdatedAt().IsZero(), "a null updated-at reads as the zero time")
}

func TestUsers_Get_NotFoundIsAPIError(t *testing.T) {
	t.Parallel()
	mux, c := testutil.ServerMux(t)
	testutil.Respond(t, mux, "GET /management/v1/user/{id}", http.StatusNotFound,
		`{"error":{"code":404,"message":"User not found","type":"UserNotFound","stack":[]}}`)

	_, err := c.Users.Get(t.Context(), userID)

	require.ErrorContains(t, err, "get user: 404 Not Found")
	var apiErr *client.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, http.StatusNotFound, apiErr.StatusCode)
	require.NotNil(t, apiErr.Model)
	assert.Equal(t, "UserNotFound", apiErr.Model.GetType())
	var genErr *managementv1.GenericOpenAPIError
	require.ErrorAs(t, err, &genErr, "errors.As must reach the generated error")
}

func TestUsers_Create_SendsBodyAndReturnsUser(t *testing.T) {
	t.Parallel()
	mux, c := testutil.ServerMux(t)
	reqs := testutil.Respond(t, mux, "POST /management/v1/user", http.StatusCreated, userBody)

	req := managementv1.NewCreateUserRequest()
	req.SetId(userID)
	req.SetName("Peter Cold")
	req.SetUserType(managementv1.USERTYPE_HUMAN)
	req.SetEmail("peter@example.com")
	req.SetUpdateIfExists(true)
	got, err := c.Users.Create(t.Context(), req)

	require.NoError(t, err)
	assert.Equal(t, userID, got.GetId())
	sent := reqs.Last(t)
	assert.JSONEq(t, `{
  "id": "oidc~d223d88c-85b6-4859-b5c5-27f3825e47f6",
  "name": "Peter Cold",
  "user-type": "human",
  "email": "peter@example.com",
  "update-if-exists": true
}`, string(sent.Body))
	assert.NotContains(t, sent.Header, "X-Project-Id", "users are not project-scoped")
}

func TestUsers_Update_SendsBody(t *testing.T) {
	t.Parallel()
	mux, c := testutil.ServerMux(t)
	reqs := testutil.Respond(t, mux, "PUT /management/v1/user/{id}", http.StatusOK, nil)

	req := managementv1.NewUpdateUserRequest("Peter Warm", managementv1.USERTYPE_HUMAN)
	err := c.Users.Update(t.Context(), userID, req)

	require.NoError(t, err)
	sent := reqs.Last(t)
	assert.Equal(t, "/management/v1/user/"+userID, sent.Path)
	assert.JSONEq(t, `{"name":"Peter Warm","user-type":"human"}`, string(sent.Body))
}

func TestUsers_Delete_SendsDelete(t *testing.T) {
	t.Parallel()
	mux, c := testutil.ServerMux(t)
	reqs := testutil.Respond(t, mux, "DELETE /management/v1/user/{id}", http.StatusNoContent, nil)

	err := c.Users.Delete(t.Context(), userID)

	require.NoError(t, err)
	assert.Equal(t, "/management/v1/user/"+userID, reqs.Last(t).Path)
}

func TestUsers_Whoami_ReturnsCaller(t *testing.T) {
	t.Parallel()
	mux, c := testutil.ServerMux(t)
	testutil.Respond(t, mux, "GET /management/v1/whoami", http.StatusOK, `{
  "id": "oidc~d223d88c-85b6-4859-b5c5-27f3825e47f6",
  "name": "Peter Cold",
  "user-type": "application",
  "created-at": "2026-01-02T03:04:05Z",
  "last-updated-with": "config-call-creation",
  "is-instance-admin": true
}`)

	got, err := c.Users.Whoami(t.Context())

	require.NoError(t, err)
	assert.Equal(t, userID, got.GetId())
	assert.Equal(t, managementv1.USERTYPE_APPLICATION, got.GetUserType())
	assert.True(t, got.GetIsInstanceAdmin())
	assert.Empty(t, got.GetEmail(), "an absent email reads as empty")
}

func TestUsers_List_OptionsReachQuery(t *testing.T) {
	t.Parallel()
	mux, c := testutil.ServerMux(t)
	reqs := testutil.Respond(t, mux, "GET /management/v1/user", http.StatusOK, userEmptyList)

	_, err := c.Users.List(t.Context(), &client.ListUserOptions{
		Name:      "alice",
		PageSize:  25,
		PageToken: "tok-2",
	})

	require.NoError(t, err)
	assert.Equal(t, url.Values{
		"name":      {"alice"},
		"pageSize":  {"25"},
		"pageToken": {"tok-2"},
	}, reqs.Last(t).Query)
}

func TestUsers_List_EmptyOptionsSendNoQuery(t *testing.T) {
	t.Parallel()
	for name, opts := range map[string]*client.ListUserOptions{
		"nil":   nil,
		"empty": {},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			mux, c := testutil.ServerMux(t)
			reqs := testutil.Respond(t, mux, "GET /management/v1/user", http.StatusOK, userEmptyList)

			_, err := c.Users.List(t.Context(), opts)

			require.NoError(t, err)
			assert.Empty(t, reqs.Last(t).Query, "unset options must send no query parameters")
		})
	}
}

func TestUsers_Search_SendsBodyAndReturnsMatches(t *testing.T) {
	t.Parallel()
	mux, c := testutil.ServerMux(t)
	reqs := testutil.Respond(t, mux, "POST /management/v1/search/user", http.StatusOK, `{"users":[{
  "id": "oidc~d223d88c-85b6-4859-b5c5-27f3825e47f6",
  "name": "Peter Cold",
  "user-type": "human",
  "email": null
}]}`)

	got, err := c.Users.Search(t.Context(), managementv1.NewSearchUserRequest("peter"))

	require.NoError(t, err)
	require.Len(t, got.GetUsers(), 1)
	u := got.GetUsers()[0]
	assert.Equal(t, userID, u.GetId())
	assert.Equal(t, managementv1.USERTYPE_HUMAN, u.GetUserType())
	assert.Empty(t, u.GetEmail(), "a null email reads as empty")
	sent := reqs.Last(t)
	assert.JSONEq(t, `{"search":"peter"}`, string(sent.Body))
	assert.NotContains(t, sent.Header, "X-Project-Id", "users are not project-scoped")

	_, err = c.Users.Search(t.Context(), nil)
	require.ErrorContains(t, err, "search users: request must not be nil")
}

func TestUsers_Actions_SendsPrincipalFilter(t *testing.T) {
	t.Parallel()
	mux, c := testutil.ServerMux(t)
	reqs := testutil.Respond(t, mux, "GET /management/v1/user/{id}/actions", http.StatusOK,
		`{"allowed-actions":[{"action":"read"},{"action":"update"}]}`)

	got, err := c.Users.Actions(t.Context(), userID,
		&client.GetUserActionsOptions{PrincipalRole: roleID})

	require.NoError(t, err)
	require.Len(t, got.GetAllowedActions(), 2)
	assert.NotNil(t, got.GetAllowedActions()[0].LakekeeperUserActionRead)
	assert.NotNil(t, got.GetAllowedActions()[1].LakekeeperUserActionUpdate)
	sent := reqs.Last(t)
	assert.Equal(t, "/management/v1/user/"+userID+"/actions", sent.Path)
	assert.Equal(t, url.Values{"principalRole": {roleID}}, sent.Query)
}

func TestUsers_Actions_NilOptionsSendNoQuery(t *testing.T) {
	t.Parallel()
	mux, c := testutil.ServerMux(t)
	reqs := testutil.Respond(t, mux, "GET /management/v1/user/{id}/actions", http.StatusOK,
		`{"allowed-actions":[]}`)

	_, err := c.Users.Actions(t.Context(), userID, nil)

	require.NoError(t, err)
	assert.Empty(t, reqs.Last(t).Query)
}

const userRolesBody = `{"roles":[{
  "id": "0198618c-5be8-7a82-a0b9-1076c9dd12f0",
  "ident": "lakekeeper~0198618c-5be8-7a82-a0b9-1076c9dd12f0",
  "name": "data-eng"
}],"next-page-token":"tok-2"}`

// roleListings are the two role listings of Users; they share options,
// project handling, and response type.
var roleListings = map[string]struct {
	pattern string
	path    string
	call    func(context.Context, *client.Client, *client.ListUserRolesOptions, ...client.CallOption) (*managementv1.ListRoleMembershipsResponse, error)
}{
	"Roles": {
		pattern: "GET /management/v1/user/{id}/roles",
		path:    "/management/v1/user/" + userID + "/roles",
		call: func(ctx context.Context, c *client.Client, o *client.ListUserRolesOptions, co ...client.CallOption) (*managementv1.ListRoleMembershipsResponse, error) {
			return c.Users.Roles(ctx, userID, o, co...)
		},
	},
	"TransitiveRoles": {
		pattern: "GET /management/v1/user/{id}/roles/transitive",
		path:    "/management/v1/user/" + userID + "/roles/transitive",
		call: func(ctx context.Context, c *client.Client, o *client.ListUserRolesOptions, co ...client.CallOption) (*managementv1.ListRoleMembershipsResponse, error) {
			return c.Users.TransitiveRoles(ctx, userID, o, co...)
		},
	},
}

func TestUsers_RoleListings_SendPagingAndProject(t *testing.T) {
	t.Parallel()
	for name, l := range roleListings {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			mux, c := testutil.ServerMux(t)
			reqs := testutil.Respond(t, mux, l.pattern, http.StatusOK, userRolesBody)

			got, err := l.call(t.Context(), c, &client.ListUserRolesOptions{PageSize: 25, PageToken: "tok-1"},
				client.WithProject(projectA))

			require.NoError(t, err)
			assert.Equal(t, "tok-2", got.GetNextPageToken())
			require.Len(t, got.GetRoles(), 1)
			assert.Equal(t, roleID, got.GetRoles()[0].GetId())
			assert.Equal(t, "data-eng", got.GetRoles()[0].GetName())
			sent := reqs.Last(t)
			assert.Equal(t, l.path, sent.Path)
			assert.Equal(t, url.Values{"pageSize": {"25"}, "pageToken": {"tok-1"}}, sent.Query)
			assert.Equal(t, projectA, sent.Header.Get("x-project-id"))
		})
	}
}

func TestUsers_RoleListings_WithoutOptionsSendNothing(t *testing.T) {
	t.Parallel()
	for name, l := range roleListings {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			mux, c := testutil.ServerMux(t)
			reqs := testutil.Respond(t, mux, l.pattern, http.StatusOK, `{"roles":[]}`)

			got, err := l.call(t.Context(), c, nil)

			require.NoError(t, err)
			assert.Empty(t, got.GetNextPageToken(), "an absent token reads as empty")
			sent := reqs.Last(t)
			assert.Empty(t, sent.Query)
			assert.NotContains(t, sent.Header, "X-Project-Id", "without WithProject the server's default project applies")
		})
	}
}

func TestUsers_TransitiveRoles_NotImplementedIsAPIError(t *testing.T) {
	t.Parallel()
	mux, c := testutil.ServerMux(t)
	testutil.Respond(t, mux, "GET /management/v1/user/{id}/roles/transitive", http.StatusNotImplemented,
		`{"error":{"code":501,"message":"Not supported by the authorizer","type":"NotImplemented","stack":[]}}`)

	_, err := c.Users.TransitiveRoles(t.Context(), userID, nil)

	require.ErrorContains(t, err, "list user transitive roles: 501 Not Implemented")
	var apiErr *client.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, http.StatusNotImplemented, apiErr.StatusCode)
}

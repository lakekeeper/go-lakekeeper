package client_test

import (
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

package commands

import (
	"net/http"
	"net/url"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lakekeeper/go-lakekeeper/pkg/client"
	"github.com/lakekeeper/go-lakekeeper/pkg/testutil"
)

const (
	userTestID   = "oidc~d223d88c-85b6-4859-b5c5-27f3825e47f6"
	userTestBody = `{
  "id": "oidc~d223d88c-85b6-4859-b5c5-27f3825e47f6",
  "name": "Peter Cold",
  "email": "peter@example.com",
  "user-type": "human",
  "created-at": "2026-01-02T03:04:05Z",
  "last-updated-with": "create-endpoint"
}`
)

func TestNewUserCmdHasSubcommands(t *testing.T) {
	t.Parallel()

	cmd := newUserCmd(&clientOptions{})
	assert.Equal(t, "user", cmd.Use)

	got := []string{}
	for _, sub := range cmd.Commands() {
		got = append(got, sub.Name())
	}
	sort.Strings(got)
	assert.Equal(t, []string{"access", "create", "delete", "get", "list", "roles", "search"}, got)
}

func TestUserList_FlagsReachQuery(t *testing.T) {
	t.Parallel()
	mux, baseURL := testutil.NewServer(t)
	reqs := testutil.Respond(t, mux, "GET /management/v1/user", http.StatusOK,
		`{"users":[`+userTestBody+`],"next-page-token":"tok-3"}`)

	out, err := runLkctl(t, baseURL, "user", "list", "--name", "peter", "--limit", "25", "--token", "tok-2")

	require.NoError(t, err)
	assert.Equal(t, url.Values{
		"name":      {"peter"},
		"pageSize":  {"25"},
		"pageToken": {"tok-2"},
	}, reqs.Last(t).Query)
	assert.Contains(t, out, userTestID)
	assert.Contains(t, out, "peter@example.com")
	assert.Contains(t, out, "Next page token: tok-3")
}

func TestUserList_DefaultsSendPageSizeOnly(t *testing.T) {
	t.Parallel()
	mux, baseURL := testutil.NewServer(t)
	reqs := testutil.Respond(t, mux, "GET /management/v1/user", http.StatusOK, `{"users":[]}`)

	out, err := runLkctl(t, baseURL, "user", "list")

	require.NoError(t, err)
	assert.Equal(t, url.Values{"pageSize": {"100"}}, reqs.Last(t).Query)
	assert.NotContains(t, out, "Next page token", "no token line without a next page")
	assert.NotContains(t, reqs.Last(t).Header, "X-Project-Id")
}

func TestUserGet_PrintsUser(t *testing.T) {
	t.Parallel()
	mux, baseURL := testutil.NewServer(t)
	reqs := testutil.Respond(t, mux, "GET /management/v1/user/{id}", http.StatusOK, userTestBody)

	out, err := runLkctl(t, baseURL, "user", "get", userTestID, "-o", "wide")

	require.NoError(t, err)
	assert.Equal(t, "/management/v1/user/"+userTestID, reqs.Last(t).Path)
	assert.Contains(t, out, "Peter Cold")
	assert.Contains(t, out, "2026-01-02T03:04:05Z")
	assert.Contains(t, out, "create-endpoint")
}

// An empty 200 body must give an error, not a nil-pointer panic, in every
// get command that prints the returned entity.
func TestGetCommands_EmptySuccessBodyIsAnError(t *testing.T) {
	t.Parallel()
	for name, args := range map[string][]string{
		"user":      {"user", "get", userTestID},
		"role":      {"role", "get", "0198618c-5be8-7a82-a0b9-1076c9dd12f0"},
		"project":   {"project", "get", "11111111-1111-1111-1111-111111111111"},
		"warehouse": {"warehouse", "get", "9d6651da-5f19-11f1-ad69-1b7104fe3cf7"},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			mux, baseURL := testutil.NewServer(t)
			testutil.Respond(t, mux, "/", http.StatusOK, nil)

			require.NotPanics(t, func() {
				_, err := runLkctl(t, baseURL, args...)
				require.ErrorIs(t, err, client.ErrEmptyResponse)
			})
		})
	}
}

func TestUserGet_NotFoundKeepsStatus(t *testing.T) {
	t.Parallel()
	mux, baseURL := testutil.NewServer(t)
	testutil.Respond(t, mux, "GET /management/v1/user/{id}", http.StatusNotFound,
		`{"error":{"code":404,"message":"User not found","type":"UserNotFound","stack":[]}}`)

	_, err := runLkctl(t, baseURL, "user", "get", userTestID)

	require.ErrorContains(t, err, "get user: 404 Not Found")
	var apiErr *client.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, http.StatusNotFound, apiErr.StatusCode)
}

func TestUserCreate_FlagsReachBody(t *testing.T) {
	t.Parallel()
	mux, baseURL := testutil.NewServer(t)
	reqs := testutil.Respond(t, mux, "POST /management/v1/user", http.StatusCreated, userTestBody)

	out, err := runLkctl(t, baseURL, "user", "create", userTestID, "Peter Cold", "human",
		"--email", "peter@example.com", "--update")

	require.NoError(t, err)
	assert.JSONEq(t, `{
  "id": "oidc~d223d88c-85b6-4859-b5c5-27f3825e47f6",
  "name": "Peter Cold",
  "user-type": "human",
  "email": "peter@example.com",
  "update-if-exists": true
}`, string(reqs.Last(t).Body))
	assert.Equal(t, "User Peter Cold registered with id "+userTestID+"\n", out)
}

func TestUserDelete_SendsDelete(t *testing.T) {
	t.Parallel()
	mux, baseURL := testutil.NewServer(t)
	reqs := testutil.Respond(t, mux, "DELETE /management/v1/user/{id}", http.StatusNoContent, nil)

	out, err := runLkctl(t, baseURL, "user", "delete", userTestID)

	require.NoError(t, err)
	assert.Equal(t, "/management/v1/user/"+userTestID, reqs.Last(t).Path)
	assert.Equal(t, "User "+userTestID+" deleted\n", out)
}

func TestUserSearch_SendsTextAndPrintsMatches(t *testing.T) {
	t.Parallel()
	mux, baseURL := testutil.NewServer(t)
	reqs := testutil.Respond(t, mux, "POST /management/v1/search/user", http.StatusOK,
		`{"users":[{"id":"`+userTestID+`","name":"Peter Cold","email":"peter@example.com","user-type":"human"}]}`)

	out, err := runLkctl(t, baseURL, "user", "search", "peter")

	require.NoError(t, err)
	assert.JSONEq(t, `{"search":"peter"}`, string(reqs.Last(t).Body))
	assert.Contains(t, out, "USER TYPE")
	assert.Contains(t, out, userTestID)
	assert.Contains(t, out, "peter@example.com")
}

const userRolesTestBody = `{"roles":[{"id":"0198618c-5be8-7a82-a0b9-1076c9dd12f0",
  "ident":"lakekeeper~0198618c-5be8-7a82-a0b9-1076c9dd12f0","name":"data-eng"}],"next-page-token":"tok-3"}`

func TestUserRoles_FlagsReachRequest(t *testing.T) {
	t.Parallel()
	for name, tc := range map[string]struct {
		pattern string
		extra   []string
	}{
		"direct":     {pattern: "GET /management/v1/user/{id}/roles"},
		"transitive": {pattern: "GET /management/v1/user/{id}/roles/transitive", extra: []string{"--transitive"}},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			mux, baseURL := testutil.NewServer(t)
			reqs := testutil.Respond(t, mux, tc.pattern, http.StatusOK, userRolesTestBody)

			args := append([]string{"user", "roles", userTestID, "-p", "proj-1", "--limit", "25", "--token", "tok-2"}, tc.extra...)
			out, err := runLkctl(t, baseURL, args...)

			require.NoError(t, err)
			sent := reqs.Last(t)
			assert.Equal(t, url.Values{"pageSize": {"25"}, "pageToken": {"tok-2"}}, sent.Query)
			assert.Equal(t, "proj-1", sent.Header.Get("x-project-id"))
			assert.Contains(t, out, "data-eng")
			assert.Contains(t, out, "lakekeeper~0198618c-5be8-7a82-a0b9-1076c9dd12f0")
			assert.Contains(t, out, "Next page token: tok-3")
		})
	}
}

func TestUserRoles_DefaultsSendNoProject(t *testing.T) {
	t.Parallel()
	mux, baseURL := testutil.NewServer(t)
	reqs := testutil.Respond(t, mux, "GET /management/v1/user/{id}/roles", http.StatusOK, `{"roles":[]}`)

	out, err := runLkctl(t, baseURL, "user", "roles", userTestID)

	require.NoError(t, err)
	sent := reqs.Last(t)
	assert.Equal(t, url.Values{"pageSize": {"100"}}, sent.Query)
	assert.NotContains(t, sent.Header, "X-Project-Id", "without -p the server's default project applies")
	assert.NotContains(t, out, "Next page token")
}

func TestUserRoles_EmptyProjectIsRejected(t *testing.T) {
	t.Parallel()
	mux, baseURL := testutil.NewServer(t)
	reqs := testutil.Respond(t, mux, "/", http.StatusOK, `{}`)

	_, err := runLkctl(t, baseURL, "user", "roles", userTestID, "-p", "")

	require.ErrorIs(t, err, client.ErrEmptyProjectID)
	assert.Zero(t, reqs.Len())
}

func TestUserAccess_SendsPrincipalAndPrintsActions(t *testing.T) {
	t.Parallel()
	mux, baseURL := testutil.NewServer(t)
	reqs := testutil.Respond(t, mux, "GET /management/v1/user/{id}/actions", http.StatusOK,
		`{"allowed-actions":[{"action":"read"},{"action":"read_role_assignments"}]}`)

	out, err := runLkctl(t, baseURL, "user", "access", userTestID, "--user", "oidc~bob")

	require.NoError(t, err)
	assert.Equal(t, url.Values{"principalUser": {"oidc~bob"}}, reqs.Last(t).Query)
	assert.Contains(t, out, "ALLOWED ACTIONS")
	assert.Contains(t, out, "read\n")
	assert.Contains(t, out, "read_role_assignments")
}

func TestUserAccess_NoActionsPrintsNoAccess(t *testing.T) {
	t.Parallel()
	mux, baseURL := testutil.NewServer(t)
	testutil.Respond(t, mux, "GET /management/v1/user/{id}/actions", http.StatusOK, `{"allowed-actions":[]}`)

	out, err := runLkctl(t, baseURL, "user", "access", userTestID)

	require.NoError(t, err)
	assert.Contains(t, out, "No access")
}

func TestUserAccess_UserAndRoleAreExclusive(t *testing.T) {
	t.Parallel()
	_, baseURL := testutil.NewServer(t)

	_, err := runLkctl(t, baseURL, "user", "access", userTestID, "--user", "u", "--role", "r")

	require.ErrorContains(t, err, "--user and --role are mutually exclusive")
}

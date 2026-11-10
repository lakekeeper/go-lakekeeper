package commands

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	managementv1 "github.com/lakekeeper/go-lakekeeper/pkg/apis/management/v1"
	"github.com/lakekeeper/go-lakekeeper/pkg/client"
	"github.com/lakekeeper/go-lakekeeper/pkg/testutil"
)

const (
	roleTestProjectID = "11111111-1111-1111-1111-111111111111"
	roleTestID        = "0198618c-5be8-7a82-a0b9-1076c9dd12f0"
)

// roleTestJSON is a server role with the given id and name.
func roleTestJSON(id, name string) string {
	return `{"id":"` + id + `","name":"` + name + `","project-id":"` + roleTestProjectID + `",
	  "provider-id":"lakekeeper","source-id":"` + id + `","ident":"lakekeeper~` + id + `",
	  "created-at":"2026-01-02T03:04:05Z"}`
}

func TestRoleProjectFlag_DefaultsToServerDefaultProject(t *testing.T) {
	t.Parallel()

	flag := newRoleCmd(&clientOptions{}).PersistentFlags().Lookup("project")

	require.NotNil(t, flag)
	assert.Equal(t, "p", flag.Shorthand)
	assert.Empty(t, flag.DefValue, "-p must not default to a client-side project")
	assert.Contains(t, flag.Usage, "server's default project")
}

func TestRoleGet_WithoutProjectFlagSendsNoProject(t *testing.T) {
	t.Parallel()
	mux, baseURL := testutil.NewServer(t)
	reqs := testutil.Respond(t, mux, "GET /management/v1/role/{id}", http.StatusOK, roleTestJSON(roleTestID, "data-eng"))

	out, err := runLkctl(t, baseURL, "role", "get", roleTestID)

	require.NoError(t, err)
	assert.Contains(t, out, "data-eng")
	assert.NotContains(t, reqs.Last(t).Header, "X-Project-Id", "without -p no project header must be sent")
}

func TestRoleGet_ProjectFlagSendsHeader(t *testing.T) {
	t.Parallel()
	mux, baseURL := testutil.NewServer(t)
	reqs := testutil.Respond(t, mux, "GET /management/v1/role/{id}", http.StatusOK, roleTestJSON(roleTestID, "data-eng"))

	_, err := runLkctl(t, baseURL, "role", "get", roleTestID, "-p", roleTestProjectID)

	require.NoError(t, err)
	assert.Equal(t, roleTestProjectID, reqs.Last(t).Header.Get("x-project-id"))
}

func TestRoleCreate_SendsNameDescriptionAndProject(t *testing.T) {
	t.Parallel()
	mux, baseURL := testutil.NewServer(t)
	reqs := testutil.Respond(t, mux, "POST /management/v1/role", http.StatusCreated, roleTestJSON(roleTestID, "data-eng"))

	out, err := runLkctl(t, baseURL, "role", "create", "data-eng", "--description", "team", "-p", roleTestProjectID)

	require.NoError(t, err)
	assert.Equal(t, "Role data-eng created with id "+roleTestID+"\n", out)
	sent := reqs.Last(t)
	assert.Equal(t, roleTestProjectID, sent.Header.Get("x-project-id"))
	assert.JSONEq(t, `{"name":"data-eng","description":"team"}`, string(sent.Body))
}

func TestRoleUpdate_SendsNameAndProject(t *testing.T) {
	t.Parallel()
	mux, baseURL := testutil.NewServer(t)
	reqs := testutil.Respond(t, mux, "POST /management/v1/role/{id}", http.StatusOK, roleTestJSON(roleTestID, "renamed"))

	out, err := runLkctl(t, baseURL, "role", "update", roleTestID, "renamed", "-p", roleTestProjectID)

	require.NoError(t, err)
	assert.Equal(t, "Role "+roleTestID+" updated\n", out)
	sent := reqs.Last(t)
	assert.Equal(t, "/management/v1/role/"+roleTestID, sent.Path)
	assert.Equal(t, roleTestProjectID, sent.Header.Get("x-project-id"))
	assert.JSONEq(t, `{"name":"renamed"}`, string(sent.Body))
}

func TestRoleDelete_SendsProject(t *testing.T) {
	t.Parallel()
	mux, baseURL := testutil.NewServer(t)
	reqs := testutil.Respond(t, mux, "DELETE /management/v1/role/{id}", http.StatusNoContent, nil)

	out, err := runLkctl(t, baseURL, "role", "delete", roleTestID, "-p", roleTestProjectID)

	require.NoError(t, err)
	assert.Equal(t, "Role "+roleTestID+" deleted\n", out)
	sent := reqs.Last(t)
	assert.Equal(t, "/management/v1/role/"+roleTestID, sent.Path)
	assert.Equal(t, roleTestProjectID, sent.Header.Get("x-project-id"))
}

func TestRoleDelete_ServerErrorIsReturnedOnce(t *testing.T) {
	t.Parallel()
	mux, baseURL := testutil.NewServer(t)
	testutil.Respond(t, mux, "DELETE /management/v1/role/{id}", http.StatusNotFound,
		`{"error":{"code":404,"message":"Role not found","type":"RoleNotFound","stack":[]}}`)

	_, err := runLkctl(t, baseURL, "role", "delete", roleTestID)

	require.Error(t, err)
	assert.Equal(t, 1, strings.Count(err.Error(), "delete role:"), "the façade error must not be wrapped again: %v", err)
	assert.Contains(t, err.Error(), "404 Not Found")
}

func TestRoleList_PagingFlagsAndProject(t *testing.T) {
	t.Parallel()
	mux, baseURL := testutil.NewServer(t)
	reqs := testutil.Respond(t, mux, "GET /management/v1/role", http.StatusOK,
		`{"roles":[`+roleTestJSON(roleTestID, "data-eng")+`],"next-page-token":"page-3"}`)

	out, err := runLkctl(t, baseURL, "role", "list", "--limit", "1", "--token", "page-2", "-p", roleTestProjectID)

	require.NoError(t, err)
	assert.Contains(t, out, "data-eng")
	assert.Contains(t, out, "Next page token: page-3")
	sent := reqs.Last(t)
	assert.Equal(t, roleTestProjectID, sent.Header.Get("x-project-id"))
	assert.Equal(t, url.Values{"pageSize": {"1"}, "pageToken": {"page-2"}}, sent.Query)
}

const roleTestOtherID = "0198618c-5be8-7a82-a0b9-1076c9dd12f1"

func TestRoleListName_SearchHitKeepsOnlyExactMatch(t *testing.T) {
	t.Parallel()
	mux, baseURL := testutil.NewServer(t)
	searches := testutil.Respond(t, mux, "POST /management/v1/search/role", http.StatusOK,
		`{"roles":[`+roleTestJSON(roleTestOtherID, "data-eng-2")+`,`+roleTestJSON(roleTestID, "data-eng")+`]}`)
	lists := testutil.Respond(t, mux, "GET /management/v1/role", http.StatusOK, `{"roles":[]}`)

	out, err := runLkctl(t, baseURL, "role", "list", "--name", "data-eng", "-p", roleTestProjectID, "-o", "json")

	require.NoError(t, err)
	var roles []managementv1.Role
	require.NoError(t, json.Unmarshal([]byte(out), &roles), out)
	require.Len(t, roles, 1)
	assert.Equal(t, roleTestID, roles[0].GetId())
	sent := searches.Last(t)
	assert.Equal(t, roleTestProjectID, sent.Header.Get("x-project-id"))
	assert.JSONEq(t, `{"search":"data-eng"}`, string(sent.Body))
	assert.Zero(t, lists.Len(), "a search hit needs no listing")
}

// The server's role search returns at most 10 roles ranked by trigram distance,
// so a role named exactly X can be missing (ties at distance 0, names over 64
// characters). A miss must not be taken as "no such role".
func TestRoleListName_SearchMissFindsRoleOnLaterListPage(t *testing.T) {
	t.Parallel()
	mux, baseURL := testutil.NewServer(t)
	testutil.Respond(t, mux, "POST /management/v1/search/role", http.StatusOK,
		`{"roles":[`+roleTestJSON(roleTestOtherID, "data-eng-2")+`]}`)
	var (
		mu         sync.Mutex
		pageTokens []string
	)
	mux.HandleFunc("GET /management/v1/role", func(w http.ResponseWriter, r *http.Request) {
		token := r.URL.Query().Get("pageToken")
		mu.Lock()
		pageTokens = append(pageTokens, token)
		mu.Unlock()
		assert.Equal(t, roleTestProjectID, r.Header.Get("x-project-id"))
		w.Header().Set("Content-Type", "application/json")
		if token == "" {
			_, _ = w.Write([]byte(`{"roles":[` + roleTestJSON(roleTestOtherID, "data-eng-2") + `],"next-page-token":"page-2"}`))
			return
		}
		_, _ = w.Write([]byte(`{"roles":[` + roleTestJSON(roleTestID, "data-eng") + `]}`))
	})

	out, err := runLkctl(t, baseURL, "role", "list", "--name", "data-eng", "-p", roleTestProjectID, "-o", "json")

	require.NoError(t, err)
	var roles []managementv1.Role
	require.NoError(t, json.Unmarshal([]byte(out), &roles), out)
	require.Len(t, roles, 1)
	assert.Equal(t, roleTestID, roles[0].GetId())
	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, []string{"", "page-2"}, pageTokens, "the listing must be paged to the end")
}

// A server that hands back a page token it already gave would make the
// fallback listing loop forever; the command must stop with an error.
func TestRoleListName_RepeatedPageTokenStopsWithError(t *testing.T) {
	t.Parallel()
	mux, baseURL := testutil.NewServer(t)
	testutil.Respond(t, mux, "POST /management/v1/search/role", http.StatusOK, `{"roles":[]}`)
	var (
		mu    sync.Mutex
		calls int
	)
	mux.HandleFunc("GET /management/v1/role", func(w http.ResponseWriter, _ *http.Request) {
		mu.Lock()
		calls++
		n := calls
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		if n > 5 { // keeps a broken implementation from hanging the test
			_, _ = w.Write([]byte(`{"roles":[]}`))
			return
		}
		_, _ = w.Write([]byte(`{"roles":[],"next-page-token":"same"}`))
	})

	_, err := runLkctl(t, baseURL, "role", "list", "--name", "data-eng")

	require.ErrorContains(t, err, `repeated page token "same"`)
	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, 2, calls, "the listing must stop as soon as a token repeats")
}

func TestRoleListName_NoMatchAnywherePrintsNoRoles(t *testing.T) {
	t.Parallel()
	mux, baseURL := testutil.NewServer(t)
	testutil.Respond(t, mux, "POST /management/v1/search/role", http.StatusOK, `{"roles":[]}`)
	lists := testutil.Respond(t, mux, "GET /management/v1/role", http.StatusOK, `{"roles":[]}`)

	out, err := runLkctl(t, baseURL, "role", "list", "--name", "data-eng")

	require.NoError(t, err)
	assert.Equal(t, "No roles available\n", out)
	assert.Equal(t, 1, lists.Len())
	assert.NotContains(t, lists.Last(t).Header, "X-Project-Id", "without -p no project header must be sent")
}

func TestRoleListName_RejectsPageToken(t *testing.T) {
	t.Parallel()
	_, baseURL := testutil.NewServer(t)

	_, err := runLkctl(t, baseURL, "role", "list", "--name", "data-eng", "--token", "page-2")

	require.ErrorContains(t, err, "none of the others can be")
}

func TestNewRoleCmdSubcommands(t *testing.T) {
	t.Parallel()

	cmd := newRoleCmd(&clientOptions{})
	assert.Equal(t, "role", cmd.Use)

	got := []string{}
	for _, sub := range cmd.Commands() {
		got = append(got, sub.Name())
	}
	sort.Strings(got)
	assert.Equal(t, []string{"access", "assignments", "create", "delete", "get", "grant", "list", "revoke", "update"}, got)
}

// TestRoleRevokeNoUsersOrRoles verifies the pre-network guard: `lkctl role
// revoke <role-id> --assignments assignee` (without any --users or --roles)
// fails before newClient is called.
func TestRoleRevokeNoUsersOrRoles(t *testing.T) {
	t.Parallel()

	root := newRoleCmd(&clientOptions{})
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"revoke", "role-1", "--assignments", "assignee"})

	err := root.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "at least one --users or --roles")
}

func TestPrintRolesText(t *testing.T) {
	t.Parallel()

	role := managementv1.Role{
		Id:          "role-1",
		Name:        "data-eng",
		ProjectId:   "proj-1",
		CreatedAt:   time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
		Description: *managementv1.NewNullableString(managementv1.PtrString("data engineering team")),
	}

	var buf bytes.Buffer
	require.NoError(t, printRoles(&buf, "text", role))
	out := buf.String()

	assert.Contains(t, out, "ID")
	assert.Contains(t, out, "NAME")
	assert.Contains(t, out, "PROJECT ID")
	assert.Contains(t, out, "role-1")
	assert.Contains(t, out, "data-eng")
	assert.Contains(t, out, "proj-1")
	// Description is wide-only — must NOT appear in text output.
	assert.NotContains(t, out, "DESCRIPTION")
	assert.NotContains(t, out, "data engineering team")
}

func TestPrintRolesWide(t *testing.T) {
	t.Parallel()

	role := managementv1.Role{
		Id:          "role-1",
		Name:        "data-eng",
		ProjectId:   "proj-1",
		CreatedAt:   time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
		Description: *managementv1.NewNullableString(managementv1.PtrString("data engineering team")),
	}

	var buf bytes.Buffer
	require.NoError(t, printRoles(&buf, "wide", role))
	out := buf.String()

	assert.Contains(t, out, "DESCRIPTION")
	assert.Contains(t, out, "data engineering team")
}

// An explicit but empty -p (e.g. -p "$UNSET") is rejected; only an omitted
// -p leaves the project to the server.
func TestRoleDelete_EmptyProjectFlagIsRejected(t *testing.T) {
	t.Parallel()
	mux, baseURL := testutil.NewServer(t)
	reqs := testutil.Respond(t, mux, "/", http.StatusNoContent, nil)

	_, err := runLkctl(t, baseURL, "role", "delete", roleTestID, "-p", "")

	require.ErrorIs(t, err, client.ErrEmptyProjectID)
	assert.Zero(t, reqs.Len(), "no request must be sent")
}

package commands

import (
	"bytes"
	"net/http"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	managementv1 "github.com/lakekeeper/go-lakekeeper/pkg/apis/management/v1"
	"github.com/lakekeeper/go-lakekeeper/pkg/client"
	"github.com/lakekeeper/go-lakekeeper/pkg/testutil"
)

const projectTestID = "11111111-1111-1111-1111-111111111111"

func TestProjectGet_ArgumentSendsHeader(t *testing.T) {
	t.Parallel()
	mux, baseURL := testutil.NewServer(t)
	reqs := testutil.Respond(t, mux, "GET /management/v1/project", http.StatusOK,
		`{"project-id":"11111111-1111-1111-1111-111111111111","project-name":"Alpha"}`)

	out, err := runLkctl(t, baseURL, "project", "get", projectTestID)

	require.NoError(t, err)
	assert.Equal(t, projectTestID, reqs.Last(t).Header.Get("x-project-id"))
	assert.Contains(t, out, "Alpha")
}

func TestProjectDelete_ServerErrorKeepsStatus(t *testing.T) {
	t.Parallel()
	mux, baseURL := testutil.NewServer(t)
	reqs := testutil.Respond(t, mux, "DELETE /management/v1/project", http.StatusForbidden,
		`{"error":{"code":403,"message":"Forbidden","type":"NotAuthorized","stack":[]}}`)

	_, err := runLkctl(t, baseURL, "project", "delete", projectTestID)

	var apiErr *client.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, http.StatusForbidden, apiErr.StatusCode)
	assert.Equal(t, projectTestID, reqs.Last(t).Header.Get("x-project-id"))
}

func TestNewProjectCmdSubcommands(t *testing.T) {
	t.Parallel()

	cmd := newProjectCmd(&clientOptions{})
	assert.Equal(t, "project", cmd.Use)

	got := []string{}
	for _, sub := range cmd.Commands() {
		got = append(got, sub.Name())
	}
	sort.Strings(got)
	assert.Equal(t, []string{"access", "assignments", "create", "delete", "get", "grant", "list", "rename", "revoke"}, got)
}

// TestProjectRevokeNoUsersOrRoles verifies the pre-network guard: `lkctl
// project revoke --assignments project_admin` (without any --users or
// --roles) fails before newClient is called.
func TestProjectRevokeNoUsersOrRoles(t *testing.T) {
	t.Parallel()

	root := newProjectCmd(&clientOptions{})
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"revoke", projectTestID, "--assignments", "project_admin"})

	err := root.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "at least one --users or --roles")
}

func TestPrintProjects(t *testing.T) {
	t.Parallel()

	projects := []managementv1.GetProjectResponse{
		{ProjectId: "proj-1", ProjectName: "Alpha"},
		{ProjectId: "proj-2", ProjectName: "Beta"},
	}

	var buf bytes.Buffer
	require.NoError(t, printProjects(&buf, projects...))
	out := buf.String()

	assert.Contains(t, out, "ID")
	assert.Contains(t, out, "NAME")
	assert.Contains(t, out, "proj-1")
	assert.Contains(t, out, "Alpha")
	assert.Contains(t, out, "proj-2")
	assert.Contains(t, out, "Beta")
}

// TestProjectGrantNoUsersOrRoles verifies the pre-network guard: `lkctl
// project grant --assignments project_admin` (without any --users or --roles)
// fails before newClient is called.
func TestProjectGrantNoUsersOrRoles(t *testing.T) {
	t.Parallel()

	root := newProjectCmd(&clientOptions{})
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"grant", projectTestID, "--assignments", "project_admin"})

	err := root.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "at least one --users or --roles")
}

// TestProjectAccessUserRoleMutex verifies the pre-network guard: passing both
// --user and --role to `lkctl project access` fails before newClient is called.
func TestProjectAccessUserRoleMutex(t *testing.T) {
	t.Parallel()

	root := newProjectCmd(&clientOptions{})
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"access", "--user", "alice", "--role", "role-1"})

	err := root.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "mutually exclusive")
}

// An empty PROJECT-ID (e.g. an unset shell variable) must not fall back to
// the server's default project: for delete that would delete it.
func TestProjectCommands_EmptyProjectIDIsRejected(t *testing.T) {
	t.Parallel()
	for _, args := range [][]string{
		{"project", "delete", ""},
		{"project", "rename", "", "new-name"},
		{"project", "get", ""},
		{"project", "access", ""},
		{"project", "assignments", ""},
		{"project", "grant", "", "--users", "oidc~alice", "--assignments", "select"},
		{"project", "revoke", "", "--users", "oidc~alice", "--assignments", "select"},
	} {
		t.Run(args[1], func(t *testing.T) {
			t.Parallel()
			mux, baseURL := testutil.NewServer(t)
			reqs := testutil.Respond(t, mux, "/", http.StatusOK, `{}`)

			_, err := runLkctl(t, baseURL, args...)

			require.ErrorIs(t, err, client.ErrEmptyProjectID)
			assert.Zero(t, reqs.Len(), "no request must be sent")
		})
	}
}

package commands

import (
	"bytes"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	managementv1 "github.com/lakekeeper/go-lakekeeper/pkg/apis/management/v1"
)

func TestNewProjectCmdSubcommands(t *testing.T) {
	t.Parallel()

	cmd := newProjectCmd(&clientOptions{})
	assert.Equal(t, "project", cmd.Use)

	got := []string{}
	for _, sub := range cmd.Commands() {
		got = append(got, sub.Name())
	}
	sort.Strings(got)
	assert.Equal(t, []string{"access", "assignments", "create", "delete", "get", "grant", "list", "rename"}, got)
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
	root.SetArgs([]string{"grant", "--assignments", "project_admin"})

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

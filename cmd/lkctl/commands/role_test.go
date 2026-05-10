package commands

import (
	"bytes"
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	managementv1 "github.com/lakekeeper/go-lakekeeper/pkg/apis/management/v1"
)

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
		Description: managementv1.PtrString("data engineering team"),
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
		Description: managementv1.PtrString("data engineering team"),
	}

	var buf bytes.Buffer
	require.NoError(t, printRoles(&buf, "wide", role))
	out := buf.String()

	assert.Contains(t, out, "DESCRIPTION")
	assert.Contains(t, out, "data engineering team")
}

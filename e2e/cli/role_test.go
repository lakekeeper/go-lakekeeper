//go:build e2e_cli

// Role lifecycle plus role assignments — granting a role to a user, then to
// another role. Mirrors the CLI surface defined in cmd/lkctl/commands/role.go.

package clie2e

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRoleLifecycle(t *testing.T) {
	requireBackend(t, BackendCompose)
	t.Parallel()

	roleID := MustProvisionRole(t)

	getOut := runOK(t, "role", "get", roleID, "--output", "json")
	var got struct {
		ID string `json:"id"`
	}
	decodeJSON(t, getOut, &got)
	assert.Equal(t, roleID, got.ID)
}

func TestRoleGrantAssigneeToUser(t *testing.T) {
	requireBackend(t, BackendCompose)
	t.Parallel()

	roleID := MustProvisionRole(t)
	user := MustProvisionUser(t)

	runOK(t,
		"role", "grant", roleID,
		"--users", user,
		"--assignments", "assignee",
	)

	// --relations exercises server-side filtering on the CSV-encoded form.
	out := runOK(t,
		"role", "assignments", roleID,
		"--relations", "assignee",
		"--output", "json",
	)
	require.Contains(t, string(out), user)

	runOK(t,
		"role", "revoke", roleID,
		"--users", user,
		"--assignments", "assignee",
	)
}

//go:build e2e_cli

package clie2e

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUserCRUD(t *testing.T) {
	requireBackend(t, BackendCompose)
	t.Parallel()

	id := MustProvisionUser(t) // create + cleanup registered

	getOut := runOK(t, "user", "get", id, "--output", "json")
	var got struct {
		ID string `json:"id"`
	}
	decodeJSON(t, getOut, &got)
	assert.Equal(t, id, got.ID)
}

func TestUserSearchFindsUser(t *testing.T) {
	requireBackend(t, BackendCompose)
	t.Parallel()

	id := MustProvisionUser(t)
	var user struct {
		Name string `json:"name"`
	}
	decodeJSON(t, runOK(t, "user", "get", id, "--output", "json"), &user)

	// The provisioned name is random and unique, so it is the closest match.
	out := runOK(t, "user", "search", user.Name, "--output", "json")
	var found struct {
		Users []struct {
			ID string `json:"id"`
		} `json:"users"`
	}
	decodeJSON(t, out, &found)
	require.NotEmpty(t, found.Users)
	assert.Equal(t, id, found.Users[0].ID)
}

func TestUserRolesListsAssignedRole(t *testing.T) {
	requireBackend(t, BackendCompose)
	t.Parallel()

	roleID := MustProvisionRole(t)
	user := MustProvisionUser(t)
	runOK(t, "role", "grant", roleID, "--users", user, "--assignments", "assignee")
	t.Cleanup(func() {
		cleanupOK(t, "role", "revoke", roleID, "--users", user, "--assignments", "assignee")
	})

	out := runOK(t, "user", "roles", user, "--output", "json")
	var listed struct {
		Roles []struct {
			ID string `json:"id"`
		} `json:"roles"`
	}
	decodeJSON(t, out, &listed)
	require.Len(t, listed.Roles, 1)
	assert.Equal(t, roleID, listed.Roles[0].ID)
}

func TestUserAccessListsActions(t *testing.T) {
	requireBackend(t, BackendCompose)
	t.Parallel()

	id := MustProvisionUser(t)

	// The admin running the suite may read and change any user.
	out := runOK(t, "user", "access", id)
	assert.Contains(t, string(out), "ALLOWED ACTIONS")
	assert.Contains(t, string(out), "read")
}

// The compose stack runs OpenFGA, which manages role assignments itself, so
// the server does not implement transitive role listing there.
func TestUserTransitiveRolesNotImplementedWithOpenFGA(t *testing.T) {
	requireBackend(t, BackendCompose)
	t.Parallel()

	user := MustProvisionUser(t)

	_, stderr, code := runFail(t, "user", "roles", user, "--transitive")
	assert.NotZero(t, code)
	assert.Contains(t, string(stderr), "501")
}

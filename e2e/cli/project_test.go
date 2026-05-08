//go:build e2e_cli

// Project lifecycle and permissions, exercised end-to-end through the lkctl
// surface. State is read back via `project get` and `project assignments` —
// not the SDK — so any regression in CLI output formatting or arg wiring
// fails the test.

package clie2e

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProjectLifecycle(t *testing.T) {
	requireBackend(t, BackendCompose)
	t.Parallel()

	name := randomName("e2e-proj")

	// create
	createOut := runOK(t, "project", "create", name)
	id := parseProjectIDFromCreate(t, string(createOut))
	t.Cleanup(func() {
		// best-effort: ignore failure if delete already ran in-test
		_, _, _, _ = activeBackend.Exec(t.Context(), nil,
			append(authFlagsOAuth2(), "project", "delete", id)...)
	})
	require.NotEmpty(t, id)

	// rename
	newName := name + "-renamed"
	runOK(t, "project", "rename", id, newName)

	// get reflects the rename
	getOut := runOK(t, "project", "get", id, "--output", "json")
	var got struct {
		ProjectID   string `json:"project-id"`
		ProjectName string `json:"project-name"`
	}
	decodeJSON(t, getOut, &got)
	assert.Equal(t, id, got.ProjectID)
	assert.Equal(t, newName, got.ProjectName)

	// delete
	runOK(t, "project", "delete", id)

	// not-found exit asserted in errors_test.go; here we just confirm
	// further `get` fails (non-zero exit).
	_, _, code := runFail(t, "project", "get", id, "--output", "json")
	assert.NotZero(t, code)
}

func TestProjectGrantRevoke(t *testing.T) {
	requireBackend(t, BackendCompose)
	t.Parallel()

	user := MustProvisionUser(t)

	// grant project_admin to user on default project
	runOK(t,
		"project", "grant",
		"--users", user,
		"--assignments", "project_admin",
	)

	// assignments lists it.
	//
	// We deliberately omit --relations: the generated client emits
	// ?relations=foo&relations=bar (explode=true), but the server
	// expects a single CSV-encoded value (style=form, explode=false).
	// Filtering happens server-side once that mismatch is fixed in the
	// generator config; for now we read the unfiltered list and check
	// the user is present anywhere in the response.
	assignOut := runOK(t,
		"project", "assignments",
		"--output", "json",
	)
	require.Contains(t, string(assignOut), user, "expected assignment for user %s in %s", user, assignOut)

	// revoke
	runOK(t,
		"project", "revoke",
		"--users", user,
		"--assignments", "project_admin",
	)
}

// parseProjectIDFromCreate digs the UUID out of `Project NAME created with id UUID`.
func parseProjectIDFromCreate(t *testing.T, out string) string {
	t.Helper()
	id := parseIDFromCreate(t, out, "with id ")
	if _, err := uuid.Parse(id); err != nil {
		t.Fatalf("expected uuid in create output, got %q: %v", id, err)
	}
	return id
}

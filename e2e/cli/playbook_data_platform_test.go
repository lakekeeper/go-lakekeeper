//go:build e2e_cli

// Cross-command playbook: walk the canonical Lakekeeper demo flow end-to-end
// (project → warehouses → roles → users → grants → access checks) as a single
// assertable Go test. The lifecycle tests cover each verb-noun matrix in
// isolation; this test covers the contracts *between* them — e.g. that a role
// granted to a user actually surfaces in `warehouse access --user`.
//
// Compose-only: the kind harness has no MinIO/S3 store deployed, so warehouse
// creation isn't possible there today (see /e2e/kind/values.yaml).

package clie2e

import (
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPlaybookAnalyticsOnboarding mirrors the "analytics team onboarding"
// narrative from examples/init.sh: stand up a project, two warehouses, two
// roles, three users; wire role grants through to user access; verify each
// assignment is observable through the CLI; assert that revoke + delete are
// orderable without leaking state.
func TestPlaybookAnalyticsOnboarding(t *testing.T) {
	requireBackend(t, BackendCompose)
	// Intentionally not t.Parallel: this scenario touches enough shared
	// resources (a project's worth of warehouses, roles, users) that running
	// it sequentially keeps cleanup ordering predictable. The lifecycle
	// tests already exercise concurrency on the individual verbs.

	suffix := uuid.NewString()[:8]

	// 1. Project ------------------------------------------------------
	projectName := "analytics-" + suffix
	createProjOut := runOK(t, "project", "create", projectName)
	projectID := parseIDFromCreate(t, string(createProjOut))
	if _, err := uuid.Parse(projectID); err != nil {
		t.Fatalf("expected uuid in project create output, got %q: %v", projectID, err)
	}
	t.Cleanup(func() {
		// Project deletion runs LAST in LIFO. By the time it executes,
		// warehouses / roles / users registered below are already gone.
		cleanupOK(t, "project", "delete", projectID)
	})

	// 2. Two MinIO-backed warehouses ---------------------------------
	rawWHID, rawWHName := createPlaybookWarehouse(t, projectID, "raw-"+suffix)
	curatedWHID, curatedWHName := createPlaybookWarehouse(t, projectID, "curated-"+suffix)

	// 3. Two roles ---------------------------------------------------
	engRoleID := createPlaybookRole(t, projectID, "data-engineer-"+suffix)
	analystRoleID := createPlaybookRole(t, projectID, "data-analyst-"+suffix)

	// 4. Three users -------------------------------------------------
	alice := createPlaybookUser(t, "alice-"+suffix)
	bob := createPlaybookUser(t, "bob-"+suffix)
	carol := createPlaybookUser(t, "carol-"+suffix)

	// 5. Warehouse permissions ---------------------------------------
	// data-engineer: modify on raw, select on curated.
	grantWarehouseToRole(t, rawWHID, engRoleID, "modify")
	grantWarehouseToRole(t, curatedWHID, engRoleID, "select")
	// data-analyst: select on curated.
	grantWarehouseToRole(t, curatedWHID, analystRoleID, "select")

	// 6. Project admin on the new project for carol ------------------
	runOK(t, "project", "grant", projectID,
		"--users", carol,
		"--assignments", "project_admin",
	)
	t.Cleanup(func() {
		cleanupOK(t, "project", "revoke", projectID,
			"--users", carol,
			"--assignments", "project_admin",
		)
	})

	// 7. Role assignments (alice → eng, bob → analyst) ---------------
	grantRoleToUser(t, engRoleID, alice)
	grantRoleToUser(t, analystRoleID, bob)

	// 8. Assignments are visible -------------------------------------
	// Role assignments: each role mentions the user we just granted.
	engAssignOut := runOK(t, "role", "assignments", engRoleID, "--output", "json")
	require.Contains(t, string(engAssignOut), alice,
		"data-engineer role assignments must list alice")

	analystAssignOut := runOK(t, "role", "assignments", analystRoleID, "--output", "json")
	require.Contains(t, string(analystAssignOut), bob,
		"data-analyst role assignments must list bob")

	// Warehouse assignments mention the role granted to them.
	rawAssignOut := runOK(t, "warehouse", "assignments", rawWHID, "--output", "json")
	require.Contains(t, string(rawAssignOut), engRoleID,
		"raw warehouse assignments must list data-engineer role")

	curatedAssignOut := runOK(t, "warehouse", "assignments", curatedWHID, "--output", "json")
	require.Contains(t, string(curatedAssignOut), engRoleID,
		"curated warehouse assignments must list data-engineer role")
	require.Contains(t, string(curatedAssignOut), analystRoleID,
		"curated warehouse assignments must list data-analyst role")

	// Project assignments mention carol.
	projAssignOut := runOK(t, "project", "assignments", projectID, "--output", "json")
	require.Contains(t, string(projAssignOut), carol,
		"project assignments must list carol")

	// 9. Access propagates from role grants to user ------------------
	// alice has modify on raw → modify-tier actions present.
	aliceOnRaw := decodePlaybookActions(t, runOK(t,
		"warehouse", "access", rawWHID, "--user", alice, "--output", "json"))
	assert.NotEmpty(t, aliceOnRaw, "alice should have actions on raw via data-engineer")
	assert.Contains(t, aliceOnRaw, "modify_storage",
		"alice on %s (raw) should have modify-tier action via data-engineer role", rawWHName)

	// alice has select on curated → select-tier actions, no modify-tier.
	aliceOnCurated := decodePlaybookActions(t, runOK(t,
		"warehouse", "access", curatedWHID, "--user", alice, "--output", "json"))
	assert.NotEmpty(t, aliceOnCurated, "alice should have select on curated via data-engineer")
	assert.NotContains(t, aliceOnCurated, "modify_storage",
		"alice on %s (curated) should NOT have modify-tier access (select only)", curatedWHName)

	// bob has select on curated.
	bobOnCurated := decodePlaybookActions(t, runOK(t,
		"warehouse", "access", curatedWHID, "--user", bob, "--output", "json"))
	assert.NotEmpty(t, bobOnCurated, "bob should have select on curated via data-analyst")
	assert.NotContains(t, bobOnCurated, "modify_storage",
		"bob on %s (curated) should NOT have modify-tier access", curatedWHName)

	// bob has no access on raw. The endpoint either returns 200 with an
	// empty list or 403; the permissive check is "no modify-tier action
	// present in whatever the call returned". Use runRaw so a 403 doesn't
	// fail the test.
	bobOnRawStdout, _, _ := runRaw(t, nil,
		append(authFlagsOAuth2(), "warehouse", "access", rawWHID, "--user", bob, "--output", "json")...,
	)
	assert.NotContains(t, string(bobOnRawStdout), "modify_storage",
		"bob on raw should NOT have modify-tier access (no grant)")

	// 10. Statistics + listings smoke check --------------------------
	for _, whID := range []string{rawWHID, curatedWHID} {
		statsOut := runOK(t, "warehouse", "statistics", whID, "--output", "json")
		var stats struct {
			Stats []map[string]any `json:"stats"`
		}
		decodeJSON(t, statsOut, &stats)
	}

	listOut := runOK(t, "warehouse", "list", "--project", projectID, "--output", "json")
	listStr := string(listOut)
	assert.Contains(t, listStr, rawWHID, "warehouse list must include raw")
	assert.Contains(t, listStr, curatedWHID, "warehouse list must include curated")

	// 11. Negative assertions ----------------------------------------
	// 11a. Local validation rejects an unknown relation before any
	// network call.
	_, stderr, code := runRaw(t, nil,
		append(authFlagsOAuth2(),
			"warehouse", "grant", rawWHID,
			"--users", alice,
			"--assignments", "not_a_real_relation",
		)...)
	assert.NotZero(t, code, "expected non-zero exit on unknown relation")
	assert.True(t, strings.Contains(string(stderr), "relation") ||
		strings.Contains(string(stderr), "unknown") ||
		strings.Contains(string(stderr), "invalid"),
		"expected relation-validation diagnostic in stderr, got %q", string(stderr))

	// 11b. Granting on a non-existent warehouse ID fails server-side.
	bogusWH := "11111111-2222-3333-4444-555555555555"
	_, _, code2 := runRaw(t, nil,
		append(authFlagsOAuth2(),
			"warehouse", "grant", bogusWH,
			"--users", alice,
			"--assignments", "describe",
		)...)
	assert.NotZero(t, code2, "expected non-zero exit when granting on missing warehouse")
}

// createPlaybookWarehouse provisions a MinIO-backed warehouse in the given
// project and registers an idempotent cleanup. Returns (id, name). Mirrors
// MustProvisionWarehouse but takes an explicit project so the playbook can
// scope its warehouses to the new project rather than the default one.
func createPlaybookWarehouse(t *testing.T, projectID, name string) (string, string) {
	t.Helper()

	cfg := warehouseConfig(name)
	args := append(authFlagsOAuth2(),
		"warehouse", "create", name,
		"--project", projectID,
		"-f", "-",
	)
	stdout, stderr, code := runRaw(t, cfg, args...)
	if code != 0 {
		t.Fatalf("create warehouse %s exit %d\nstdout: %s\nstderr: %s", name, code, stdout, stderr)
	}
	id := parseIDFromCreate(t, string(stdout))

	t.Cleanup(func() {
		cleanupOK(t, "warehouse", "delete", id)
	})
	return id, name
}

// createPlaybookRole provisions a role on the given project and registers an
// idempotent cleanup. Returns the role id.
func createPlaybookRole(t *testing.T, projectID, name string) string {
	t.Helper()

	out := runOK(t, "role", "create", name, "--project", projectID, "--output", "json")
	var role struct {
		ID string `json:"id"`
	}
	decodeJSON(t, out, &role)
	if role.ID == "" {
		t.Fatalf("create role %s: empty id in output %s", name, out)
	}
	t.Cleanup(func() {
		cleanupOK(t, "role", "delete", role.ID, "--project", projectID)
	})
	return role.ID
}

// createPlaybookUser provisions a fresh OIDC user named after the persona
// (alice/bob/carol) and registers cleanup. Returns the OIDC subject id.
func createPlaybookUser(t *testing.T, persona string) string {
	t.Helper()

	id := "oidc~" + uuid.NewString()
	runOK(t, "user", "create", id, persona, "human", "--email", persona+"@example.com")
	t.Cleanup(func() {
		cleanupOK(t, "user", "delete", id)
	})
	return id
}

// grantWarehouseToRole adds a warehouse assignment for the given role and
// registers a revoke cleanup. LIFO ordering puts the revoke before the role
// and warehouse deletes, so revoking succeeds against still-existing entities.
func grantWarehouseToRole(t *testing.T, warehouseID, roleID, relation string) {
	t.Helper()

	runOK(t, "warehouse", "grant", warehouseID,
		"--roles", roleID,
		"--assignments", relation,
	)
	t.Cleanup(func() {
		cleanupOK(t, "warehouse", "revoke", warehouseID,
			"--roles", roleID,
			"--assignments", relation,
		)
	})
}

// grantRoleToUser assigns the user as an assignee of the role and registers
// a revoke cleanup.
func grantRoleToUser(t *testing.T, roleID, userID string) {
	t.Helper()

	runOK(t, "role", "grant", roleID,
		"--users", userID,
		"--assignments", "assignee",
	)
	t.Cleanup(func() {
		cleanupOK(t, "role", "revoke", roleID,
			"--users", userID,
			"--assignments", "assignee",
		)
	})
}

// decodePlaybookActions extracts the allowed-actions array from a
// `warehouse access` JSON response. Kept in this file rather than promoting
// to helpers_test.go because no other test reads the field today; promote
// when a second caller appears.
func decodePlaybookActions(t *testing.T, raw []byte) []string {
	t.Helper()

	var resp struct {
		AllowedActions []string `json:"allowed-actions"`
	}
	decodeJSON(t, raw, &resp)
	return resp.AllowedActions
}

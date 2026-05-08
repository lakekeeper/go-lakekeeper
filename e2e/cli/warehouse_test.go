//go:build e2e_cli

// Warehouse lifecycle, statistics, protection toggle, and permissions —
// driven entirely through lkctl. Compose-only: kind doesn't add coverage
// here, only latency.

package clie2e

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWarehouseLifecycle(t *testing.T) {
	requireBackend(t, BackendCompose)
	t.Parallel()

	id, _ := MustProvisionWarehouse(t)

	// get -o json round-trips the resource we just created
	getOut := runOK(t, "warehouse", "get", id, "--output", "json")
	require.Contains(t, string(getOut), id)

	// deactivate / activate
	runOK(t, "warehouse", "deactivate", id)
	runOK(t, "warehouse", "activate", id)

	// set-protection on then off. The CLI emits indented JSON, so decode
	// rather than substring-match — `"protected": true` (with a space)
	// would slip past a naive Contains.
	protectOn := runOK(t, "warehouse", "set-protection", id, "--protected=true", "--output", "json")
	var on struct {
		Protected bool `json:"protected"`
	}
	decodeJSON(t, protectOn, &on)
	require.True(t, on.Protected)

	protectOff := runOK(t, "warehouse", "set-protection", id, "--protected=false", "--output", "json")
	var off struct {
		Protected bool `json:"protected"`
	}
	decodeJSON(t, protectOff, &off)
	require.False(t, off.Protected)
}

func TestWarehouseStatistics(t *testing.T) {
	requireBackend(t, BackendCompose)
	t.Parallel()

	id, _ := MustProvisionWarehouse(t)

	// statistics: an empty warehouse may return zero or one sample point;
	// either way the call must succeed and JSON must be parseable.
	out := runOK(t, "warehouse", "statistics", id, "--output", "json")
	var resp struct {
		Stats []map[string]any `json:"stats"`
	}
	decodeJSON(t, out, &resp)
}

func TestWarehouseGrantRevoke(t *testing.T) {
	requireBackend(t, BackendCompose)
	t.Parallel()

	whID, _ := MustProvisionWarehouse(t)
	user := MustProvisionUser(t)

	runOK(t,
		"warehouse", "grant", whID,
		"--users", user,
		"--assignments", "describe",
	)

	// --relations is intentionally omitted; see project_test.go for the
	// generator-config mismatch behind that decision.
	out := runOK(t,
		"warehouse", "assignments", whID,
		"--output", "json",
	)
	require.Contains(t, string(out), user)

	// access for the granted user surfaces non-empty allowed actions
	accessOut := runOK(t,
		"warehouse", "access", whID,
		"--user", user,
		"--output", "json",
	)
	var access struct {
		AllowedActions []string `json:"allowed-actions"`
	}
	decodeJSON(t, accessOut, &access)
	assert.NotEmpty(t, access.AllowedActions, "describe grant should yield non-empty allowed-actions")

	runOK(t,
		"warehouse", "revoke", whID,
		"--users", user,
		"--assignments", "describe",
	)

	// after revoke, the assignments listing no longer mentions the user
	postOut := runOK(t,
		"warehouse", "assignments", whID,
		"--output", "json",
	)
	assert.NotContains(t, string(postOut), user)
}

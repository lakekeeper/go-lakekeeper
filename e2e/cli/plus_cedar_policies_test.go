//go:build e2e_cli && e2e_plus

package clie2e

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// cedarApplyRequest is an apply request body that writes one policy, named
// name, permitting a user no test signs in as: it changes nobody's access.
func cedarApplyRequest(t *testing.T, name string) []byte {
	t.Helper()
	body, err := json.Marshal(map[string]any{
		"writes": []any{map[string]any{
			"name":        name,
			"description": "lkctl e2e",
			"policy": map[string]any{
				"effect": "permit",
				"principal": map[string]any{
					"op":     "==",
					"entity": map[string]any{"type": "Lakekeeper::User", "id": "oidc~" + randomUUID()},
				},
				"action":     map[string]any{"op": "All"},
				"resource":   map[string]any{"op": "All"},
				"conditions": []any{},
			},
		}},
	})
	require.NoError(t, err)
	return body
}

// cedarApply runs an apply command with body on stdin and returns stdout.
func cedarApply(t *testing.T, body []byte, args ...string) []byte {
	t.Helper()
	full := append(authFlagsOAuth2(), args...)
	stdout, stderr, code := runRaw(t, body, append(full, "-f", "-", "--break-glass", "lkctl e2e")...)
	require.Zerof(t, code, "lkctl %v: exit %d\nstdout: %s\nstderr: %s", args, code, stdout, stderr)
	return stdout
}

// cedarDelete deletes the named policy through apply. It is for t.Cleanup,
// so like cleanupOK it runs on a fresh context and reports with t.Errorf.
func cedarDelete(t *testing.T, name string, args ...string) {
	t.Helper()
	body, err := json.Marshal(map[string]any{"deletes": []string{name}})
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	full := append(append(authFlagsOAuth2(), args...), "-f", "-", "--break-glass", "lkctl e2e cleanup")
	stdout, stderr, code, err := activeBackend.Exec(ctx, body, full...)
	if err != nil || code != 0 {
		t.Errorf("cleanup lkctl %v: exit %d, err %v\nstdout: %s\nstderr: %s", args, code, err, stdout, stderr)
	}
}

type scopePolicies struct {
	Policies []struct {
		ID     string         `json:"id"`
		Name   string         `json:"name"`
		Policy map[string]any `json:"policy"`
	} `json:"policies"`
	ScopeVersion int64 `json:"scope-version"`
}

func (p scopePolicies) names() []string {
	var names []string
	for _, e := range p.Policies {
		names = append(names, e.Name)
	}
	return names
}

func TestPlusCedarProjectPoliciesApplyListGet(t *testing.T) {
	requireBackend(t, BackendComposePlus)
	t.Parallel()

	name := randomName("e2e-project-policy")
	applied := cedarApply(t, cedarApplyRequest(t, name), "cedar", "project-policies", "apply")
	t.Cleanup(func() {
		cedarDelete(t, name, "cedar", "project-policies", "apply")
	})
	assert.Regexp(t, `(?m)^create\s+`+name+`\s+\S+$`, string(applied))
	assert.Regexp(t, `(?m)^Scope version: \d+$`, string(applied))

	var listed scopePolicies
	decodeJSON(t, runOK(t, "cedar", "project-policies", "list", "-o", "json"), &listed)
	assert.Contains(t, listed.names(), name)
	assert.Contains(t, string(runOK(t, "cedar", "project-policies", "list")), name)

	var got struct {
		Name   string         `json:"name"`
		Policy map[string]any `json:"policy"`
	}
	decodeJSON(t, runOK(t, "cedar", "project-policies", "get", name, "-o", "json"), &got)
	assert.Equal(t, name, got.Name)
	assert.Equal(t, "permit", got.Policy["effect"])
	assert.Regexp(t, `(?m)^\S+\s+`+name+`\s+`, string(runOK(t, "cedar", "project-policies", "get", name)))
}

func TestPlusCedarWarehousePoliciesApplyListGet(t *testing.T) {
	requireBackend(t, BackendComposePlus)
	t.Parallel()

	whID, _ := MustProvisionWarehouse(t)
	name := randomName("e2e-warehouse-policy")
	applied := cedarApply(t, cedarApplyRequest(t, name), "cedar", "warehouse-policies", "apply", whID)
	t.Cleanup(func() {
		cedarDelete(t, name, "cedar", "warehouse-policies", "apply", whID)
	})
	assert.Regexp(t, `(?m)^create\s+`+name+`\s+\S+$`, string(applied))

	var listed scopePolicies
	decodeJSON(t, runOK(t, "cedar", "warehouse-policies", "list", whID, "-o", "json"), &listed)
	assert.Equal(t, []string{name}, listed.names())

	var got struct {
		Name string `json:"name"`
	}
	decodeJSON(t, runOK(t, "cedar", "warehouse-policies", "get", whID, name, "-o", "json"), &got)
	assert.Equal(t, name, got.Name)
}

// TestPlusCedarWarehousePoliciesMissingWarehouseIsNotFound checks that a
// genuine not-found on a Lakekeeper+ path stays a not-found: the server can
// run Cedar, so it must not be reported as unsupported.
func TestPlusCedarWarehousePoliciesMissingWarehouseIsNotFound(t *testing.T) {
	requireBackend(t, BackendComposePlus)
	t.Parallel()

	_, stderr, _ := runFail(t, "cedar", "warehouse-policies", "list", randomUUID())

	assert.Contains(t, string(stderr), "list warehouse cedar policies: ")
	assert.Contains(t, string(stderr), "404")
	assert.NotContains(t, string(stderr), "requires")
}

func TestPlusCedarBreakGlassStatus(t *testing.T) {
	requireBackend(t, BackendComposePlus)
	t.Parallel()

	var status struct {
		Available *bool `json:"break-glass-available"`
	}
	decodeJSON(t, runOK(t, "cedar", "break-glass-status", "-o", "json"), &status)
	require.NotNil(t, status.Available, "break-glass-available")

	assert.Regexp(t, `^Break-glass available: (true|false)\n$`, string(runOK(t, "cedar", "break-glass-status")))
}

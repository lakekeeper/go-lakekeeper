//go:build e2e_cli && e2e_plus

package clie2e

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type predefinedPolicies struct {
	Curated  *bool `json:"curated"`
	Policies []struct {
		ID         string  `json:"id"`
		Enabled    bool    `json:"enabled"`
		CanToggle  bool    `json:"can-toggle"`
		DisabledBy *string `json:"disabled-by"`
		Retired    bool    `json:"retired"`
	} `json:"policies"`
	PredefinedPoliciesEnabled *bool `json:"predefined-policies-enabled"`
}

// toggleable returns the id of the first enabled, switchable, provided policy.
func (p predefinedPolicies) toggleable(t *testing.T) string {
	t.Helper()
	for _, e := range p.Policies {
		if e.Enabled && e.CanToggle && !e.Retired {
			return e.ID
		}
	}
	t.Fatalf("no enabled predefined policy the caller can switch: %+v", p.Policies)
	return ""
}

func (p predefinedPolicies) status(t *testing.T, id string) (enabled bool, disabledBy string) {
	t.Helper()
	for _, e := range p.Policies {
		if e.ID == id {
			if e.DisabledBy != nil {
				disabledBy = *e.DisabledBy
			}
			return e.Enabled, disabledBy
		}
	}
	t.Fatalf("predefined policy %s not listed", id)
	return false, ""
}

// cedarResetPredefined resets a scope's predefined policies. It is for
// t.Cleanup, so like cleanupOK it runs on a fresh context.
func cedarResetPredefined(t *testing.T, args ...string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	full := append(backendAdminFlags(), args...)
	stdout, stderr, code, err := activeBackend.Exec(ctx, nil, full...)
	if err != nil || code != 0 {
		t.Errorf("cleanup lkctl %v: exit %d, err %v\nstdout: %s\nstderr: %s", args, code, err, stdout, stderr)
	}
}

// TestPlusCedarProjectPredefinedPoliciesSetAndReset switches a predefined
// policy of the default project. It is not parallel: a warehouse listing
// reflects its project's decisions, so no other test may run meanwhile.
func TestPlusCedarProjectPredefinedPoliciesSetAndReset(t *testing.T) {
	requireBackend(t, BackendComposePlus)

	group := []string{"cedar", "project-predefined-policies"}
	cmd := func(args ...string) []string { return append(append([]string{}, group...), args...) }
	t.Cleanup(func() { cedarResetPredefined(t, cmd("reset")...) })

	var before predefinedPolicies
	decodeJSON(t, runOK(t, cmd("list", "-o", "json")...), &before)
	require.NotNil(t, before.Curated, "curated")
	require.NotNil(t, before.PredefinedPoliciesEnabled, "predefined-policies-enabled")
	id := before.toggleable(t)

	text := string(runOK(t, cmd("list")...))
	assert.Regexp(t, `(?m)^ID\s+ENABLED\s+DISABLED BY\s+CAN TOGGLE$`, text)
	assert.Regexp(t, `(?m)^`+regexp.QuoteMeta(id)+`\s+true\s+-\s+true$`, text)
	assert.Regexp(t, `(?m)^Curated: (true|false)$`, text)

	set := string(runOK(t, cmd("set", "--disable", id, "--break-glass", "lkctl e2e")...))
	assert.Regexp(t, `(?m)^`+regexp.QuoteMeta(id)+`\s+false\s+project\s+`, set)
	assert.Contains(t, set, "Curated: true\n")

	var disabled predefinedPolicies
	decodeJSON(t, runOK(t, cmd("list", "-o", "json")...), &disabled)
	enabled, by := disabled.status(t, id)
	assert.False(t, enabled)
	assert.Equal(t, "project", by)
	assert.True(t, *disabled.Curated)

	assert.Equal(t, "Predefined policies of the project reset\n", string(runOK(t, cmd("reset")...)))

	var after predefinedPolicies
	decodeJSON(t, runOK(t, cmd("list", "-o", "json")...), &after)
	enabled, by = after.status(t, id)
	assert.True(t, enabled)
	assert.Empty(t, by)
	assert.False(t, *after.Curated)
}

func TestPlusCedarWarehousePredefinedPoliciesSetAndReset(t *testing.T) {
	requireBackend(t, BackendComposePlus)
	t.Parallel()

	whID, _ := MustProvisionWarehouse(t)
	group := []string{"cedar", "warehouse-predefined-policies"}
	cmd := func(args ...string) []string { return append(append([]string{}, group...), args...) }
	t.Cleanup(func() { cedarResetPredefined(t, cmd("reset", whID)...) })

	var before predefinedPolicies
	decodeJSON(t, runOK(t, cmd("list", whID, "-o", "json")...), &before)
	id := before.toggleable(t)

	set := string(runOK(t, cmd("set", whID, "--disable", id, "--break-glass", "lkctl e2e")...))
	assert.Regexp(t, `(?m)^`+regexp.QuoteMeta(id)+`\s+false\s+warehouse\s+`, set)

	var disabled predefinedPolicies
	decodeJSON(t, runOK(t, cmd("list", whID, "-o", "json")...), &disabled)
	enabled, by := disabled.status(t, id)
	assert.False(t, enabled)
	assert.Equal(t, "warehouse", by)

	assert.Equal(t, "Predefined policies of warehouse "+whID+" reset\n", string(runOK(t, cmd("reset", whID)...)))

	var after predefinedPolicies
	decodeJSON(t, runOK(t, cmd("list", whID, "-o", "json")...), &after)
	enabled, by = after.status(t, id)
	assert.True(t, enabled)
	assert.Empty(t, by)
	assert.False(t, *after.Curated)
}

//go:build e2e_cli

package clie2e

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestCedarSchemaRequiresLakekeeperPlus runs a Lakekeeper+ command against
// the Lakekeeper compose stack. Its Lakekeeper+ counterparts are in
// plus_test.go.
func TestCedarSchemaRequiresLakekeeperPlus(t *testing.T) {
	requireBackend(t, BackendCompose)
	t.Parallel()

	_, stderr, _ := runFail(t, "cedar", "schema")

	assert.Regexp(t,
		`^Error: get cedar schema: requires Lakekeeper\+; server is Lakekeeper \S+\n$`,
		string(stderr))
}

// TestCedarProjectPoliciesRequiresLakekeeperPlus: its Lakekeeper+
// counterparts are in plus_cedar_policies_test.go.
func TestCedarProjectPoliciesRequiresLakekeeperPlus(t *testing.T) {
	requireBackend(t, BackendCompose)
	t.Parallel()

	_, stderr, _ := runFail(t, "cedar", "project-policies", "list")

	assert.Regexp(t,
		`^Error: list project cedar policies: requires Lakekeeper\+; server is Lakekeeper \S+\n$`,
		string(stderr))
}

// TestCedarPredefinedPoliciesRequiresLakekeeperPlus: its Lakekeeper+
// counterparts are in plus_cedar_predefined_test.go.
func TestCedarPredefinedPoliciesRequiresLakekeeperPlus(t *testing.T) {
	requireBackend(t, BackendCompose)
	t.Parallel()

	_, stderr, _ := runFail(t, "cedar", "project-predefined-policies", "list")

	assert.Regexp(t,
		`^Error: list project predefined cedar policies: requires Lakekeeper\+; server is Lakekeeper \S+\n$`,
		string(stderr))
}

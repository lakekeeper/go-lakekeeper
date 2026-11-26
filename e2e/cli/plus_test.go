//go:build e2e_cli && e2e_plus

package clie2e

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Lakekeeper+ tests carry the extra build tag e2e_plus and the name prefix
// TestPlus; e2e/plus/run.sh selects them, plus the read-only tests shared
// with the Lakekeeper suite, with a -run allow-list.

func TestPlusServerInfoReportsEditionAndAuthzBackend(t *testing.T) {
	requireBackend(t, BackendComposePlus, BackendComposePlusOpenFGA)
	t.Parallel()

	out := runOK(t, "server", "info", "--output", "json")

	var info struct {
		AuthzBackend      string  `json:"authz-backend"`
		EnterpriseVersion *string `json:"lakekeeper-enterprise-version"`
	}
	decodeJSON(t, out, &info)

	want := map[string]string{
		BackendComposePlus:        "cedar",
		BackendComposePlusOpenFGA: "openfga",
	}[activeBackend.Name()]
	assert.Equal(t, want, info.AuthzBackend)
	if assert.NotNil(t, info.EnterpriseVersion, "lakekeeper-enterprise-version") {
		assert.NotEmpty(t, *info.EnterpriseVersion)
	}
}

func TestPlusCedarSchemaPrintsSchema(t *testing.T) {
	requireBackend(t, BackendComposePlus)
	t.Parallel()

	var schema struct {
		Fragments map[string]string `json:"schema-fragments"`
	}
	decodeJSON(t, runOK(t, "cedar", "schema", "--output", "json"), &schema)
	require.NotEmpty(t, schema.Fragments, "schema-fragments")

	// text output: every fragment, under a comment naming its file
	text := string(runOK(t, "cedar", "schema"))
	for path, content := range schema.Fragments {
		assert.Contains(t, text, "// "+path+"\n")
		assert.Contains(t, text, content)
	}
}

func TestPlusCedarSchemaRequiresCedarBackend(t *testing.T) {
	requireBackend(t, BackendComposePlusOpenFGA)
	t.Parallel()

	_, stderr, _ := runFail(t, "cedar", "schema")

	assert.Regexp(t,
		`^Error: get cedar schema: requires the cedar authorization backend; server is Lakekeeper\+ \S+ with the openfga authorization backend\n$`,
		string(stderr))
}

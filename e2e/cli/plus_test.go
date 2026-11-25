//go:build e2e_cli && e2e_plus

package clie2e

import (
	"testing"

	"github.com/stretchr/testify/assert"
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

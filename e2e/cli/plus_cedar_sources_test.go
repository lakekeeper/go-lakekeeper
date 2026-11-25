//go:build e2e_cli && e2e_plus

package clie2e

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The compose stack's Cedar files, as the server sees them (docker-compose.yml
// mounts tests/cedar at /etc/lakekeeper/cedar).
const (
	cedarPolicyFile = "/etc/lakekeeper/cedar/policies.cedar"
	cedarEntityFile = "/etc/lakekeeper/cedar/entities.json"
	adminPrincipal  = "oidc~6deeb417-cdf9-4320-8a30-ddecea77a4bd"
)

type cedarSourceConfig struct {
	SourceType string `json:"source-type"`
	FilePath   string `json:"file-path"`
}

func TestPlusCedarPolicySources(t *testing.T) {
	requireBackend(t, BackendComposePlus)
	t.Parallel()

	var got struct {
		Sources []struct {
			SourceID    string            `json:"source-id"`
			Tier        string            `json:"tier"`
			PolicyCount int64             `json:"policy-count"`
			Config      cedarSourceConfig `json:"config"`
		} `json:"policy-sources"`
	}
	decodeJSON(t, runOK(t, "cedar", "policy-sources", "-o", "json"), &got)

	require.Len(t, got.Sources, 1)
	src := got.Sources[0]
	assert.Equal(t, cedarSourceConfig{SourceType: "file", FilePath: cedarPolicyFile}, src.Config)
	assert.Equal(t, "server", src.Tier)
	assert.Equal(t, int64(1), src.PolicyCount)

	assert.Regexp(t, `(?m)^SOURCE ID\s+TIER\s+POLICIES\s+SOURCE$`,
		string(runOK(t, "cedar", "policy-sources")))
	assert.Regexp(t, `(?m)^`+src.SourceID+`\s+server\s+1\s+file `+cedarPolicyFile+`$`,
		string(runOK(t, "cedar", "policy-sources")))
}

func TestPlusCedarEntitySources(t *testing.T) {
	requireBackend(t, BackendComposePlus)
	t.Parallel()

	var got struct {
		Sources []struct {
			SourceID    string            `json:"source-id"`
			EntityCount int64             `json:"entity-count"`
			Config      cedarSourceConfig `json:"config"`
		} `json:"entity-sources"`
	}
	decodeJSON(t, runOK(t, "cedar", "entity-sources", "-o", "json"), &got)

	require.Len(t, got.Sources, 1)
	src := got.Sources[0]
	assert.Equal(t, cedarSourceConfig{SourceType: "file", FilePath: cedarEntityFile}, src.Config)
	// The server counts its built-in entities (the actions) with the file's one.
	assert.Positive(t, src.EntityCount)

	assert.Regexp(t, `(?m)^`+src.SourceID+`\s+\d+\s+file `+cedarEntityFile+`$`,
		string(runOK(t, "cedar", "entity-sources")))
}

type cedarPolicies struct {
	Policies []struct {
		Effect    string         `json:"effect"`
		Principal map[string]any `json:"principal"`
	} `json:"policies"`
}

func TestPlusCedarServerPoliciesList(t *testing.T) {
	requireBackend(t, BackendComposePlus)
	t.Parallel()

	var sources struct {
		Sources []struct {
			SourceID string `json:"source-id"`
		} `json:"policy-sources"`
	}
	decodeJSON(t, runOK(t, "cedar", "policy-sources", "-o", "json"), &sources)
	require.Len(t, sources.Sources, 1)
	fileSource := sources.Sources[0].SourceID

	// Without a filter: the file's policy, and the predefined-policies tier.
	var all cedarPolicies
	decodeJSON(t, runOK(t, "cedar", "server-policies", "list", "-o", "json"), &all)
	require.NotEmpty(t, all.Policies)

	// Filtered to the file source: exactly the file's one policy.
	var fromFile cedarPolicies
	decodeJSON(t, runOK(t, "cedar", "server-policies", "list", "--source", fileSource, "-o", "json"), &fromFile)
	require.Len(t, fromFile.Policies, 1)
	assert.Equal(t, "permit", fromFile.Policies[0].Effect)
	assert.Equal(t, "==", fromFile.Policies[0].Principal["op"])

	text := string(runOK(t, "cedar", "server-policies", "list", "--source", fileSource))
	assert.Regexp(t, `(?m)^EFFECT\s+PRINCIPAL\s+ACTION\s+RESOURCE$`, text)
	assert.Regexp(t, `(?m)^permit\s+== Lakekeeper::User::"`+adminPrincipal+`"\s+All\s+All$`, text)

	var forbids cedarPolicies
	decodeJSON(t, runOK(t, "cedar", "server-policies", "list",
		"--source", fileSource, "--effect", "forbid", "-o", "json"), &forbids)
	assert.Empty(t, forbids.Policies)
}

type cedarEntities struct {
	Entities []struct {
		UID struct {
			Type string `json:"type"`
			ID   string `json:"id"`
		} `json:"uid"`
		Parents []struct {
			Type string `json:"type"`
			ID   string `json:"id"`
		} `json:"parents"`
	} `json:"entities"`
}

// has reports whether an entity with this type and id is in the result.
func (e cedarEntities) has(typ, id string) bool {
	for _, x := range e.Entities {
		if x.UID.Type == typ && x.UID.ID == id {
			return true
		}
	}
	return false
}

func TestPlusCedarResolveEntities(t *testing.T) {
	requireBackend(t, BackendComposePlus)
	t.Parallel()

	const defaultProject = "00000000-0000-0000-0000-000000000000"
	whID, _ := MustProvisionWarehouse(t)

	tests := []struct {
		name     string
		args     []string
		wantType string
		wantID   string
	}{
		{name: "server", args: []string{"server"}, wantType: "Lakekeeper::Server"},
		{name: "project", args: []string{"project", defaultProject}, wantType: "Lakekeeper::Project", wantID: defaultProject},
		{name: "warehouse", args: []string{"warehouse", whID}, wantType: "Lakekeeper::Warehouse", wantID: whID},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			args := append([]string{"cedar", "resolve-entities"}, tc.args...)

			var got cedarEntities
			decodeJSON(t, runOK(t, append(args, "-o", "json")...), &got)

			assert.True(t, got.has("Lakekeeper::User", adminPrincipal), "the caller is the principal: %+v", got)
			found := false
			for _, e := range got.Entities {
				if e.UID.Type == tc.wantType && (tc.wantID == "" || e.UID.ID == tc.wantID) {
					found = true
				}
			}
			assert.Truef(t, found, "entity %s %s in %+v", tc.wantType, tc.wantID, got)

			text := string(runOK(t, args...))
			assert.Regexp(t, `(?m)^TYPE\s+ID\s+PARENTS$`, text)
			assert.Regexp(t, `(?m)^Lakekeeper::User\s+`+adminPrincipal+`(\s|$)`, text)
		})
	}
}

//go:build integration_plus

package plus

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	managementv1 "github.com/lakekeeper/go-lakekeeper/pkg/apis/management/v1"
	"github.com/lakekeeper/go-lakekeeper/pkg/client"
)

// The real server's answers decode into the typed Cedar models: the
// sources' configs into their variant, policies and entities into
// CedarPolicy and CedarEntity.

func skipWithoutCedarBackend(t *testing.T) {
	t.Helper()
	if authzBackend != client.AuthzBackendCedar {
		t.Skipf("needs the cedar authorization backend, running %s", authzBackend)
	}
}

func TestCedarSources_DecodeIntoTypedConfigs(t *testing.T) {
	skipWithoutCedarBackend(t)

	policies, err := sharedClient.Cedar.PolicySources(t.Context())
	require.NoError(t, err)
	require.Len(t, policies.GetPolicySources(), 1)
	ps := policies.GetPolicySources()[0]
	require.NotNil(t, ps.Config.CedarSourceConfigFile)
	assert.Equal(t, "/etc/lakekeeper/cedar/policies.cedar", ps.Config.CedarSourceConfigFile.GetFilePath())
	assert.Equal(t, managementv1.CEDARPOLICYTIER_SERVER, ps.GetTier())

	entities, err := sharedClient.Cedar.EntitySources(t.Context())
	require.NoError(t, err)
	require.Len(t, entities.GetEntitySources(), 1)
	es := entities.GetEntitySources()[0]
	require.NotNil(t, es.Config.CedarSourceConfigFile)
	assert.Equal(t, "/etc/lakekeeper/cedar/entities.json", es.Config.CedarSourceConfigFile.GetFilePath())
}

func TestCedarServerPolicies_DecodeIntoTypedPolicies(t *testing.T) {
	skipWithoutCedarBackend(t)

	all, err := sharedClient.Cedar.ListServerPolicies(t.Context(), nil)
	require.NoError(t, err)
	require.NotEmpty(t, all.GetPolicies())
	for _, p := range all.GetPolicies() {
		assert.Contains(t, []managementv1.PolicyEffect{managementv1.POLICYEFFECT_PERMIT, managementv1.POLICYEFFECT_FORBID}, p.GetEffect())
		assert.NotEmpty(t, p.Principal.GetOp())
		assert.NotEmpty(t, p.Action.GetOp())
		assert.NotEmpty(t, p.Resource.GetOp())
	}

	sources, err := sharedClient.Cedar.PolicySources(t.Context())
	require.NoError(t, err)
	fromFile, err := sharedClient.Cedar.ListServerPolicies(t.Context(), &managementv1.ListCedarPoliciesQuery{
		SourceFilter: []string{sources.GetPolicySources()[0].GetSourceId()},
	})
	require.NoError(t, err)
	require.Len(t, fromFile.GetPolicies(), 1)
	p := fromFile.GetPolicies()[0]
	assert.Equal(t, "==", p.Principal.GetOp())
	assert.Equal(t, "Lakekeeper::User", p.Principal.Entity.GetType())
}

func TestCedarResolveEntities_DecodeIntoTypedEntities(t *testing.T) {
	skipWithoutCedarBackend(t)
	server := managementv1.CEDARRESOLVERESOURCESERVER_SERVER

	got, err := sharedClient.Cedar.ResolveEntities(t.Context(), managementv1.NewResolveCedarEntitiesRequest(
		managementv1.CedarResolveResourceServerAsCedarResolveResource(&server)))

	require.NoError(t, err)
	assert.Equal(t, int(got.GetEntityCount()), len(got.GetEntities()))
	types := map[string]int{}
	for _, e := range got.GetEntities() {
		types[e.Uid.GetType()]++
		assert.NotEmpty(t, e.Uid.GetId())
	}
	assert.Equal(t, 1, types["Lakekeeper::Server"])
	assert.Equal(t, 1, types["Lakekeeper::User"])
}

func TestCedarSources_OnOpenFGAIsUnsupported(t *testing.T) {
	if authzBackend != client.AuthzBackendOpenFGA {
		t.Skipf("needs the openfga authorization backend, running %s", authzBackend)
	}

	_, err := sharedClient.Cedar.PolicySources(t.Context())

	var unsupported *client.UnsupportedError
	require.ErrorAs(t, err, &unsupported)
	assert.Equal(t, client.ReasonAuthzBackend, unsupported.Reason)
}

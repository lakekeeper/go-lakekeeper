package client_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	managementv1 "github.com/lakekeeper/go-lakekeeper/pkg/apis/management/v1"
	"github.com/lakekeeper/go-lakekeeper/pkg/client"
	"github.com/lakekeeper/go-lakekeeper/pkg/testutil"
)

const (
	cedarPolicySourcesPath   = "GET /management/v1/permissions/cedar/server-policy-sources"
	cedarEntitySourcesPath   = "GET /management/v1/permissions/cedar/entity-sources"
	cedarServerPoliciesPath  = "POST /management/v1/permissions/cedar/server-policy-list"
	cedarResolveEntitiesPath = "POST /management/v1/permissions/cedar/resolve-entities"
)

func TestCedar_PolicySources_ReturnsTypedSources(t *testing.T) {
	t.Parallel()
	mux, c := testutil.ServerMux(t)
	testutil.Respond(t, mux, cedarPolicySourcesPath, http.StatusOK, `{"policy-sources":[
		{"source-id":"s1","policy-count":3,"tier":"server",
		 "config":{"source-type":"file","file-path":"/etc/lakekeeper/cedar/policies.cedar"}},
		{"source-id":"s2","policy-count":1,"tier":"server",
		 "config":{"source-type":"kubernetes-config-map","configmap-name":"cedar-policies"}}]}`)

	got, err := c.Cedar.PolicySources(t.Context())

	require.NoError(t, err)
	sources := got.GetPolicySources()
	require.Len(t, sources, 2)
	assert.Equal(t, "s1", sources[0].GetSourceId())
	assert.Equal(t, int64(3), sources[0].GetPolicyCount())
	assert.Equal(t, managementv1.CEDARPOLICYTIER_SERVER, sources[0].GetTier())
	require.NotNil(t, sources[0].Config.CedarSourceConfigFile)
	assert.Equal(t, "/etc/lakekeeper/cedar/policies.cedar", sources[0].Config.CedarSourceConfigFile.GetFilePath())
	require.NotNil(t, sources[1].Config.CedarSourceConfigKubernetesConfigMap)
	assert.Equal(t, "cedar-policies", sources[1].Config.CedarSourceConfigKubernetesConfigMap.GetConfigmapName())
}

func TestCedar_PolicySources_OnLakekeeperIsUnsupported(t *testing.T) {
	t.Parallel()
	mux, c := testutil.ServerMux(t)
	testutil.Respond(t, mux, cedarPolicySourcesPath, http.StatusNotFound, nil)
	testutil.Respond(t, mux, "GET /management/v1/info", http.StatusOK, lakekeeperInfo())

	_, err := c.Cedar.PolicySources(t.Context())

	require.ErrorIs(t, err, client.ErrUnsupported)
	require.EqualError(t, err, "list cedar policy sources: requires Lakekeeper+; server is Lakekeeper 0.15.1")
}

func TestCedar_EntitySources_ReturnsTypedSources(t *testing.T) {
	t.Parallel()
	mux, c := testutil.ServerMux(t)
	testutil.Respond(t, mux, cedarEntitySourcesPath, http.StatusOK, `{"entity-sources":[
		{"source-id":"e1","entity-count":225,
		 "config":{"source-type":"file","file-path":"/etc/lakekeeper/cedar/entities.json"}}]}`)

	got, err := c.Cedar.EntitySources(t.Context())

	require.NoError(t, err)
	sources := got.GetEntitySources()
	require.Len(t, sources, 1)
	assert.Equal(t, "e1", sources[0].GetSourceId())
	assert.Equal(t, int64(225), sources[0].GetEntityCount())
	require.NotNil(t, sources[0].Config.CedarSourceConfigFile)
	assert.Equal(t, "/etc/lakekeeper/cedar/entities.json", sources[0].Config.CedarSourceConfigFile.GetFilePath())
}

func TestCedar_EntitySources_OnOtherAuthzBackendIsUnsupported(t *testing.T) {
	t.Parallel()
	mux, c := testutil.ServerMux(t)
	testutil.Respond(t, mux, cedarEntitySourcesPath, http.StatusNotFound, nil)
	testutil.Respond(t, mux, "GET /management/v1/info", http.StatusOK, lakekeeperPlusInfo("openfga"))

	_, err := c.Cedar.EntitySources(t.Context())

	require.EqualError(t, err, "list cedar entity sources: requires the cedar authorization backend; "+
		"server is Lakekeeper+ 0.15.2 with the openfga authorization backend")
}

const serverPoliciesBody = `{"next-page-token":"p2","policies":[{"effect":"permit",
	"principal":{"op":"==","entity":{"type":"Lakekeeper::User","id":"oidc~admin"}},
	"action":{"op":"All"},"resource":{"op":"is","entity_type":"Lakekeeper::Table"},"conditions":[]}]}`

func TestCedar_ListServerPolicies_SendsQueryAndReturnsTypedPolicies(t *testing.T) {
	t.Parallel()
	mux, c := testutil.ServerMux(t)
	list := testutil.Respond(t, mux, cedarServerPoliciesPath, http.StatusOK, serverPoliciesBody)
	query := managementv1.ListCedarPoliciesQuery{SourceFilter: []string{"s1"}}
	query.SetEffectFilter(managementv1.CEDAREFFECTFILTER_PERMIT)

	got, err := c.Cedar.ListServerPolicies(t.Context(), &query)

	require.NoError(t, err)
	assert.Equal(t, "p2", got.GetNextPageToken())
	require.Len(t, got.GetPolicies(), 1)
	p := got.GetPolicies()[0]
	assert.Equal(t, managementv1.POLICYEFFECT_PERMIT, p.GetEffect())
	assert.Equal(t, "oidc~admin", p.Principal.Entity.GetId())
	assert.Equal(t, "Lakekeeper::Table", p.Resource.GetEntityType())
	var sent map[string]any
	require.NoError(t, json.Unmarshal(list.Last(t).Body, &sent))
	assert.Equal(t, map[string]any{"source-filter": []any{"s1"}, "effect-filter": "Permit"}, sent)
}

func TestCedar_ListServerPolicies_NilQuerySendsEmptyBody(t *testing.T) {
	t.Parallel()
	mux, c := testutil.ServerMux(t)
	list := testutil.Respond(t, mux, cedarServerPoliciesPath, http.StatusOK, `{"policies":[]}`)

	got, err := c.Cedar.ListServerPolicies(t.Context(), nil)

	require.NoError(t, err)
	assert.Empty(t, got.GetPolicies())
	assert.Empty(t, got.GetNextPageToken())
	assert.JSONEq(t, `{}`, string(list.Last(t).Body))
}

func TestCedar_ListServerPolicies_OnLakekeeperIsUnsupported(t *testing.T) {
	t.Parallel()
	mux, c := testutil.ServerMux(t)
	testutil.Respond(t, mux, cedarServerPoliciesPath, http.StatusNotFound, nil)
	testutil.Respond(t, mux, "GET /management/v1/info", http.StatusOK, lakekeeperInfo())

	_, err := c.Cedar.ListServerPolicies(t.Context(), nil)

	require.EqualError(t, err, "list server cedar policies: requires Lakekeeper+; server is Lakekeeper 0.15.1")
}

const resolvedEntitiesBody = `{"entity-count":2,"entities":[
	{"uid":{"type":"Lakekeeper::User","id":"oidc~admin"},"attrs":{"provider_id":"oidc"},"parents":[]},
	{"uid":{"type":"Lakekeeper::Warehouse","id":"` + warehouseA + `"},"attrs":{},
	 "parents":[{"type":"Lakekeeper::Project","id":"` + projectA + `"}]}]}`

func TestCedar_ResolveEntities_SendsResourceAndPrincipal(t *testing.T) {
	t.Parallel()
	mux, c := testutil.ServerMux(t)
	resolve := testutil.Respond(t, mux, cedarResolveEntitiesPath, http.StatusOK, resolvedEntitiesBody)
	req := managementv1.NewResolveCedarEntitiesRequest(managementv1.CedarResolveResourceWarehouseAsCedarResolveResource(
		managementv1.NewCedarResolveResourceWarehouse(*managementv1.NewCedarResolveResourceWarehouseWarehouse(warehouseA))))
	req.SetPrincipal(managementv1.UserOrRole{User: new("oidc~peter")})

	got, err := c.Cedar.ResolveEntities(t.Context(), req)

	require.NoError(t, err)
	assert.Equal(t, int32(2), got.GetEntityCount())
	require.Len(t, got.GetEntities(), 2)
	wh := got.GetEntities()[1]
	assert.Equal(t, managementv1.CedarEntityUid{Type: "Lakekeeper::Warehouse", Id: warehouseA}, wh.GetUid())
	assert.Equal(t, []managementv1.CedarEntityUid{{Type: "Lakekeeper::Project", Id: projectA}}, wh.GetParents())
	assert.JSONEq(t, `{"principal":{"user":"oidc~peter"},"resource":{"warehouse":{"warehouse-id":"`+warehouseA+`"}}}`,
		string(resolve.Last(t).Body))
}

func TestCedar_ResolveEntities_ServerResource(t *testing.T) {
	t.Parallel()
	mux, c := testutil.ServerMux(t)
	resolve := testutil.Respond(t, mux, cedarResolveEntitiesPath, http.StatusOK, `{"entity-count":0,"entities":[]}`)
	server := managementv1.CEDARRESOLVERESOURCESERVER_SERVER

	_, err := c.Cedar.ResolveEntities(t.Context(), managementv1.NewResolveCedarEntitiesRequest(
		managementv1.CedarResolveResourceServerAsCedarResolveResource(&server)))

	require.NoError(t, err)
	assert.JSONEq(t, `{"resource":"server"}`, string(resolve.Last(t).Body))
}

func TestCedar_ResolveEntities_NilRequestFailsBeforeAnyRequest(t *testing.T) {
	t.Parallel()
	mux, c := testutil.ServerMux(t)
	resolve := testutil.Respond(t, mux, cedarResolveEntitiesPath, http.StatusOK, `{"entity-count":0,"entities":[]}`)

	_, err := c.Cedar.ResolveEntities(t.Context(), nil)

	require.EqualError(t, err, "resolve cedar entities: request must not be nil")
	assert.Zero(t, resolve.Len())
}

func TestCedar_ResolveEntities_OnLakekeeperIsUnsupported(t *testing.T) {
	t.Parallel()
	mux, c := testutil.ServerMux(t)
	testutil.Respond(t, mux, cedarResolveEntitiesPath, http.StatusNotFound, nil)
	testutil.Respond(t, mux, "GET /management/v1/info", http.StatusOK, lakekeeperInfo())
	server := managementv1.CEDARRESOLVERESOURCESERVER_SERVER

	_, err := c.Cedar.ResolveEntities(t.Context(), managementv1.NewResolveCedarEntitiesRequest(
		managementv1.CedarResolveResourceServerAsCedarResolveResource(&server)))

	require.EqualError(t, err, "resolve cedar entities: requires Lakekeeper+; server is Lakekeeper 0.15.1")
}

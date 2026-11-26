package commands

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lakekeeper/go-lakekeeper/pkg/client"
	"github.com/lakekeeper/go-lakekeeper/pkg/testutil"
)

func TestCedarSourcesHelp_MarksLakekeeperPlus(t *testing.T) {
	t.Parallel()
	baseURL := noServer(t)

	help, err := runLkctl(t, baseURL, "cedar", "--help")

	require.NoError(t, err)
	assert.Regexp(t, `policy-sources\s+.*\(Lakekeeper\+\)`, help)
	assert.Regexp(t, `entity-sources\s+.*\(Lakekeeper\+\)`, help)
	assert.Regexp(t, `server-policies\s+.*\(Lakekeeper\+\)`, help)
	assert.Regexp(t, `resolve-entities\s+.*\(Lakekeeper\+\)`, help)
}

func TestCedarPolicySources_PrintsTable(t *testing.T) {
	t.Parallel()
	mux, baseURL := testutil.NewServer(t)
	testutil.Respond(t, mux, "GET /management/v1/permissions/cedar/server-policy-sources", http.StatusOK,
		`{"policy-sources":[
		{"source-id":"s1","policy-count":3,"tier":"server","config":{"source-type":"file","file-path":"/p.cedar"}},
		{"source-id":"s2","policy-count":1,"tier":"server","config":{"source-type":"kubernetes-config-map","configmap-name":"cm"}}]}`)

	out, err := runLkctl(t, baseURL, "cedar", "policy-sources")

	require.NoError(t, err)
	assert.Equal(t, ""+
		"SOURCE ID  TIER    POLICIES  SOURCE\n"+
		"s1         server  3         file /p.cedar\n"+
		"s2         server  1         configmap cm\n", out)
}

func TestCedarPolicySources_Empty(t *testing.T) {
	t.Parallel()
	mux, baseURL := testutil.NewServer(t)
	testutil.Respond(t, mux, "GET /management/v1/permissions/cedar/server-policy-sources", http.StatusOK,
		`{"policy-sources":[]}`)

	out, err := runLkctl(t, baseURL, "cedar", "policy-sources")

	require.NoError(t, err)
	assert.Equal(t, "No Cedar policy sources\n", out)
}

func TestCedarPolicySources_OnLakekeeperIsUnsupported(t *testing.T) {
	t.Parallel()
	mux, baseURL := testutil.NewServer(t)
	testutil.Respond(t, mux, "GET /management/v1/permissions/cedar/server-policy-sources", http.StatusNotFound, nil)
	testutil.Respond(t, mux, "GET /management/v1/info", http.StatusOK, serverTestInfoBody)

	_, err := runLkctl(t, baseURL, "cedar", "policy-sources")

	require.ErrorIs(t, err, client.ErrUnsupported)
	require.EqualError(t, err, "list cedar policy sources: requires Lakekeeper+; server is Lakekeeper 0.15.0")
}

func TestCedarEntitySources_PrintsTable(t *testing.T) {
	t.Parallel()
	mux, baseURL := testutil.NewServer(t)
	testutil.Respond(t, mux, "GET /management/v1/permissions/cedar/entity-sources", http.StatusOK,
		`{"entity-sources":[{"source-id":"e1","entity-count":225,"config":{"source-type":"file","file-path":"/e.json"}}]}`)

	out, err := runLkctl(t, baseURL, "cedar", "entity-sources")

	require.NoError(t, err)
	assert.Equal(t, ""+
		"SOURCE ID  ENTITIES  SOURCE\n"+
		"e1         225       file /e.json\n", out)
}

const serverPolicyListPath = "POST /management/v1/permissions/cedar/server-policy-list"

func TestCedarServerPoliciesList_PrintsScopesAndNextPage(t *testing.T) {
	t.Parallel()
	mux, baseURL := testutil.NewServer(t)
	testutil.Respond(t, mux, serverPolicyListPath, http.StatusOK, `{"next-page-token":"tok2","policies":[
		{"effect":"permit","principal":{"op":"==","entity":{"type":"Lakekeeper::User","id":"oidc~a"}},
		 "action":{"op":"All"},"resource":{"op":"All"},"conditions":[]},
		{"effect":"forbid","principal":{"op":"in","entity":{"type":"Lakekeeper::Role","id":"r"}},
		 "action":{"op":"in","entities":[{"type":"Lakekeeper::Action","id":"A"},{"type":"Lakekeeper::Action","id":"B"}]},
		 "resource":{"op":"is","entity_type":"Lakekeeper::Table","in":{"entity":{"type":"Lakekeeper::Warehouse","id":"w"}}},
		 "conditions":[{"kind":"when","body":{}}]},
		{"effect":"permit","principal":{"op":"==","slot":"?principal"},
		 "action":{"op":"==","entity":{"type":"Lakekeeper::Action","id":"C"}},"resource":{"op":"is","entity_type":"Lakekeeper::View"},
		 "conditions":[]}]}`)

	out, err := runLkctl(t, baseURL, "cedar", "server-policies", "list")

	require.NoError(t, err)
	assert.Equal(t, ""+
		"EFFECT  PRINCIPAL                      ACTION                                                 RESOURCE\n"+
		"permit  == Lakekeeper::User::\"oidc~a\"  All                                                    All\n"+
		"forbid  in Lakekeeper::Role::\"r\"       in [Lakekeeper::Action::\"A\", Lakekeeper::Action::\"B\"]  is Lakekeeper::Table in Lakekeeper::Warehouse::\"w\"\n"+
		"permit  == ?principal                  == Lakekeeper::Action::\"C\"                             is Lakekeeper::View\n"+
		"\nNext page token: tok2\n", out)
}

func TestCedarServerPoliciesList_SendsFilters(t *testing.T) {
	t.Parallel()
	mux, baseURL := testutil.NewServer(t)
	list := testutil.Respond(t, mux, serverPolicyListPath, http.StatusOK, `{"policies":[]}`)

	out, err := runLkctl(t, baseURL, "cedar", "server-policies", "list",
		"--source", "s1", "--source", "s2", "--effect", "forbid", "--page-size", "10", "--page-token", "tok")

	require.NoError(t, err)
	assert.Equal(t, "No Cedar policies\n", out)
	var sent map[string]any
	require.NoError(t, json.Unmarshal(list.Last(t).Body, &sent))
	assert.Equal(t, map[string]any{
		"source-filter": []any{"s1", "s2"},
		"effect-filter": "Forbid",
		"page-size":     float64(10),
		"page-token":    "tok",
	}, sent)
}

func TestCedarServerPoliciesList_WithoutFlagsSendsEmptyQuery(t *testing.T) {
	t.Parallel()
	mux, baseURL := testutil.NewServer(t)
	list := testutil.Respond(t, mux, serverPolicyListPath, http.StatusOK, `{"policies":[]}`)

	_, err := runLkctl(t, baseURL, "cedar", "server-policies", "list")

	require.NoError(t, err)
	assert.JSONEq(t, `{}`, string(list.Last(t).Body))
}

func TestCedarServerPoliciesList_RejectsUnknownEffect(t *testing.T) {
	t.Parallel()

	_, err := runLkctl(t, noServer(t), "cedar", "server-policies", "list", "--effect", "maybe")

	require.EqualError(t, err, `invalid --effect "maybe": want permit or forbid`)
}

const resolveEntitiesPath = "POST /management/v1/permissions/cedar/resolve-entities"

const resolvedBody = `{"entity-count":4,"entities":[
	{"uid":{"type":"Lakekeeper::User","id":"oidc~a"},"attrs":{},"parents":[{"type":"Lakekeeper::Role","id":"p/r"}]},
	{"uid":{"type":"Lakekeeper::Warehouse","id":"w"},"attrs":{},
	 "parents":[{"type":"Lakekeeper::Project","id":"p"},{"type":"Lakekeeper::Server","id":"s"}]},
	{"uid":{"type":"Lakekeeper::Action","id":"A"},"attrs":{},"parents":[]},
	{"uid":{"type":"Lakekeeper::Action","id":"B"},"attrs":{},"parents":[]}]}`

func TestCedarResolveEntities_PrintsEntitiesWithoutActions(t *testing.T) {
	t.Parallel()
	mux, baseURL := testutil.NewServer(t)
	resolve := testutil.Respond(t, mux, resolveEntitiesPath, http.StatusOK, resolvedBody)

	out, err := runLkctl(t, baseURL, "cedar", "resolve-entities", "warehouse", "w")

	require.NoError(t, err)
	assert.Equal(t, ""+
		"TYPE                   ID      PARENTS\n"+
		"Lakekeeper::User       oidc~a  Lakekeeper::Role::\"p/r\"\n"+
		"Lakekeeper::Warehouse  w       Lakekeeper::Project::\"p\", Lakekeeper::Server::\"s\"\n"+
		"\nEntities: 4 (2 Lakekeeper::Action entities not shown; -o wide shows them)\n", out)
	assert.JSONEq(t, `{"resource":{"warehouse":{"warehouse-id":"w"}}}`, string(resolve.Last(t).Body))
}

func TestCedarResolveEntities_WideShowsActions(t *testing.T) {
	t.Parallel()
	mux, baseURL := testutil.NewServer(t)
	testutil.Respond(t, mux, resolveEntitiesPath, http.StatusOK, resolvedBody)

	out, err := runLkctl(t, baseURL, "cedar", "resolve-entities", "server", "-o", "wide")

	require.NoError(t, err)
	assert.Contains(t, out, "Lakekeeper::Action     A")
	assert.Contains(t, out, "\nEntities: 4\n")
}

func TestCedarResolveEntities_SendsResourceAndPrincipal(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		args []string
		want string
	}{
		{"server", []string{"server"}, `{"resource":"server"}`},
		{"project with id", []string{"project", "p"}, `{"resource":{"project":{"project-id":"p"}}}`},
		{"project without id", []string{"project"}, `{"resource":{"project":{}}}`},
		{
			"warehouse for a user",
			[]string{"warehouse", "w", "--user", "oidc~u"},
			`{"principal":{"user":"oidc~u"},"resource":{"warehouse":{"warehouse-id":"w"}}}`,
		},
		{
			"server for a role",
			[]string{"server", "--role", "r1"},
			`{"principal":{"role":"r1"},"resource":"server"}`,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			mux, baseURL := testutil.NewServer(t)
			resolve := testutil.Respond(t, mux, resolveEntitiesPath, http.StatusOK, `{"entity-count":0,"entities":[]}`)

			_, err := runLkctl(t, baseURL, append([]string{"cedar", "resolve-entities"}, tc.args...)...)

			require.NoError(t, err)
			assert.JSONEq(t, tc.want, string(resolve.Last(t).Body))
		})
	}
}

func TestCedarResolveEntities_RejectsBadArguments(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		args []string
		want string
	}{
		{"no resource", nil, "requires a resource: server, project [PROJECTID] or warehouse WAREHOUSEID"},
		{"unknown resource", []string{"table", "t"}, `unknown resource "table": want server, project or warehouse`},
		{"server with id", []string{"server", "x"}, "server takes no id"},
		{"warehouse without id", []string{"warehouse"}, "warehouse requires a WAREHOUSEID"},
		{
			"user and role",
			[]string{"server", "--user", "u", "--role", "r"},
			"if any flags in the group [user role] are set none of the others can be; [role user] were all set",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := runLkctl(t, noServer(t), append([]string{"cedar", "resolve-entities"}, tc.args...)...)
			require.EqualError(t, err, tc.want)
		})
	}
}

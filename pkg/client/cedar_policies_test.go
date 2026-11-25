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
	cedarProjectPoliciesPath   = "/management/v1/permissions/cedar/project/policies"
	cedarWarehousePoliciesPath = "/management/v1/permissions/cedar/warehouse/" + warehouseA + "/policies"
	cedarBreakGlassStatusPath  = "GET /management/v1/permissions/cedar/break-glass-status"

	warehouseA = "0198618c-5be8-7a82-a0b9-1076c9dd12f0"
)

const scopePoliciesBody = `{"can-write":true,"scope-version":7,"policies":[
	{"id":"p1","name":"readers","created-at":"2026-11-25T10:00:00Z","policy":{"effect":"permit"}}]}`

const scopePolicyBody = `{"id":"p1","name":"readers","created-at":"2026-11-25T10:00:00Z","policy":{"effect":"permit"}}`

const applyScopePoliciesBody = `{"scope-version":8,"changed":[{"id":"p1","name":"readers","op":"create"}]}`

func applyReadersRequest() *managementv1.ApplyScopePoliciesRequest {
	req := managementv1.NewApplyScopePoliciesRequest()
	req.SetWrites([]managementv1.ScopePolicyUpsert{
		{Name: *managementv1.NewNullableString(new("readers")), Policy: map[string]any{"effect": "permit"}},
	})
	return req
}

func TestCedar_ListProjectPolicies_SendsProjectAndBreakGlass(t *testing.T) {
	t.Parallel()
	mux, c := testutil.ServerMux(t)
	list := testutil.Respond(t, mux, "GET "+cedarProjectPoliciesPath, http.StatusOK, scopePoliciesBody)

	got, err := c.Cedar.ListProjectPolicies(t.Context(),
		client.WithProject(projectA), client.WithBreakGlass("incident 42"))

	require.NoError(t, err)
	assert.Equal(t, int64(7), got.GetScopeVersion())
	assert.True(t, got.GetCanWrite())
	require.Len(t, got.GetPolicies(), 1)
	assert.Equal(t, "readers", got.GetPolicies()[0].GetName())
	sent := list.Last(t)
	assert.Equal(t, projectA, sent.Header.Get("x-project-id"))
	assert.Equal(t, "incident 42", sent.Header.Get("x-break-glass"))
}

func TestCedar_ListProjectPolicies_WithoutOptionsSendsNeitherHeader(t *testing.T) {
	t.Parallel()
	mux, c := testutil.ServerMux(t)
	list := testutil.Respond(t, mux, "GET "+cedarProjectPoliciesPath, http.StatusOK, scopePoliciesBody)

	_, err := c.Cedar.ListProjectPolicies(t.Context())

	require.NoError(t, err)
	sent := list.Last(t)
	assert.NotContains(t, sent.Header, "X-Project-Id")
	assert.NotContains(t, sent.Header, "X-Break-Glass")
}

func TestCedar_ListProjectPolicies_OnLakekeeperIsUnsupported(t *testing.T) {
	t.Parallel()
	mux, c := testutil.ServerMux(t)
	testutil.Respond(t, mux, "GET "+cedarProjectPoliciesPath, http.StatusNotFound, nil)
	testutil.Respond(t, mux, "GET /management/v1/info", http.StatusOK, lakekeeperInfo())

	_, err := c.Cedar.ListProjectPolicies(t.Context())

	require.ErrorIs(t, err, client.ErrUnsupported)
	require.EqualError(t, err, "list project cedar policies: requires Lakekeeper+; server is Lakekeeper 0.15.1")
}

func TestCedar_GetProjectPolicy_ReturnsEntryByName(t *testing.T) {
	t.Parallel()
	mux, c := testutil.ServerMux(t)
	get := testutil.Respond(t, mux, "GET "+cedarProjectPoliciesPath+"/{name}", http.StatusOK, scopePolicyBody)

	got, err := c.Cedar.GetProjectPolicy(t.Context(), "readers", client.WithProject(projectA))

	require.NoError(t, err)
	assert.Equal(t, "p1", got.GetId())
	assert.Equal(t, map[string]any{"effect": "permit"}, got.GetPolicy())
	sent := get.Last(t)
	assert.Equal(t, cedarProjectPoliciesPath+"/readers", sent.Path)
	assert.Equal(t, projectA, sent.Header.Get("x-project-id"))
}

func TestCedar_ApplyProjectPolicies_SendsRequestProjectAndBreakGlass(t *testing.T) {
	t.Parallel()
	mux, c := testutil.ServerMux(t)
	apply := testutil.Respond(t, mux, "POST "+cedarProjectPoliciesPath, http.StatusOK, applyScopePoliciesBody)

	got, err := c.Cedar.ApplyProjectPolicies(t.Context(), applyReadersRequest(),
		client.WithProject(projectA), client.WithBreakGlass("incident 42"))

	require.NoError(t, err)
	assert.Equal(t, int64(8), got.GetScopeVersion())
	assert.Equal(t, []managementv1.ScopePolicyChange{{Id: "p1", Name: "readers", Op: managementv1.SCOPEPOLICYCHANGEOP_CREATE}},
		got.GetChanged())
	sent := apply.Last(t)
	assert.Equal(t, projectA, sent.Header.Get("x-project-id"))
	assert.Equal(t, "incident 42", sent.Header.Get("x-break-glass"))
	assert.JSONEq(t, `{"replace":false,"writes":[{"name":"readers","policy":{"effect":"permit"}}]}`, string(sent.Body))
}

func TestCedar_ApplyProjectPolicies_NilRequestFailsWithoutRequest(t *testing.T) {
	t.Parallel()
	_, c := testutil.ServerMux(t)

	_, err := c.Cedar.ApplyProjectPolicies(t.Context(), nil)

	require.EqualError(t, err, "apply project cedar policies: request must not be nil")
}

func TestCedar_ListWarehousePolicies_SendsBreakGlass(t *testing.T) {
	t.Parallel()
	mux, c := testutil.ServerMux(t)
	list := testutil.Respond(t, mux, "GET "+cedarWarehousePoliciesPath, http.StatusOK, scopePoliciesBody)

	got, err := c.Cedar.ListWarehousePolicies(t.Context(), warehouseA, client.WithBreakGlass("incident 42"))

	require.NoError(t, err)
	assert.Len(t, got.GetPolicies(), 1)
	assert.Equal(t, "incident 42", list.Last(t).Header.Get("x-break-glass"))
}

func TestCedar_GetWarehousePolicy_ReturnsEntryByName(t *testing.T) {
	t.Parallel()
	mux, c := testutil.ServerMux(t)
	get := testutil.Respond(t, mux, "GET "+cedarWarehousePoliciesPath+"/{name}", http.StatusOK, scopePolicyBody)

	got, err := c.Cedar.GetWarehousePolicy(t.Context(), warehouseA, "readers")

	require.NoError(t, err)
	assert.Equal(t, "readers", got.GetName())
	assert.Equal(t, cedarWarehousePoliciesPath+"/readers", get.Last(t).Path)
}

func TestCedar_ApplyWarehousePolicies_SendsRequestAndBreakGlass(t *testing.T) {
	t.Parallel()
	mux, c := testutil.ServerMux(t)
	apply := testutil.Respond(t, mux, "POST "+cedarWarehousePoliciesPath, http.StatusOK, applyScopePoliciesBody)

	got, err := c.Cedar.ApplyWarehousePolicies(t.Context(), warehouseA, applyReadersRequest(),
		client.WithBreakGlass("incident 42"))

	require.NoError(t, err)
	assert.Len(t, got.GetChanged(), 1)
	sent := apply.Last(t)
	assert.Equal(t, "incident 42", sent.Header.Get("x-break-glass"))
	var body map[string]any
	require.NoError(t, json.Unmarshal(sent.Body, &body))
	assert.Contains(t, body, "writes")
}

// A 404 on a capable server is the warehouse missing, not Cedar: it stays
// an *APIError.
func TestCedar_ListWarehousePolicies_MissingWarehouseStaysNotFound(t *testing.T) {
	t.Parallel()
	mux, c := testutil.ServerMux(t)
	testutil.Respond(t, mux, "GET "+cedarWarehousePoliciesPath, http.StatusNotFound,
		`{"error":{"code":404,"message":"Warehouse not found","type":"WarehouseNotFound","stack":[]}}`)
	testutil.Respond(t, mux, "GET /management/v1/info", http.StatusOK, lakekeeperPlusInfo("cedar"))

	_, err := c.Cedar.ListWarehousePolicies(t.Context(), warehouseA)

	require.NotErrorIs(t, err, client.ErrUnsupported)
	var apiErr *client.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, http.StatusNotFound, apiErr.StatusCode)
}

func TestCedar_BreakGlassStatus_SendsProjectAsQueryParameter(t *testing.T) {
	t.Parallel()
	mux, c := testutil.ServerMux(t)
	status := testutil.Respond(t, mux, cedarBreakGlassStatusPath, http.StatusOK, `{"break-glass-available":true}`)

	got, err := c.Cedar.BreakGlassStatus(t.Context(), client.WithProject(projectA))

	require.NoError(t, err)
	assert.True(t, got.GetBreakGlassAvailable())
	sent := status.Last(t)
	assert.Equal(t, projectA, sent.Query.Get("project-id"))
	assert.NotContains(t, sent.Header, "X-Project-Id")
}

func TestCedar_BreakGlassStatus_WithoutProjectSendsNone(t *testing.T) {
	t.Parallel()
	mux, c := testutil.ServerMux(t)
	status := testutil.Respond(t, mux, cedarBreakGlassStatusPath, http.StatusOK, `{"break-glass-available":false}`)

	_, err := c.Cedar.BreakGlassStatus(t.Context())

	require.NoError(t, err)
	assert.NotContains(t, status.Last(t).Query, "project-id")
}

// Options an operation cannot send fail before any request, so a
// break-glass reason or a project is never silently dropped.
func TestCallOptions_NotAcceptedFailWithoutRequest(t *testing.T) {
	t.Parallel()
	_, c := testutil.ServerMux(t) // any request is answered 404 by the empty mux

	cases := map[string]func() error{
		"get project policy with break-glass": func() error {
			_, err := c.Cedar.GetProjectPolicy(t.Context(), "readers", client.WithBreakGlass("r"))
			return err
		},
		"break-glass status with break-glass": func() error {
			_, err := c.Cedar.BreakGlassStatus(t.Context(), client.WithBreakGlass("r"))
			return err
		},
		"list warehouse policies with project": func() error {
			_, err := c.Cedar.ListWarehousePolicies(t.Context(), warehouseA, client.WithProject(projectA))
			return err
		},
		"apply warehouse policies with project": func() error {
			_, err := c.Cedar.ApplyWarehousePolicies(t.Context(), warehouseA, applyReadersRequest(), client.WithProject(projectA))
			return err
		},
		"list roles with break-glass": func() error {
			_, err := c.Roles.List(t.Context(), nil, client.WithBreakGlass("r"))
			return err
		},
	}
	for name, call := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			require.ErrorIs(t, call(), client.ErrCallOptionNotAccepted)
		})
	}
}

func TestWithBreakGlass_EmptyReasonFailsWithoutRequest(t *testing.T) {
	t.Parallel()
	_, c := testutil.ServerMux(t)

	_, err := c.Cedar.ListProjectPolicies(t.Context(), client.WithBreakGlass(""))

	require.ErrorIs(t, err, client.ErrEmptyBreakGlassReason)
	require.EqualError(t, err, "list project cedar policies: break-glass reason must not be empty")
}

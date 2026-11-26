package client_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	managementv1 "github.com/lakekeeper/go-lakekeeper/pkg/apis/management/v1"
	"github.com/lakekeeper/go-lakekeeper/pkg/client"
	"github.com/lakekeeper/go-lakekeeper/pkg/testutil"
)

const (
	cedarProjectPredefinedPath   = "/management/v1/permissions/cedar/project/predefined-policies"
	cedarWarehousePredefinedPath = "/management/v1/permissions/cedar/warehouse/" + warehouseA + "/predefined-policies"
)

const predefinedPoliciesBody = `{"curated":true,"predefined-policies-enabled":true,"policies":[
	{"id":"predefined-grants-table-select","description":"Grants allow table select","enabled":false,
	 "can-toggle":true,"disabled-by":"project"}]}`

func TestCedar_ListProjectPredefinedPolicies_SendsProjectBreakGlassAndIncludePolicy(t *testing.T) {
	t.Parallel()
	mux, c := testutil.ServerMux(t)
	list := testutil.Respond(t, mux, "GET "+cedarProjectPredefinedPath, http.StatusOK, predefinedPoliciesBody)

	got, err := c.Cedar.ListProjectPredefinedPolicies(t.Context(),
		&client.PredefinedPoliciesOptions{IncludePolicy: true},
		client.WithProject(projectA), client.WithBreakGlass("incident 42"))

	require.NoError(t, err)
	assert.True(t, got.GetCurated())
	require.Len(t, got.GetPolicies(), 1)
	assert.Equal(t, "predefined-grants-table-select", got.GetPolicies()[0].GetId())
	assert.Equal(t, managementv1.PREDEFINEDPOLICYDISABLEDBY_PROJECT, got.GetPolicies()[0].GetDisabledBy())
	sent := list.Last(t)
	assert.Equal(t, projectA, sent.Header.Get("x-project-id"))
	assert.Equal(t, "incident 42", sent.Header.Get("x-break-glass"))
	assert.Equal(t, "true", sent.Query.Get("include-policy"))
}

func TestCedar_ListProjectPredefinedPolicies_NilOptionsSendNoQuery(t *testing.T) {
	t.Parallel()
	mux, c := testutil.ServerMux(t)
	list := testutil.Respond(t, mux, "GET "+cedarProjectPredefinedPath, http.StatusOK, predefinedPoliciesBody)

	_, err := c.Cedar.ListProjectPredefinedPolicies(t.Context(), nil)

	require.NoError(t, err)
	sent := list.Last(t)
	assert.Empty(t, sent.Query)
	assert.NotContains(t, sent.Header, "X-Project-Id")
}

func TestCedar_ListProjectPredefinedPolicies_OnLakekeeperIsUnsupported(t *testing.T) {
	t.Parallel()
	mux, c := testutil.ServerMux(t)
	testutil.Respond(t, mux, "GET "+cedarProjectPredefinedPath, http.StatusNotFound, nil)
	testutil.Respond(t, mux, "GET /management/v1/info", http.StatusOK, lakekeeperInfo())

	_, err := c.Cedar.ListProjectPredefinedPolicies(t.Context(), nil)

	require.ErrorIs(t, err, client.ErrUnsupported)
	require.EqualError(t, err, "list project predefined cedar policies: requires Lakekeeper+; server is Lakekeeper 0.15.1")
}

func TestCedar_SetProjectPredefinedPolicies_SendsChangesAndOptions(t *testing.T) {
	t.Parallel()
	mux, c := testutil.ServerMux(t)
	set := testutil.Respond(t, mux, "POST "+cedarProjectPredefinedPath, http.StatusOK, predefinedPoliciesBody)

	got, err := c.Cedar.SetProjectPredefinedPolicies(t.Context(),
		[]managementv1.PredefinedPolicyChange{{Id: "predefined-grants-table-select", Enabled: false}},
		&client.PredefinedPoliciesOptions{IncludePolicy: true},
		client.WithProject(projectA), client.WithBreakGlass("incident 42"))

	require.NoError(t, err)
	assert.Len(t, got.GetPolicies(), 1)
	sent := set.Last(t)
	assert.Equal(t, projectA, sent.Header.Get("x-project-id"))
	assert.Equal(t, "incident 42", sent.Header.Get("x-break-glass"))
	assert.Equal(t, "true", sent.Query.Get("include-policy"))
	assert.JSONEq(t, `{"changes":[{"id":"predefined-grants-table-select","enabled":false}]}`, string(sent.Body))
}

func TestCedar_SetPredefinedPolicies_NoChangesFailWithoutRequest(t *testing.T) {
	t.Parallel()
	_, c := testutil.ServerMux(t)

	_, err := c.Cedar.SetProjectPredefinedPolicies(t.Context(), nil, nil)
	require.EqualError(t, err, "set project predefined cedar policies: at least one change is required")

	_, err = c.Cedar.SetWarehousePredefinedPolicies(t.Context(), warehouseA, []managementv1.PredefinedPolicyChange{}, nil)
	require.EqualError(t, err, "set warehouse predefined cedar policies: at least one change is required")
}

func TestCedar_ResetProjectPredefinedPolicies_SendsProject(t *testing.T) {
	t.Parallel()
	mux, c := testutil.ServerMux(t)
	reset := testutil.Respond(t, mux, "DELETE "+cedarProjectPredefinedPath, http.StatusNoContent, nil)

	err := c.Cedar.ResetProjectPredefinedPolicies(t.Context(), client.WithProject(projectA))

	require.NoError(t, err)
	assert.Equal(t, projectA, reset.Last(t).Header.Get("x-project-id"))
}

func TestCedar_ResetProjectPredefinedPolicies_OnLakekeeperIsUnsupported(t *testing.T) {
	t.Parallel()
	mux, c := testutil.ServerMux(t)
	testutil.Respond(t, mux, "DELETE "+cedarProjectPredefinedPath, http.StatusNotFound, nil)
	testutil.Respond(t, mux, "GET /management/v1/info", http.StatusOK, lakekeeperInfo())

	err := c.Cedar.ResetProjectPredefinedPolicies(t.Context())

	require.ErrorIs(t, err, client.ErrUnsupported)
	require.EqualError(t, err, "reset project predefined cedar policies: requires Lakekeeper+; server is Lakekeeper 0.15.1")
}

func TestCedar_ListWarehousePredefinedPolicies_SendsBreakGlassAndIncludePolicy(t *testing.T) {
	t.Parallel()
	mux, c := testutil.ServerMux(t)
	list := testutil.Respond(t, mux, "GET "+cedarWarehousePredefinedPath, http.StatusOK, predefinedPoliciesBody)

	got, err := c.Cedar.ListWarehousePredefinedPolicies(t.Context(), warehouseA,
		&client.PredefinedPoliciesOptions{IncludePolicy: true}, client.WithBreakGlass("incident 42"))

	require.NoError(t, err)
	assert.Len(t, got.GetPolicies(), 1)
	sent := list.Last(t)
	assert.Equal(t, "incident 42", sent.Header.Get("x-break-glass"))
	assert.Equal(t, "true", sent.Query.Get("include-policy"))
}

func TestCedar_SetWarehousePredefinedPolicies_SendsChangesAndBreakGlass(t *testing.T) {
	t.Parallel()
	mux, c := testutil.ServerMux(t)
	set := testutil.Respond(t, mux, "POST "+cedarWarehousePredefinedPath, http.StatusOK, predefinedPoliciesBody)

	_, err := c.Cedar.SetWarehousePredefinedPolicies(t.Context(), warehouseA,
		[]managementv1.PredefinedPolicyChange{{Id: "predefined-grants-table-select", Enabled: true}},
		nil, client.WithBreakGlass("incident 42"))

	require.NoError(t, err)
	sent := set.Last(t)
	assert.Equal(t, "incident 42", sent.Header.Get("x-break-glass"))
	assert.Empty(t, sent.Query)
	assert.JSONEq(t, `{"changes":[{"id":"predefined-grants-table-select","enabled":true}]}`, string(sent.Body))
}

func TestCedar_ResetWarehousePredefinedPolicies_SendsDelete(t *testing.T) {
	t.Parallel()
	mux, c := testutil.ServerMux(t)
	reset := testutil.Respond(t, mux, "DELETE "+cedarWarehousePredefinedPath, http.StatusNoContent, nil)

	err := c.Cedar.ResetWarehousePredefinedPolicies(t.Context(), warehouseA)

	require.NoError(t, err)
	assert.Equal(t, cedarWarehousePredefinedPath, reset.Last(t).Path)
}

// Options an operation cannot send fail before any request.
func TestCedar_PredefinedPolicies_NotAcceptedOptionsFailWithoutRequest(t *testing.T) {
	t.Parallel()
	_, c := testutil.ServerMux(t)

	cases := map[string]func() error{
		"reset project with break-glass": func() error {
			return c.Cedar.ResetProjectPredefinedPolicies(t.Context(), client.WithBreakGlass("r"))
		},
		"list warehouse with project": func() error {
			_, err := c.Cedar.ListWarehousePredefinedPolicies(t.Context(), warehouseA, nil, client.WithProject(projectA))
			return err
		},
		"set warehouse with project": func() error {
			_, err := c.Cedar.SetWarehousePredefinedPolicies(t.Context(), warehouseA,
				[]managementv1.PredefinedPolicyChange{{Id: "predefined-x", Enabled: true}}, nil, client.WithProject(projectA))
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

package commands

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lakekeeper/go-lakekeeper/pkg/testutil"
)

const (
	cedarProjectPredefinedPath   = "/management/v1/permissions/cedar/project/predefined-policies"
	cedarWarehousePredefinedPath = "/management/v1/permissions/cedar/warehouse/" + testWarehouseID + "/predefined-policies"

	predefinedPoliciesBody = `{"curated":true,"predefined-policies-enabled":true,"policies":[
		{"id":"predefined-grants-table-select","description":"Grants allow table select","enabled":false,
		 "can-toggle":true,"disabled-by":"project","updated-at":"2026-11-25T10:00:00Z","updated-by":"oidc~admin"},
		{"id":"predefined-owner-all","description":"Owners may do everything","enabled":true,"can-toggle":false},
		{"id":"predefined-old","description":"Retired","enabled":false,"can-toggle":true,"retired":true}]}`
)

func TestCedarPredefined_ListText(t *testing.T) {
	t.Parallel()
	mux, baseURL := testutil.NewServer(t)
	testutil.Respond(t, mux, "GET "+cedarProjectPredefinedPath, http.StatusOK, predefinedPoliciesBody)

	out, err := runLkctl(t, baseURL, "cedar", "project-predefined-policies", "list")

	require.NoError(t, err)
	assert.Equal(t, `ID                              ENABLED  DISABLED BY  CAN TOGGLE
predefined-grants-table-select  false    project      true
predefined-owner-all            true     -            false
predefined-old                  false    (retired)    true

Curated: true
Predefined policies enabled: true
`, out)
}

func TestCedarPredefined_ListWide(t *testing.T) {
	t.Parallel()
	mux, baseURL := testutil.NewServer(t)
	testutil.Respond(t, mux, "GET "+cedarWarehousePredefinedPath, http.StatusOK, predefinedPoliciesBody)

	out, err := runLkctl(t, baseURL, "cedar", "warehouse-predefined-policies", "list", testWarehouseID, "-o", "wide")

	require.NoError(t, err)
	assert.Contains(t, out, "ID                              ENABLED  DISABLED BY  CAN TOGGLE  UPDATED AT            UPDATED BY  DESCRIPTION\n")
	assert.Contains(t, out, "predefined-grants-table-select  false    project      true        2026-11-25T10:00:00Z  oidc~admin  Grants allow table select\n")
}

func TestCedarPredefined_ListEmpty(t *testing.T) {
	t.Parallel()
	mux, baseURL := testutil.NewServer(t)
	testutil.Respond(t, mux, "GET "+cedarProjectPredefinedPath, http.StatusOK,
		`{"curated":false,"predefined-policies-enabled":false,"policies":[]}`)

	out, err := runLkctl(t, baseURL, "cedar", "project-predefined-policies", "list")

	require.NoError(t, err)
	assert.Equal(t, "No predefined policies\n\nCurated: false\nPredefined policies enabled: false\n", out)
}

func TestCedarPredefined_ListSendsProjectBreakGlassAndIncludePolicy(t *testing.T) {
	t.Parallel()
	mux, baseURL := testutil.NewServer(t)
	list := testutil.Respond(t, mux, "GET "+cedarProjectPredefinedPath, http.StatusOK, predefinedPoliciesBody)

	_, err := runLkctl(t, baseURL, "cedar", "project-predefined-policies", "list",
		"-p", "proj-1", "--break-glass", "incident 42", "--include-policy", "-o", "json")

	require.NoError(t, err)
	sent := list.Last(t)
	assert.Equal(t, "proj-1", sent.Header.Get("x-project-id"))
	assert.Equal(t, "incident 42", sent.Header.Get("x-break-glass"))
	assert.Equal(t, "true", sent.Query.Get("include-policy"))
}

func TestCedarPredefined_SetSendsChangesAndPrintsListing(t *testing.T) {
	t.Parallel()
	mux, baseURL := testutil.NewServer(t)
	set := testutil.Respond(t, mux, "POST "+cedarWarehousePredefinedPath, http.StatusOK, predefinedPoliciesBody)

	out, err := runLkctl(t, baseURL, "cedar", "warehouse-predefined-policies", "set", testWarehouseID,
		"--disable", "predefined-grants-table-select", "--enable", "predefined-a,predefined-b",
		"--break-glass", "incident 42")

	require.NoError(t, err)
	assert.Contains(t, out, "predefined-grants-table-select  false    project      true\n")
	assert.Contains(t, out, "Curated: true\n")
	sent := set.Last(t)
	assert.Equal(t, "incident 42", sent.Header.Get("x-break-glass"))
	assert.JSONEq(t, `{"changes":[
		{"id":"predefined-a","enabled":true},
		{"id":"predefined-b","enabled":true},
		{"id":"predefined-grants-table-select","enabled":false}]}`, string(sent.Body))
}

func TestCedarPredefined_SetWithoutChangesFails(t *testing.T) {
	t.Parallel()
	baseURL := noServer(t)

	_, err := runLkctl(t, baseURL, "cedar", "project-predefined-policies", "set")

	require.EqualError(t, err, "name at least one policy with --enable or --disable")
}

func TestCedarPredefined_Reset(t *testing.T) {
	t.Parallel()
	mux, baseURL := testutil.NewServer(t)
	projectReset := testutil.Respond(t, mux, "DELETE "+cedarProjectPredefinedPath, http.StatusNoContent, nil)
	testutil.Respond(t, mux, "DELETE "+cedarWarehousePredefinedPath, http.StatusNoContent, nil)

	out, err := runLkctl(t, baseURL, "cedar", "project-predefined-policies", "reset", "-p", "proj-1")
	require.NoError(t, err)
	assert.Equal(t, "Predefined policies of project proj-1 reset\n", out)
	assert.Equal(t, "proj-1", projectReset.Last(t).Header.Get("x-project-id"))

	out, err = runLkctl(t, baseURL, "cedar", "project-predefined-policies", "reset")
	require.NoError(t, err)
	assert.Equal(t, "Predefined policies of the project reset\n", out)

	out, err = runLkctl(t, baseURL, "cedar", "warehouse-predefined-policies", "reset", testWarehouseID)
	require.NoError(t, err)
	assert.Equal(t, "Predefined policies of warehouse "+testWarehouseID+" reset\n", out)
}

// --break-glass exists only on the commands whose operations accept the
// x-break-glass header; every command is marked as Lakekeeper+.
func TestCedarPredefined_HelpMarksLakekeeperPlusAndBreakGlass(t *testing.T) {
	t.Parallel()
	baseURL := noServer(t)

	for _, tc := range []struct {
		args       []string
		breakGlass bool
	}{
		{[]string{"cedar", "project-predefined-policies", "list"}, true},
		{[]string{"cedar", "project-predefined-policies", "set"}, true},
		{[]string{"cedar", "project-predefined-policies", "reset"}, false},
		{[]string{"cedar", "warehouse-predefined-policies", "list"}, true},
		{[]string{"cedar", "warehouse-predefined-policies", "set"}, true},
		{[]string{"cedar", "warehouse-predefined-policies", "reset"}, false},
	} {
		help, err := runLkctl(t, baseURL, append(tc.args, "--help")...)
		require.NoError(t, err)
		assert.Contains(t, help, "(Lakekeeper+)", tc.args)
		if tc.breakGlass {
			assert.Contains(t, help, "--break-glass", tc.args)
		} else {
			assert.NotContains(t, help, "--break-glass", tc.args)
		}
	}
}

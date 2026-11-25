package commands

import (
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lakekeeper/go-lakekeeper/pkg/testutil"
)

const (
	testWarehouseID            = "0198618c-5be8-7a82-a0b9-1076c9dd12f0"
	cedarProjectPoliciesPath   = "/management/v1/permissions/cedar/project/policies"
	cedarWarehousePoliciesPath = "/management/v1/permissions/cedar/warehouse/" + testWarehouseID + "/policies"

	scopePoliciesBody = `{"can-write":true,"scope-version":7,"policies":[
		{"id":"p1","name":"readers","description":"read all","created-at":"2026-11-25T10:00:00Z","policy":{"effect":"permit"}}]}`
	applyScopePoliciesBody = `{"scope-version":8,"changed":[{"id":"p1","name":"readers","op":"create"}]}`
)

func TestCedarPolicies_BreakGlassSendsHeader(t *testing.T) {
	t.Parallel()
	mux, baseURL := testutil.NewServer(t)
	projectList := testutil.Respond(t, mux, "GET "+cedarProjectPoliciesPath, http.StatusOK, scopePoliciesBody)
	projectApply := testutil.Respond(t, mux, "POST "+cedarProjectPoliciesPath, http.StatusOK, applyScopePoliciesBody)
	warehouseList := testutil.Respond(t, mux, "GET "+cedarWarehousePoliciesPath, http.StatusOK, scopePoliciesBody)
	warehouseApply := testutil.Respond(t, mux, "POST "+cedarWarehousePoliciesPath, http.StatusOK, applyScopePoliciesBody)
	file := filepath.Join(t.TempDir(), "apply.json")
	require.NoError(t, os.WriteFile(file, []byte(`{"deletes":["old"]}`), 0o600))

	runs := []struct {
		args []string
		log  *testutil.RequestLog
	}{
		{[]string{"cedar", "project-policies", "list"}, projectList},
		{[]string{"cedar", "project-policies", "apply", "-f", file}, projectApply},
		{[]string{"cedar", "warehouse-policies", "list", testWarehouseID}, warehouseList},
		{[]string{"cedar", "warehouse-policies", "apply", testWarehouseID, "-f", file}, warehouseApply},
	}
	for _, run := range runs {
		_, err := runLkctl(t, baseURL, append(run.args, "--break-glass", "incident 42")...)
		require.NoError(t, err, run.args)
		assert.Equal(t, "incident 42", run.log.Last(t).Header.Get("x-break-glass"), run.args)
	}
	assert.JSONEq(t, `{"deletes":["old"]}`, string(projectApply.Last(t).Body))
}

func TestCedarPolicies_WithoutBreakGlassSendsNoHeader(t *testing.T) {
	t.Parallel()
	mux, baseURL := testutil.NewServer(t)
	list := testutil.Respond(t, mux, "GET "+cedarProjectPoliciesPath, http.StatusOK, scopePoliciesBody)

	_, err := runLkctl(t, baseURL, "cedar", "project-policies", "list", "-p", "proj-1")

	require.NoError(t, err)
	sent := list.Last(t)
	assert.NotContains(t, sent.Header, "X-Break-Glass")
	assert.Equal(t, "proj-1", sent.Header.Get("x-project-id"))
}

// --break-glass exists only on the commands whose operations accept the
// x-break-glass header.
func TestCedarPolicies_BreakGlassFlagOnlyWhereAccepted(t *testing.T) {
	t.Parallel()
	baseURL := noServer(t)

	for _, args := range [][]string{
		{"cedar", "project-policies", "list"},
		{"cedar", "project-policies", "apply"},
		{"cedar", "warehouse-policies", "list"},
		{"cedar", "warehouse-policies", "apply"},
	} {
		help, err := runLkctl(t, baseURL, append(args, "--help")...)
		require.NoError(t, err)
		assert.Contains(t, help, "--break-glass", args)
		assert.Contains(t, help, "(Lakekeeper+)", args)
	}
	for _, args := range [][]string{
		{"cedar", "project-policies", "get"},
		{"cedar", "warehouse-policies", "get"},
		{"cedar", "break-glass-status"},
		{"cedar", "schema"},
	} {
		help, err := runLkctl(t, baseURL, append(args, "--help")...)
		require.NoError(t, err)
		assert.NotContains(t, help, "--break-glass", args)
		assert.Contains(t, help, "(Lakekeeper+)", args)
	}
}

func TestCedarPolicies_WarehouseCommandsHaveNoProjectFlag(t *testing.T) {
	t.Parallel()
	help, err := runLkctl(t, noServer(t), "cedar", "warehouse-policies", "list", "--help")

	require.NoError(t, err)
	assert.NotContains(t, help, "--project")
}

func TestCedarPolicies_ListPrintsTable(t *testing.T) {
	t.Parallel()
	mux, baseURL := testutil.NewServer(t)
	testutil.Respond(t, mux, "GET "+cedarProjectPoliciesPath, http.StatusOK, scopePoliciesBody)

	out, err := runLkctl(t, baseURL, "cedar", "project-policies", "list")

	require.NoError(t, err)
	assert.Equal(t, "ID  NAME     CREATED AT            UPDATED AT\n"+
		"p1  readers  2026-11-25T10:00:00Z  \n"+
		"\nScope version: 7\nCan write: true\n", out)
}

func TestCedarPolicies_ListWithoutPolicies(t *testing.T) {
	t.Parallel()
	mux, baseURL := testutil.NewServer(t)
	testutil.Respond(t, mux, "GET "+cedarWarehousePoliciesPath, http.StatusOK,
		`{"can-write":false,"scope-version":0,"project-scope-version":3,"policies":[]}`)

	out, err := runLkctl(t, baseURL, "cedar", "warehouse-policies", "list", testWarehouseID)

	require.NoError(t, err)
	assert.Equal(t, "No Cedar policies\n\nScope version: 0\nProject scope version: 3\nCan write: false\n", out)
}

func TestCedarPolicies_GetWidePrintsDescription(t *testing.T) {
	t.Parallel()
	mux, baseURL := testutil.NewServer(t)
	testutil.Respond(t, mux, "GET "+cedarWarehousePoliciesPath+"/readers", http.StatusOK,
		`{"id":"p1","name":"readers","description":"read all","created-at":"2026-11-25T10:00:00Z",`+
			`"updated-at":"2026-11-25T11:00:00Z","policy":{"effect":"permit"}}`)

	out, err := runLkctl(t, baseURL, "cedar", "warehouse-policies", "get", testWarehouseID, "readers", "-o", "wide")

	require.NoError(t, err)
	assert.Equal(t, "ID  NAME     CREATED AT            UPDATED AT            DESCRIPTION\n"+
		"p1  readers  2026-11-25T10:00:00Z  2026-11-25T11:00:00Z  read all\n", out)
}

func TestCedarPolicies_ApplyPrintsChanges(t *testing.T) {
	t.Parallel()
	mux, baseURL := testutil.NewServer(t)
	testutil.Respond(t, mux, "POST "+cedarProjectPoliciesPath, http.StatusOK, applyScopePoliciesBody)

	out, err := runLkctlWithStdin(t, baseURL, `{"writes":[{"name":"readers","policy":{"effect":"permit"}}]}`,
		"cedar", "project-policies", "apply", "-f", "-")

	require.NoError(t, err)
	assert.Equal(t, "OP      NAME     ID\ncreate  readers  p1\n\nScope version: 8\n", out)
}

func TestCedarPolicies_ApplyWithoutChanges(t *testing.T) {
	t.Parallel()
	mux, baseURL := testutil.NewServer(t)
	testutil.Respond(t, mux, "POST "+cedarProjectPoliciesPath, http.StatusOK, `{"scope-version":8,"changed":[]}`)

	out, err := runLkctlWithStdin(t, baseURL, `{"writes":[]}`, "cedar", "project-policies", "apply", "-f", "-")

	require.NoError(t, err)
	assert.Equal(t, "No changes\n\nScope version: 8\n", out)
}

func TestCedarPolicies_ApplyRejectsInvalidJSON(t *testing.T) {
	t.Parallel()

	_, err := runLkctlWithStdin(t, noServer(t), `{"writes":`, "cedar", "project-policies", "apply", "-f", "-")

	require.ErrorContains(t, err, "decode apply request:")
}

func TestCedarBreakGlassStatus_SendsProjectAsQueryParameter(t *testing.T) {
	t.Parallel()
	mux, baseURL := testutil.NewServer(t)
	status := testutil.Respond(t, mux, "GET /management/v1/permissions/cedar/break-glass-status", http.StatusOK,
		`{"break-glass-available":true}`)

	out, err := runLkctl(t, baseURL, "cedar", "break-glass-status", "-p", "proj-1")

	require.NoError(t, err)
	assert.Equal(t, "Break-glass available: true\n", out)
	assert.Equal(t, "proj-1", status.Last(t).Query.Get("project-id"))
}

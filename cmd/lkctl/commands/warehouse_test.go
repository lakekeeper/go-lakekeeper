package commands

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	managementv1 "github.com/lakekeeper/go-lakekeeper/pkg/apis/management/v1"
	"github.com/lakekeeper/go-lakekeeper/pkg/testutil"
)

func TestWarehouseProjectFlag_DefaultsToServerDefaultProject(t *testing.T) {
	t.Parallel()

	flag := newWarehouseCmd(&clientOptions{}).PersistentFlags().Lookup("project")

	require.NotNil(t, flag)
	assert.Equal(t, "p", flag.Shorthand)
	assert.Empty(t, flag.DefValue, "-p must not default to a client-side project")
	assert.Contains(t, flag.Usage, "server's default project")
}

const createdWarehouseBody = `{
  "id": "9d6651da-5f19-11f1-ad69-1b7104fe3cf7", "warehouse-id": "9d6651da-5f19-11f1-ad69-1b7104fe3cf7",
  "name": "wh", "project-id": "00000000-0000-0000-0000-000000000000",
  "storage-profile": {"type": "s3", "bucket": "b", "region": "r", "flavor": "aws", "sts-enabled": false},
  "delete-profile": {"type": "hard"}, "status": "active", "protected": false,
  "allowed-format-versions": [1, 2, 3]}`

// writeWarehouseConfig writes a minimal `warehouse create` config; extra is
// spliced into the JSON object (e.g. `"project-id":"..."`).
func writeWarehouseConfig(t *testing.T, extra string) string {
	t.Helper()
	cfg := `{"warehouse-name":"wh","storage-profile":{"type":"s3","bucket":"b","region":"r","sts-enabled":false}`
	if extra != "" {
		cfg += "," + extra
	}
	path := filepath.Join(t.TempDir(), "wh.json")
	require.NoError(t, os.WriteFile(path, []byte(cfg+"}"), 0o600))
	return path
}

func TestWarehouseCreate_ProjectFlagSendsHeaderOnly(t *testing.T) {
	t.Parallel()
	mux, baseURL := testutil.NewServer(t)
	reqs := testutil.Respond(t, mux, "POST /management/v1/warehouse", http.StatusCreated, createdWarehouseBody)

	_, err := runLkctl(t, baseURL, "warehouse", "create", "wh", "-f", writeWarehouseConfig(t, ""), "-p", "11111111-1111-1111-1111-111111111111")

	require.NoError(t, err)
	got := reqs.Last(t)
	assert.Equal(t, "11111111-1111-1111-1111-111111111111", got.Header.Get("x-project-id"))
	var body map[string]any
	require.NoError(t, json.Unmarshal(got.Body, &body))
	assert.NotContains(t, body, "project-id", "the deprecated project-id body field must not be sent")
}

func TestWarehouseCreate_ConfigProjectIsSentAsHeader(t *testing.T) {
	t.Parallel()
	mux, baseURL := testutil.NewServer(t)
	reqs := testutil.Respond(t, mux, "POST /management/v1/warehouse", http.StatusCreated, createdWarehouseBody)
	config := writeWarehouseConfig(t, `"project-id":"22222222-2222-2222-2222-222222222222"`)

	_, err := runLkctl(t, baseURL, "warehouse", "create", "wh", "-f", config)

	require.NoError(t, err)
	got := reqs.Last(t)
	assert.Equal(t, "22222222-2222-2222-2222-222222222222", got.Header.Get("x-project-id"))
	var body map[string]any
	require.NoError(t, json.Unmarshal(got.Body, &body))
	assert.NotContains(t, body, "project-id", "the deprecated project-id body field must not be sent")
}

func TestWarehouseCreate_ConfigProjectMismatchingFlagIsRejected(t *testing.T) {
	t.Parallel()
	mux, baseURL := testutil.NewServer(t)
	reqs := testutil.Respond(t, mux, "POST /management/v1/warehouse", http.StatusCreated, createdWarehouseBody)
	config := writeWarehouseConfig(t, `"project-id":"22222222-2222-2222-2222-222222222222"`)

	_, err := runLkctl(t, baseURL, "warehouse", "create", "wh", "-f", config, "-p", "11111111-1111-1111-1111-111111111111")

	require.ErrorContains(t, err, "project id in config does not match --project")
	assert.Zero(t, reqs.Len(), "no request must be sent")
}

func TestWarehouseCreate_WithoutProjectFlagSendsNoProject(t *testing.T) {
	t.Parallel()
	mux, baseURL := testutil.NewServer(t)
	reqs := testutil.Respond(t, mux, "POST /management/v1/warehouse", http.StatusCreated, createdWarehouseBody)

	out, err := runLkctl(t, baseURL, "warehouse", "create", "wh", "-f", writeWarehouseConfig(t, ""))

	require.NoError(t, err)
	assert.Equal(t, "Warehouse wh created with id 9d6651da-5f19-11f1-ad69-1b7104fe3cf7\n", out)
	got := reqs.Last(t)
	assert.NotContains(t, got.Header, "X-Project-Id")
	var body map[string]any
	require.NoError(t, json.Unmarshal(got.Body, &body))
	assert.NotContains(t, body, "project-id", "without -p no project must be sent")
}

func TestWarehouseStatistics_PagingFlagsReachQueryAndNextTokenIsPrinted(t *testing.T) {
	t.Parallel()
	mux, baseURL := testutil.NewServer(t)
	reqs := testutil.Respond(t, mux, "GET /management/v1/warehouse/{id}/statistics", http.StatusOK, `{
	  "warehouse-ident": "wh-1", "next-page-token": "next-1",
	  "stats": [{"number-of-tables": 7, "number-of-views": 3, "timestamp": "2026-05-04T10:00:00Z", "updated-at": "2026-05-04T10:00:00Z"}]}`)

	out, err := runLkctl(t, baseURL, "warehouse", "statistics", "wh-1", "--page-size", "25", "--page-token", "page-2")

	require.NoError(t, err)
	got := reqs.Last(t)
	assert.Equal(t, "/management/v1/warehouse/wh-1/statistics", got.Path)
	assert.Equal(t, "25", got.Query.Get("page_size"))
	assert.Equal(t, "page-2", got.Query.Get("page_token"))
	assert.Contains(t, out, "2026-05-04T10:00:00Z")
	assert.Contains(t, out, "Next page token: next-1")
}

func TestWarehouseStatistics_NoNextTokenPrintsNoHint(t *testing.T) {
	t.Parallel()
	mux, baseURL := testutil.NewServer(t)
	reqs := testutil.Respond(t, mux, "GET /management/v1/warehouse/{id}/statistics", http.StatusOK,
		`{"warehouse-ident": "wh-1", "stats": []}`)

	out, err := runLkctl(t, baseURL, "warehouse", "statistics", "wh-1")

	require.NoError(t, err)
	assert.False(t, reqs.Last(t).Query.Has("page_token"), "no --page-token must send no page_token")
	assert.Equal(t, "No statistics available\n", out)
}

func TestWarehouseList_WithoutProjectFlagSendsNoProject(t *testing.T) {
	t.Parallel()
	mux, baseURL := testutil.NewServer(t)
	reqs := testutil.Respond(t, mux, "GET /management/v1/warehouse", http.StatusOK, map[string]any{"warehouses": []any{}})

	_, err := runLkctl(t, baseURL, "warehouse", "list")

	require.NoError(t, err)
	got := reqs.Last(t)
	assert.NotContains(t, got.Header, "X-Project-Id", "without -p no project header must be sent")
	assert.False(t, got.Query.Has("projectId"), "the deprecated projectId query parameter must not be sent")
}

func TestWarehouseList_ProjectFlagSendsHeader(t *testing.T) {
	t.Parallel()
	mux, baseURL := testutil.NewServer(t)
	reqs := testutil.Respond(t, mux, "GET /management/v1/warehouse", http.StatusOK, map[string]any{"warehouses": []any{}})

	_, err := runLkctl(t, baseURL, "warehouse", "list", "-p", "11111111-1111-1111-1111-111111111111")

	require.NoError(t, err)
	got := reqs.Last(t)
	assert.Equal(t, "11111111-1111-1111-1111-111111111111", got.Header.Get("x-project-id"))
	assert.False(t, got.Query.Has("projectId"), "the deprecated projectId query parameter must not be sent")
}

func TestNewWarehouseCmdSubcommands(t *testing.T) {
	t.Parallel()

	cmd := newWarehouseCmd(&clientOptions{})
	assert.Equal(t, "warehouse", cmd.Use)

	got := []string{}
	for _, sub := range cmd.Commands() {
		got = append(got, sub.Name())
	}
	sort.Strings(got)
	assert.Equal(t, []string{"access", "activate", "assignments", "create", "deactivate", "delete", "get", "grant", "list", "rename", "revoke", "set-protection", "statistics"}, got)
}

// TestWarehouseSetProtectionRequiresFlag verifies that `lkctl warehouse
// set-protection WAREHOUSEID` rejects a missing --protected flag before
// newClient is called: cobra's MarkFlagRequired surfaces the error.
func TestWarehouseSetProtectionRequiresFlag(t *testing.T) {
	t.Parallel()

	root := newWarehouseCmd(&clientOptions{})
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"set-protection", "wh-1"})

	err := root.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "protected")
}

// TestWarehouseAccessUserRoleMutex verifies the pre-network guard: passing
// both --user and --role to `lkctl warehouse access` fails before newClient
// is called.
func TestWarehouseAccessUserRoleMutex(t *testing.T) {
	t.Parallel()

	root := newWarehouseCmd(&clientOptions{})
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"access", "wh-1", "--user", "alice", "--role", "role-1"})

	err := root.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "mutually exclusive")
}

// TestWarehouseGrantNoUsersOrRoles verifies the pre-network guard: `lkctl
// warehouse grant <id> --assignments ownership` (without any --users or
// --roles) fails before newClient is called.
func TestWarehouseGrantNoUsersOrRoles(t *testing.T) {
	t.Parallel()

	root := newWarehouseCmd(&clientOptions{})
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"grant", "wh-1", "--assignments", "ownership"})

	err := root.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "at least one --users or --roles")
}

// TestWarehouseRevokeNoUsersOrRoles is the revoke-side counterpart to
// TestWarehouseGrantNoUsersOrRoles: same pre-network guard, same error.
func TestWarehouseRevokeNoUsersOrRoles(t *testing.T) {
	t.Parallel()

	root := newWarehouseCmd(&clientOptions{})
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"revoke", "wh-1", "--assignments", "ownership"})

	err := root.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "at least one --users or --roles")
}

func TestPrintWarehousesText(t *testing.T) {
	t.Parallel()

	s3Profile := managementv1.StorageProfileS3AsStorageProfile(
		managementv1.NewStorageProfileS3("my-bucket", "us-east-1", false, "s3"),
	)
	wh := managementv1.GetWarehouseResponse{
		WarehouseId:    "wh-1",
		Name:           "main",
		ProjectId:      "proj-1",
		Status:         managementv1.WAREHOUSESTATUS_ACTIVE,
		StorageProfile: s3Profile,
	}

	var buf bytes.Buffer
	require.NoError(t, printWarehouses(&buf, "text", wh))
	out := buf.String()

	assert.Contains(t, out, "wh-1")
	assert.Contains(t, out, "main")
	assert.Contains(t, out, "proj-1")
	assert.Contains(t, out, string(managementv1.WAREHOUSESTATUS_ACTIVE))
	assert.Contains(t, out, "s3", "STORAGE column should show the storage family")
}

func TestStorageFamily(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		profile managementv1.StorageProfile
		want    string
	}{
		{
			name:    "s3",
			profile: managementv1.StorageProfileS3AsStorageProfile(managementv1.NewStorageProfileS3("b", "r", false, "s3")),
			want:    "s3",
		},
		{
			name:    "gcs",
			profile: managementv1.StorageProfileGcsAsStorageProfile(managementv1.NewStorageProfileGcs("b", "gcs")),
			want:    "gcs",
		},
		{
			name:    "adls",
			profile: managementv1.StorageProfileAdlsAsStorageProfile(managementv1.NewStorageProfileAdls("acct", "fs", "adls")),
			want:    "adls",
		},
		{
			name:    "empty union",
			profile: managementv1.StorageProfile{},
			want:    "unknown",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, storageFamily(tc.profile))
		})
	}
}

func TestPrintWarehousesEmpty(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	require.NoError(t, printWarehouses(&buf, "text"))
	assert.Equal(t, "No warehouses available\n", buf.String())
}

func TestPrintWarehouseStatistics(t *testing.T) {
	t.Parallel()

	ts := time.Date(2026, 5, 4, 10, 0, 0, 0, time.UTC)
	stats := []managementv1.WarehouseStatistics{
		{NumberOfTables: 7, NumberOfViews: 3, Timestamp: ts, UpdatedAt: ts},
	}

	var buf bytes.Buffer
	require.NoError(t, printWarehouseStatistics(&buf, stats...))
	out := buf.String()

	assert.Contains(t, out, "TIMESTAMP")
	assert.Contains(t, out, "TABLES")
	assert.Contains(t, out, "VIEWS")
	assert.Contains(t, out, "UPDATED AT")
	assert.Contains(t, out, "2026-05-04T10:00:00Z")
	assert.Contains(t, out, "7")
	assert.Contains(t, out, "3")
}

func TestPrintWarehouseStatisticsEmpty(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	require.NoError(t, printWarehouseStatistics(&buf))
	assert.Equal(t, "No statistics available\n", buf.String())
}

// TestWarehouseCreateRejectsNameMismatch covers the pre-network validation
// path that compares the WarehouseName in the JSON config against the
// positional NAME argument. Reaching it requires a valid CreateWarehouseRequest
// JSON (warehouse-name + storage-profile), but the request never goes out
// over the network because the name check fires before newClient.
func TestWarehouseCreateRejectsNameMismatch(t *testing.T) {
	t.Parallel()

	configJSON := `{
  "warehouse-name": "Different Name",
  "storage-profile": {
    "type": "s3",
    "bucket": "my-bucket",
    "region": "us-east-1",
    "sts-enabled": false
  }
}`

	root := newWarehouseCmd(&clientOptions{})
	root.SetIn(strings.NewReader(configJSON))
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"create", "Expected Name", "-f", "-"})

	err := root.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "warehouse name in config does not match")
}

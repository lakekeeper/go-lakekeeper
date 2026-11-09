package commands

import (
	"bytes"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	managementv1 "github.com/lakekeeper/go-lakekeeper/pkg/apis/management/v1"
)

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

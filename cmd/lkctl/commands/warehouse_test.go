package commands

import (
	"bytes"
	"sort"
	"strings"
	"testing"

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
	assert.Equal(t, []string{"create", "delete", "get", "list"}, got)
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

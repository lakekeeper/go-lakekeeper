package client

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	managementv1 "github.com/lakekeeper/go-lakekeeper/pkg/apis/management/v1"
)

// warehouseResponseWithUnknownField is a real GetWarehouse response captured
// from Lakekeeper latest-main. It carries `allowed-format-versions`, a field
// added upstream after the v0.12.3 release that the committed spec does not
// know about. A client library must tolerate such additive server fields rather
// than reject the whole response.
const warehouseResponseWithUnknownField = `{
  "id": "9d6651da-5f19-11f1-ad69-1b7104fe3cf7",
  "warehouse-id": "9d6651da-5f19-11f1-ad69-1b7104fe3cf7",
  "name": "test-wh",
  "project-id": "00000000-0000-0000-0000-000000000000",
  "storage-profile": {
    "type": "s3",
    "bucket": "testacc",
    "region": "eu-local-1",
    "endpoint": "http://minio:9000/",
    "path-style-access": true,
    "flavor": "aws",
    "sts-enabled": false
  },
  "storage-credential-type": {"type": "s3", "credential-type": "access-key"},
  "delete-profile": {"type": "hard"},
  "status": "active",
  "protected": false,
  "allowed-format-versions": [1, 2, 3],
  "updated-at": null
}`

func TestGetWarehouseResponse_ToleratesUnknownField(t *testing.T) {
	t.Parallel()

	var wh managementv1.GetWarehouseResponse
	err := json.Unmarshal([]byte(warehouseResponseWithUnknownField), &wh)
	require.NoError(t, err)

	// The known fields still decode correctly alongside the unknown one.
	assert.Equal(t, "test-wh", wh.Name)
	assert.Equal(t, managementv1.WAREHOUSESTATUS_ACTIVE, wh.Status)
	require.NotNil(t, wh.StorageProfile.StorageProfileS3)
	assert.Equal(t, "testacc", wh.StorageProfile.StorageProfileS3.Bucket)
}

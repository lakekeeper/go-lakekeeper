package lakekeeper_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	managementv1 "github.com/lakekeeper/go-lakekeeper/pkg/apis/management/v1"
	"github.com/lakekeeper/go-lakekeeper/pkg/lakekeeper"
)

// TestUmbrellaTypeIdentity confirms the umbrella's type aliases resolve to
// the same types in the underlying packages, so values flow between the two
// imports without conversion. This is the load-bearing assumption that makes
// the umbrella a drop-in convenience instead of a parallel API.
//
// We assert identity by passing umbrella-built values to functions whose
// parameters are typed against the underlying managementv1 package — this
// only compiles if the aliases preserve type identity. Doing it via
// function calls (rather than explicit `var x managementv1.T = ...`)
// keeps staticcheck's redundant-type rule satisfied.
func TestUmbrellaTypeIdentity(t *testing.T) {
	t.Parallel()

	bucketOf := func(p managementv1.StorageProfile) string {
		return p.StorageProfileS3.Bucket
	}
	hasAccessKey := func(c managementv1.StorageCredential) bool {
		return c.StorageCredentialAccessKey != nil
	}
	whName := func(req *managementv1.CreateWarehouseRequest) string {
		return req.WarehouseName
	}

	sp := lakekeeper.NewS3Profile("bucket", "us-east-1",
		lakekeeper.WithS3Endpoint("http://minio:9000"))
	require.NotNil(t, sp.StorageProfileS3)
	assert.Equal(t, "bucket", bucketOf(sp))

	sc := lakekeeper.NewS3AccessKey("ak", "sk")
	require.True(t, hasAccessKey(sc))

	req := lakekeeper.NewCreateWarehouseRequest(sp, "test")
	require.NotNil(t, req)
	assert.Equal(t, "test", whName(req))
}

// TestUmbrellaPermissionConsts confirms the principal-kind re-exports
// resolve to the same constant value the underlying package uses, so a
// caller can mix umbrella and pkg/permissions imports without surprises.
func TestUmbrellaPermissionConsts(t *testing.T) {
	t.Parallel()

	out, err := lakekeeper.BuildAssignmentSet([]string{"admin"},
		lakekeeper.PrincipalSet{Users: []string{"alice"}})
	require.NoError(t, err)
	require.Len(t, out, 1)
	row, ok := lakekeeper.DescribeAssignment(out[0])
	require.True(t, ok)
	assert.Equal(t, "user", row.PrincipalType)
	assert.Equal(t, "alice", row.PrincipalID)
	assert.Equal(t, "admin", row.Relation)
}

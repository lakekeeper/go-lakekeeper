package lakekeeper_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	managementv1 "github.com/lakekeeper/go-lakekeeper/pkg/apis/management/v1"
	"github.com/lakekeeper/go-lakekeeper/pkg/lakekeeper"
	"github.com/lakekeeper/go-lakekeeper/pkg/testutil"
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

// TestUmbrellaFacadeCallOptions confirms the umbrella exposes the façade call
// option and list options, so umbrella-only callers can scope a warehouse
// list to a project.
func TestUmbrellaFacadeCallOptions(t *testing.T) {
	t.Parallel()

	mux, c := testutil.ServerMux(t)
	reqs := testutil.Respond(t, mux, "GET /management/v1/warehouse", http.StatusOK, map[string]any{"warehouses": []any{}})

	var opts []lakekeeper.CallOption
	opts = append(opts, lakekeeper.WithProject("11111111-1111-1111-1111-111111111111"))
	_, err := c.Warehouses.List(t.Context(), &lakekeeper.ListWarehousesOptions{}, opts...)

	require.NoError(t, err)
	assert.Equal(t, "11111111-1111-1111-1111-111111111111", reqs.Last(t).Header.Get("x-project-id"))
}

// TestUmbrellaListRolesOptions confirms umbrella-only callers can page roles.
func TestUmbrellaListRolesOptions(t *testing.T) {
	t.Parallel()

	mux, c := testutil.ServerMux(t)
	reqs := testutil.Respond(t, mux, "GET /management/v1/role", http.StatusOK, map[string]any{"roles": []any{}})

	_, err := c.Roles.List(t.Context(), &lakekeeper.ListRolesOptions{PageSize: 5})

	require.NoError(t, err)
	assert.Equal(t, "5", reqs.Last(t).Query.Get("pageSize"))
}

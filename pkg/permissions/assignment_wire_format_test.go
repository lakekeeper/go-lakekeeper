package permissions

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	managementv1 "github.com/lakekeeper/go-lakekeeper/pkg/apis/management/v1"
)

// TestAssignmentWireFormat is a contract test for the assignment-leaf wire
// shape that BuildAssignment / DescribeAssignment in this package round-trip
// through, and that downstream consumers (including the integration suite)
// depend on.
//
// The contract is `{"type": <relation>, "user"|"role": <id>}` for every
// resource family. If the OpenAPI spec or generator template ever renames
// those keys (e.g. emits `principal_id` instead of `user`/`role`),
// DescribeAssignment returns (_, false) and every permission test fails with
// the opaque "could not decode assignment" message. This test fails first,
// with a clear diagnostic, so the breakage points at the spec/preprocessor
// (api/openapi/preprocess) rather than at consumers.
//
// One leaf per assignment family (Server / Project / Warehouse / Role) is
// covered. All assignment leaves are emitted from the same generator
// template, so cross-family coverage guards against a partial-rename
// regression where one family is updated and another is not.
func TestAssignmentWireFormat(t *testing.T) {
	tests := []struct {
		name    string
		value   any
		wantKey string
	}{
		{
			name:    "server / user-principal leaf",
			value:   managementv1.NewServerAssignmentAdminUser("u1", "admin"),
			wantKey: "user",
		},
		{
			name:    "server / role-principal leaf",
			value:   managementv1.NewServerAssignmentAdminRole("r1", "admin"),
			wantKey: "role",
		},
		{
			name:    "project / user-principal leaf",
			value:   managementv1.NewProjectAssignmentProjectAdminUser("u1", "project_admin"),
			wantKey: "user",
		},
		{
			name:    "project / role-principal leaf",
			value:   managementv1.NewProjectAssignmentProjectAdminRole("r1", "project_admin"),
			wantKey: "role",
		},
		{
			name:    "warehouse / user-principal leaf",
			value:   managementv1.NewWarehouseAssignmentOwnershipUser("u1", "ownership"),
			wantKey: "user",
		},
		{
			name:    "warehouse / role-principal leaf",
			value:   managementv1.NewWarehouseAssignmentOwnershipRole("r1", "ownership"),
			wantKey: "role",
		},
		{
			name:    "role / user-principal leaf",
			value:   managementv1.NewRoleAssignmentOwnershipUser("u1", "ownership"),
			wantKey: "user",
		},
		{
			name:    "role / role-principal leaf",
			value:   managementv1.NewRoleAssignmentOwnershipRole("r1", "ownership"),
			wantKey: "role",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			b, err := json.Marshal(tc.value)
			require.NoError(t, err)

			var got map[string]any
			require.NoError(t, json.Unmarshal(b, &got))

			_, hasType := got["type"]
			assert.Truef(t, hasType, "expected `type` key, got %v", got)

			_, hasPrincipal := got[tc.wantKey]
			assert.Truef(t, hasPrincipal, "expected %q key, got %v", tc.wantKey, got)
		})
	}
}

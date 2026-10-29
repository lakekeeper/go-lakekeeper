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
// with a clear diagnostic, so the breakage points at the spec or generator
// config rather than at consumers.
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
			value:   withUser(managementv1.NewServerAssignmentAdmin("admin"), "u1"),
			wantKey: "user",
		},
		{
			name:    "server / role-principal leaf",
			value:   withRole(managementv1.NewServerAssignmentAdmin("admin"), "r1"),
			wantKey: "role",
		},
		{
			name:    "project / user-principal leaf",
			value:   withUser(managementv1.NewProjectAssignmentProjectAdmin("project_admin"), "u1"),
			wantKey: "user",
		},
		{
			name:    "project / role-principal leaf",
			value:   withRole(managementv1.NewProjectAssignmentProjectAdmin("project_admin"), "r1"),
			wantKey: "role",
		},
		{
			name:    "warehouse / user-principal leaf",
			value:   withUser(managementv1.NewWarehouseAssignmentOwnership("ownership"), "u1"),
			wantKey: "user",
		},
		{
			name:    "warehouse / role-principal leaf",
			value:   withRole(managementv1.NewWarehouseAssignmentOwnership("ownership"), "r1"),
			wantKey: "role",
		},
		{
			name:    "role / user-principal leaf",
			value:   withUser(managementv1.NewRoleAssignmentOwnership("ownership"), "u1"),
			wantKey: "user",
		},
		{
			name:    "role / role-principal leaf",
			value:   withRole(managementv1.NewRoleAssignmentOwnership("ownership"), "r1"),
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
			assert.Lenf(t, got, 2, "expected only `type` and %q, got %v", tc.wantKey, got)
		})
	}
}

// principalSetter is implemented by every generated assignment leaf: the spec
// models the principal as optional `user` and `role` fields on one struct.
type principalSetter interface {
	SetUser(string)
	SetRole(string)
}

func withUser[T principalSetter](a T, id string) T {
	a.SetUser(id)
	return a
}

func withRole[T principalSetter](a T, id string) T {
	a.SetRole(id)
	return a
}

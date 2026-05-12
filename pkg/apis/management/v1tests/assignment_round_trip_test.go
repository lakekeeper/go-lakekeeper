package v1tests

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	managementv1 "github.com/lakekeeper/go-lakekeeper/pkg/apis/management/v1"
)

// Golden fixtures for the *Assignment umbrellas. Each payload exercises one
// presence-discriminated variant of the inner UserOrRole oneOf combined with
// one outer-enum value of the assignment type. A conformant client must
// accept these payloads on unmarshal and reproduce the same shape on
// marshal — in particular it must NOT emit the sibling-variant key.

const (
	goldenRoleAssignmentOwnershipUser = `{
		"type": "ownership",
		"user": "u-1"
	}`

	goldenRoleAssignmentAssigneeRole = `{
		"type": "assignee",
		"role": "00000000-0000-0000-0000-000000000001"
	}`

	goldenNamespaceAssignmentOwnershipUser = `{
		"type": "ownership",
		"user": "u-2"
	}`

	goldenServerAssignmentAdminRole = `{
		"type": "admin",
		"role": "00000000-0000-0000-0000-000000000002"
	}`
)

func assertJSONEqual(t *testing.T, want, got []byte) {
	t.Helper()
	var wantMap, gotMap map[string]any
	require.NoError(t, json.Unmarshal(want, &wantMap))
	require.NoError(t, json.Unmarshal(got, &gotMap))
	assert.Equal(t, wantMap, gotMap)
}

func assertNoExtraKeys(t *testing.T, payload []byte, forbidden ...string) {
	t.Helper()
	var got map[string]any
	require.NoError(t, json.Unmarshal(payload, &got))
	for _, k := range forbidden {
		_, present := got[k]
		assert.Falsef(t, present, "payload should not contain key %q, got %v", k, got)
	}
}

func TestRoleAssignment_OwnershipUser_RoundTrip(t *testing.T) {
	t.Parallel()

	var a managementv1.RoleAssignment
	require.NoError(t, json.Unmarshal([]byte(goldenRoleAssignmentOwnershipUser), &a))

	out, err := json.Marshal(a)
	require.NoError(t, err)

	assertJSONEqual(t, []byte(goldenRoleAssignmentOwnershipUser), out)
	assertNoExtraKeys(t, out, "role")
}

func TestRoleAssignment_AssigneeRole_RoundTrip(t *testing.T) {
	t.Parallel()

	var a managementv1.RoleAssignment
	require.NoError(t, json.Unmarshal([]byte(goldenRoleAssignmentAssigneeRole), &a))

	out, err := json.Marshal(a)
	require.NoError(t, err)

	assertJSONEqual(t, []byte(goldenRoleAssignmentAssigneeRole), out)
	assertNoExtraKeys(t, out, "user")
}

func TestNamespaceAssignment_OwnershipUser_RoundTrip(t *testing.T) {
	t.Parallel()

	var a managementv1.NamespaceAssignment
	require.NoError(t, json.Unmarshal([]byte(goldenNamespaceAssignmentOwnershipUser), &a))

	out, err := json.Marshal(a)
	require.NoError(t, err)

	assertJSONEqual(t, []byte(goldenNamespaceAssignmentOwnershipUser), out)
	assertNoExtraKeys(t, out, "role")
}

func TestServerAssignment_AdminRole_RoundTrip(t *testing.T) {
	t.Parallel()

	var a managementv1.ServerAssignment
	require.NoError(t, json.Unmarshal([]byte(goldenServerAssignmentAdminRole), &a))

	out, err := json.Marshal(a)
	require.NoError(t, err)

	assertJSONEqual(t, []byte(goldenServerAssignmentAdminRole), out)
	assertNoExtraKeys(t, out, "user")
}

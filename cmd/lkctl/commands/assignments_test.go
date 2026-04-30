package commands

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	managementv1 "github.com/lakekeeper/go-lakekeeper/pkg/apis/management/v1"
)

func TestDescribeAssignmentUser(t *testing.T) {
	t.Parallel()

	a := managementv1.ServerAssignmentAdminAsServerAssignment(&managementv1.ServerAssignmentAdmin{
		ServerAssignmentAdminUser: managementv1.NewServerAssignmentAdminUser("alice", "admin"),
	})

	got, ok := describeAssignment(a)
	require.True(t, ok)
	assert.Equal(t, "User", got.PrincipalType)
	assert.Equal(t, "alice", got.PrincipalID)
	assert.Equal(t, "admin", got.Relation)
}

func TestDescribeAssignmentRole(t *testing.T) {
	t.Parallel()

	a := managementv1.ServerAssignmentOperatorAsServerAssignment(&managementv1.ServerAssignmentOperator{
		ServerAssignmentOperatorRole: managementv1.NewServerAssignmentOperatorRole("role-id", "operator"),
	})

	got, ok := describeAssignment(a)
	require.True(t, ok)
	assert.Equal(t, "Role", got.PrincipalType)
	assert.Equal(t, "role-id", got.PrincipalID)
	assert.Equal(t, "operator", got.Relation)
}

func TestPrintAssignmentsEmpty(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	require.NoError(t, printAssignments[managementv1.ServerAssignment](&buf))
	assert.Equal(t, "No assignments\n", buf.String())
}

func TestPrintAssignmentsDroppedWarning(t *testing.T) {
	t.Parallel()

	// Empty union value — describeAssignment returns ok=false, so the row drops.
	var empty managementv1.ServerAssignment

	var buf bytes.Buffer
	require.NoError(t, printAssignments(&buf, empty))
	assert.Contains(t, buf.String(), "1 assignment(s) could not be decoded")
}

func TestPrintAssignmentsTable(t *testing.T) {
	t.Parallel()

	a := managementv1.ServerAssignmentAdminAsServerAssignment(&managementv1.ServerAssignmentAdmin{
		ServerAssignmentAdminUser: managementv1.NewServerAssignmentAdminUser("alice", "admin"),
	})

	var buf bytes.Buffer
	require.NoError(t, printAssignments(&buf, a))
	out := buf.String()
	assert.Contains(t, out, "PRINCIPAL TYPE")
	assert.Contains(t, out, "alice")
	assert.Contains(t, out, "admin")
	assert.Contains(t, out, "User")
}

package commands

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	managementv1 "github.com/lakekeeper/go-lakekeeper/pkg/apis/management/v1"
)

func TestBuildAssignmentServerAdminUser(t *testing.T) {
	t.Parallel()

	got, err := buildAssignment[managementv1.ServerAssignment]("admin", principalUser, "alice")
	require.NoError(t, err)

	require.NotNil(t, got.ServerAssignmentAdmin)
	require.NotNil(t, got.ServerAssignmentAdmin.ServerAssignmentAdminUser)
	assert.Equal(t, "alice", got.ServerAssignmentAdmin.ServerAssignmentAdminUser.User)
	assert.Equal(t, "admin", got.ServerAssignmentAdmin.ServerAssignmentAdminUser.Type)
}

func TestBuildAssignmentServerOperatorRole(t *testing.T) {
	t.Parallel()

	got, err := buildAssignment[managementv1.ServerAssignment]("operator", principalRole, "role-id")
	require.NoError(t, err)

	require.NotNil(t, got.ServerAssignmentOperator)
	require.NotNil(t, got.ServerAssignmentOperator.ServerAssignmentOperatorRole)
	assert.Equal(t, "role-id", got.ServerAssignmentOperator.ServerAssignmentOperatorRole.Role)
	assert.Equal(t, "operator", got.ServerAssignmentOperator.ServerAssignmentOperatorRole.Type)
}

func TestBuildAssignmentProjectSelectUser(t *testing.T) {
	t.Parallel()

	got, err := buildAssignment[managementv1.ProjectAssignment]("select", principalUser, "alice")
	require.NoError(t, err)

	require.NotNil(t, got.ProjectAssignmentSelect)
	require.NotNil(t, got.ProjectAssignmentSelect.ProjectAssignmentSelectUser)
	assert.Equal(t, "alice", got.ProjectAssignmentSelect.ProjectAssignmentSelectUser.User)
	assert.Equal(t, "select", got.ProjectAssignmentSelect.ProjectAssignmentSelectUser.Type)
}

func TestBuildAssignmentMissingFields(t *testing.T) {
	t.Parallel()

	_, err := buildAssignment[managementv1.ServerAssignment]("", principalUser, "x")
	require.Error(t, err)

	_, err = buildAssignment[managementv1.ServerAssignment]("admin", principalUser, "")
	require.Error(t, err)
}

func TestBuildAssignmentRejectsUnknownRelation(t *testing.T) {
	t.Parallel()

	// `ServerAssignment.UnmarshalJSON` silently returns a zero value when the
	// `type` discriminator matches no known variant. buildAssignment must catch
	// that so a typo'd relation surfaces as a CLI error, not as an opaque
	// server-side rejection later.
	_, err := buildAssignment[managementv1.ServerAssignment]("janitor", principalUser, "alice")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "janitor")
}

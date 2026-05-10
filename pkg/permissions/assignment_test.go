package permissions

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	managementv1 "github.com/lakekeeper/go-lakekeeper/pkg/apis/management/v1"
)

func TestBuildAssignmentServerAdminUser(t *testing.T) {
	t.Parallel()

	got, err := BuildAssignment[managementv1.ServerAssignment]("admin", PrincipalUser, "alice")
	require.NoError(t, err)

	require.NotNil(t, got.ServerAssignmentAdmin)
	require.NotNil(t, got.ServerAssignmentAdmin.ServerAssignmentAdminUser)
	assert.Equal(t, "alice", got.ServerAssignmentAdmin.ServerAssignmentAdminUser.User)
	assert.Equal(t, "admin", got.ServerAssignmentAdmin.ServerAssignmentAdminUser.Type)
}

func TestBuildAssignmentServerOperatorRole(t *testing.T) {
	t.Parallel()

	got, err := BuildAssignment[managementv1.ServerAssignment]("operator", PrincipalRole, "role-id")
	require.NoError(t, err)

	require.NotNil(t, got.ServerAssignmentOperator)
	require.NotNil(t, got.ServerAssignmentOperator.ServerAssignmentOperatorRole)
	assert.Equal(t, "role-id", got.ServerAssignmentOperator.ServerAssignmentOperatorRole.Role)
	assert.Equal(t, "operator", got.ServerAssignmentOperator.ServerAssignmentOperatorRole.Type)
}

func TestBuildAssignmentProjectSelectUser(t *testing.T) {
	t.Parallel()

	got, err := BuildAssignment[managementv1.ProjectAssignment]("select", PrincipalUser, "alice")
	require.NoError(t, err)

	require.NotNil(t, got.ProjectAssignmentSelect)
	require.NotNil(t, got.ProjectAssignmentSelect.ProjectAssignmentSelectUser)
	assert.Equal(t, "alice", got.ProjectAssignmentSelect.ProjectAssignmentSelectUser.User)
	assert.Equal(t, "select", got.ProjectAssignmentSelect.ProjectAssignmentSelectUser.Type)
}

func TestBuildAssignmentMissingFields(t *testing.T) {
	t.Parallel()

	_, err := BuildAssignment[managementv1.ServerAssignment]("", PrincipalUser, "x")
	require.Error(t, err)

	_, err = BuildAssignment[managementv1.ServerAssignment]("admin", PrincipalUser, "")
	require.Error(t, err)
}

func TestBuildAssignmentRejectsZeroPrincipalKind(t *testing.T) {
	t.Parallel()

	// A caller who forgets to set the kind (e.g. `var k PrincipalKind`) must
	// get an error rather than silently a user-principal payload. Guards
	// against the zero-value-equals-PrincipalUser footgun.
	var unset PrincipalKind
	_, err := BuildAssignment[managementv1.ServerAssignment]("admin", unset, "alice")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "principal kind")
}

func TestBuildAssignmentRejectsUnknownRelation(t *testing.T) {
	t.Parallel()

	// `ServerAssignment.UnmarshalJSON` silently returns a zero value when the
	// `type` discriminator matches no known variant. BuildAssignment must
	// catch that so a typo'd relation surfaces as a CLI error, not as an
	// opaque server-side rejection later.
	_, err := BuildAssignment[managementv1.ServerAssignment]("janitor", PrincipalUser, "alice")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "janitor")
}

func TestDescribeAssignmentUser(t *testing.T) {
	t.Parallel()

	a := managementv1.ServerAssignmentAdminAsServerAssignment(&managementv1.ServerAssignmentAdmin{
		ServerAssignmentAdminUser: managementv1.NewServerAssignmentAdminUser("alice", "admin"),
	})

	got, ok := DescribeAssignment(a)
	require.True(t, ok)
	assert.Equal(t, "user", got.PrincipalType)
	assert.Equal(t, "alice", got.PrincipalID)
	assert.Equal(t, "admin", got.Relation)
}

func TestDescribeAssignmentRole(t *testing.T) {
	t.Parallel()

	a := managementv1.ServerAssignmentOperatorAsServerAssignment(&managementv1.ServerAssignmentOperator{
		ServerAssignmentOperatorRole: managementv1.NewServerAssignmentOperatorRole("role-id", "operator"),
	})

	got, ok := DescribeAssignment(a)
	require.True(t, ok)
	assert.Equal(t, "role", got.PrincipalType)
	assert.Equal(t, "role-id", got.PrincipalID)
	assert.Equal(t, "operator", got.Relation)
}

func TestDescribeAssignmentEmpty(t *testing.T) {
	t.Parallel()

	var empty managementv1.ServerAssignment
	_, ok := DescribeAssignment(empty)
	assert.False(t, ok)
}

func TestBuildAssignmentSetExpandsCartesian(t *testing.T) {
	t.Parallel()

	// 2 relations × (2 users + 1 role) = 6 assignments, in
	// (rel0,user0) (rel0,user1) (rel0,role0) (rel1,user0) (rel1,user1) (rel1,role0) order.
	got, err := BuildAssignmentSet[managementv1.ServerAssignment](
		[]string{"admin", "operator"},
		PrincipalSet{Users: []string{"alice", "bob"}, Roles: []string{"team"}},
	)
	require.NoError(t, err)
	require.Len(t, got, 6)

	rows := make([]AssignmentRow, 0, len(got))
	for _, a := range got {
		row, ok := DescribeAssignment(a)
		require.True(t, ok)
		rows = append(rows, row)
	}
	assert.ElementsMatch(t, []AssignmentRow{
		{PrincipalType: "user", PrincipalID: "alice", Relation: "admin"},
		{PrincipalType: "user", PrincipalID: "bob", Relation: "admin"},
		{PrincipalType: "role", PrincipalID: "team", Relation: "admin"},
		{PrincipalType: "user", PrincipalID: "alice", Relation: "operator"},
		{PrincipalType: "user", PrincipalID: "bob", Relation: "operator"},
		{PrincipalType: "role", PrincipalID: "team", Relation: "operator"},
	}, rows)
}

func TestBuildAssignmentSetRejectsEmptyPrincipals(t *testing.T) {
	t.Parallel()

	_, err := BuildAssignmentSet[managementv1.ServerAssignment]([]string{"admin"}, PrincipalSet{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no principals")
}

func TestBuildAssignmentSetPropagatesUnknownRelation(t *testing.T) {
	t.Parallel()

	_, err := BuildAssignmentSet[managementv1.ServerAssignment](
		[]string{"definitely-not-a-relation"},
		PrincipalSet{Users: []string{"alice"}},
	)
	require.Error(t, err)
}

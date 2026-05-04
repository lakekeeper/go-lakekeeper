package commands

import (
	"bytes"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	managementv1 "github.com/lakekeeper/go-lakekeeper/pkg/apis/management/v1"
)

func TestNewServerCmdHasSubcommands(t *testing.T) {
	t.Parallel()

	cmd := newServerCmd(&clientOptions{})
	assert.Equal(t, "server", cmd.Use)

	got := []string{}
	for _, sub := range cmd.Commands() {
		got = append(got, sub.Name())
	}
	sort.Strings(got)
	assert.Equal(t, []string{"access", "assignments", "bootstrap", "grant", "info", "revoke"}, got)
}

// TestServerRevokeNoUsersOrRoles verifies the pre-network guard: `lkctl
// server revoke --assignments admin` (without any --users or --roles) fails
// before newClient is called.
func TestServerRevokeNoUsersOrRoles(t *testing.T) {
	t.Parallel()

	root := newServerCmd(&clientOptions{})
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"revoke", "--assignments", "admin"})

	err := root.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "at least one --users or --roles")
}

func TestPrintServerInfoText(t *testing.T) {
	t.Parallel()

	info := &managementv1.ServerInfo{
		ServerId:                     "srv-1",
		Version:                      "0.9.1",
		LakekeeperVersion:            "0.9.1",
		DefaultProjectId:             *managementv1.NewNullableString(managementv1.PtrString("default-proj")),
		Bootstrapped:                 true,
		AuthzBackend:                 "openfga",
		AwsSystemIdentitiesEnabled:   true,
		AzureSystemIdentitiesEnabled: false,
		GcpSystemIdentitiesEnabled:   true,
		Queues:                       []string{"q1", "q2"},
	}

	var buf bytes.Buffer
	require.NoError(t, printServerInfo(&buf, info))
	out := buf.String()

	for _, want := range []string{"srv-1", "0.9.1", "openfga", "default-proj", "q1", "q2"} {
		assert.Contains(t, out, want, "expected %q in %q", want, out)
	}
}

func TestPrintAllowedActionsEmpty(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	require.NoError(t, printAllowedActions[managementv1.ServerAction](&buf, nil))
	assert.Equal(t, "No access\n", buf.String())
}

func TestPrintAllowedActionsTable(t *testing.T) {
	t.Parallel()

	actions := []managementv1.ServerAction{
		managementv1.SERVERACTION_CREATE_PROJECT,
		managementv1.SERVERACTION_PROVISION_USERS,
	}

	var buf bytes.Buffer
	require.NoError(t, printAllowedActions(&buf, actions))
	out := buf.String()

	assert.Contains(t, out, "ALLOWED ACTIONS")
	assert.Contains(t, out, string(managementv1.SERVERACTION_CREATE_PROJECT))
	assert.Contains(t, out, string(managementv1.SERVERACTION_PROVISION_USERS))
}

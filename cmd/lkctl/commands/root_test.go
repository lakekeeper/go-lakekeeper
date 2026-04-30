package commands

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCommandRegistersSubcommands(t *testing.T) {
	t.Parallel()

	root := NewCommand()
	got := []string{}
	for _, sub := range root.Commands() {
		got = append(got, sub.Name())
	}
	sort.Strings(got)
	want := []string{"catalog", "project", "role", "server", "user", "version", "warehouse", "whoami"}
	assert.Equal(t, want, got)
}

func TestNewCommandRegistersPersistentClientFlags(t *testing.T) {
	t.Parallel()

	root := NewCommand()
	for _, name := range []string{"server", "auth-url", "client-id", "client-secret", "scopes", "bootstrap", "debug"} {
		require.NotNil(t, root.PersistentFlags().Lookup(name), "missing persistent flag %q", name)
	}
}

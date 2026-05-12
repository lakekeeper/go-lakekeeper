package commands

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewUserCmdHasSubcommands(t *testing.T) {
	t.Parallel()

	cmd := newUserCmd(&clientOptions{})
	assert.Equal(t, "user", cmd.Use)

	got := []string{}
	for _, sub := range cmd.Commands() {
		got = append(got, sub.Name())
	}
	sort.Strings(got)
	assert.Equal(t, []string{"create", "delete", "get", "list"}, got)
}

package commands

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddListFlags(t *testing.T) {
	t.Parallel()

	cmd := &cobra.Command{}
	var opts listOpts
	addListFlags(cmd, &opts)

	require.NoError(t, cmd.ParseFlags([]string{"--limit", "42", "--token", "abc", "--name", "alice"}))
	assert.Equal(t, int64(42), opts.limit)
	assert.Equal(t, "abc", opts.token)
	assert.Equal(t, "alice", opts.name)
}

func TestAddListFlagsDefaults(t *testing.T) {
	t.Parallel()

	cmd := &cobra.Command{}
	var opts listOpts
	addListFlags(cmd, &opts)
	require.NoError(t, cmd.ParseFlags(nil))
	assert.Equal(t, int64(100), opts.limit)
}

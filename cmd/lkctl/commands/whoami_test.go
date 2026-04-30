package commands

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewWhoamiCmdShape(t *testing.T) {
	t.Parallel()

	cmd := newWhoamiCmd(&clientOptions{})
	assert.Equal(t, "whoami", cmd.Use)

	out, err := cmd.Flags().GetString("output")
	require.NoError(t, err)
	assert.Equal(t, "text", out)
}

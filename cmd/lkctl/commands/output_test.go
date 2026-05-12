package commands

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPrintJSON(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	require.NoError(t, printJSON(&buf, map[string]int{"a": 1}))

	assert.Equal(t, "{\n  \"a\": 1\n}\n", buf.String())
}

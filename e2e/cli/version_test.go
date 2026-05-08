//go:build e2e_cli

package clie2e

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestVersionShort exercises the `--short` shape: just `lkctl: <version>`
// on stdout, no server round-trip needed when --client is also set. Runs on
// every backend because --client short-circuits before any server call.
func TestVersionShort(t *testing.T) {
	t.Parallel()

	stdout, stderr, code := runRaw(t, nil,
		"version", "--short", "--client",
	)
	require.Equalf(t, 0, code, "exit %d\nstderr: %s", code, stderr)
	assert.True(t, strings.HasPrefix(strings.TrimSpace(string(stdout)), "lkctl:"),
		"expected `lkctl: ...` output, got %q", stdout)
}

// TestVersionClientOnly confirms --client returns success without contacting
// the server; we don't supply auth flags deliberately.
func TestVersionClientOnly(t *testing.T) {
	t.Parallel()

	stdout, stderr, code := runRaw(t, nil, "version", "--client")
	require.Equalf(t, 0, code, "exit %d\nstderr: %s", code, stderr)
	assert.Contains(t, string(stdout), "lkctl:")
	assert.NotContains(t, string(stdout), "lakekeeper:",
		"--client should not include server version line")
}

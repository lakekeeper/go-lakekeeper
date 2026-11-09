package commands

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	managementv1 "github.com/lakekeeper/go-lakekeeper/pkg/apis/management/v1"
)

func TestNewWhoamiCmdShape(t *testing.T) {
	t.Parallel()

	cmd := newWhoamiCmd(&clientOptions{})
	assert.Equal(t, "whoami", cmd.Use)

	out, err := cmd.Flags().GetString("output")
	require.NoError(t, err)
	assert.Equal(t, "text", out)
}

func newTestWhoami() *managementv1.WhoamiResponse {
	created := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	w := managementv1.NewWhoamiResponse(created, "oidc~abc", managementv1.USERLASTUPDATEDWITH_CREATE_ENDPOINT,
		"Alice", managementv1.USERTYPE_HUMAN, true)
	w.SetEmail("alice@example.com")
	return w
}

func TestPrintWhoamiText(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	require.NoError(t, printWhoami(&buf, "text", newTestWhoami()))

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	require.Len(t, lines, 2)
	assert.Equal(t, []string{"ID", "NAME", "EMAIL", "USER", "TYPE", "INSTANCE", "ADMIN"}, strings.Fields(lines[0]))
	assert.Equal(t, []string{"oidc~abc", "Alice", "alice@example.com", "human", "true"}, strings.Fields(lines[1]))
}

func TestPrintWhoamiWide(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	require.NoError(t, printWhoami(&buf, "wide", newTestWhoami()))

	got := buf.String()
	assert.Contains(t, got, "INSTANCE ADMIN")
	assert.Contains(t, got, "CREATED AT")
	assert.Contains(t, got, "2026-01-02T03:04:05Z")
	assert.Contains(t, got, "LAST UPDATED WITH")
}

func TestPrintWhoamiUnknownFormat(t *testing.T) {
	t.Parallel()

	err := printWhoami(&bytes.Buffer{}, "yaml", newTestWhoami())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown output format")
}

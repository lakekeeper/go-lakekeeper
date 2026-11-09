package client_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/lakekeeper/go-lakekeeper/pkg/testutil"
)

// Remaining nil-request guards of façades not yet covered by their own
// <resource>_test.go. Each resource slice moves its guards next to its
// façade tests and deletes them here.

func TestProjects_NilRequestsRejected(t *testing.T) {
	t.Parallel()
	_, c := testutil.ServerMux(t)

	_, err := c.Projects.Create(t.Context(), nil)
	require.ErrorContains(t, err, "request must not be nil")

	err = c.Projects.Rename(t.Context(), "p", nil)
	require.ErrorContains(t, err, "request must not be nil")
}

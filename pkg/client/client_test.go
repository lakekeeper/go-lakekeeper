package client_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lakekeeper/go-lakekeeper/pkg/client"
	"github.com/lakekeeper/go-lakekeeper/pkg/testutil"
)

func TestNew_BaseURLNormalization(t *testing.T) {
	t.Parallel()
	for name, suffix := range map[string]string{
		"server root":                  "",
		"trailing slash":               "/",
		"management prefix":            "/management/v1",
		"management prefix with slash": "/management/v1/",
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			mux, root := testutil.NewServer(t)
			info := testutil.Respond(t, mux, "GET /management/v1/info", http.StatusOK, serverInfoBody(true))
			c, err := client.New(t.Context(), root+suffix, "test-token", client.WithoutRetries())
			require.NoError(t, err)

			_, err = c.Server.Info(t.Context())

			require.NoError(t, err)
			assert.Equal(t, 1, info.Len(), "request must reach /management/v1/info")
			assert.Equal(t, root, c.BaseURL().String())
		})
	}
}

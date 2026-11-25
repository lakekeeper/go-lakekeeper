//go:build integration

package integration

import (
	"context"
	"net/http"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2/clientcredentials"

	"github.com/lakekeeper/go-lakekeeper/pkg/client"
	"github.com/lakekeeper/go-lakekeeper/pkg/core"
)

// infoCounter counts the server info requests a client sends.
type infoCounter struct {
	base http.RoundTripper

	mu sync.Mutex
	n  int
}

func (ic *infoCounter) RoundTrip(r *http.Request) (*http.Response, error) {
	if r.URL.Path == "/management/v1/info" {
		ic.mu.Lock()
		ic.n++
		ic.mu.Unlock()
	}
	return ic.base.RoundTrip(r)
}

func (ic *infoCounter) count() int {
	ic.mu.Lock()
	defer ic.mu.Unlock()
	return ic.n
}

func TestCapabilities_ReportsLakekeeper(t *testing.T) {
	info, err := sharedClient.Server.Info(t.Context())
	require.NoError(t, err)

	got, err := sharedClient.Capabilities(t.Context())

	require.NoError(t, err)
	assert.Equal(t, client.EditionLakekeeper, got.Edition)
	assert.Equal(t, info.GetLakekeeperVersion(), got.LakekeeperVersion)
	assert.NotEmpty(t, got.LakekeeperVersion)
	assert.Empty(t, got.LakekeeperPlusVersion)
	assert.Equal(t, client.AuthzBackendOpenFGA, got.AuthzBackend)
	assert.Equal(t, info.GetQueues(), got.Queues)
}

func TestCedarSchema_OnLakekeeperIsUnsupported(t *testing.T) {
	cfg := clientcredentials.Config{
		ClientID:     os.Getenv("LAKEKEEPER_CLIENT_ID"),
		ClientSecret: os.Getenv("LAKEKEEPER_CLIENT_SECRET"),
		TokenURL:     os.Getenv("LAKEKEEPER_TOKEN_URL"),
		Scopes:       strings.Fields(os.Getenv("LAKEKEEPER_SCOPE")),
	}
	counter := &infoCounter{base: http.DefaultTransport}
	c, err := client.NewWithAuthSource(t.Context(), os.Getenv("LAKEKEEPER_BASE_URL"),
		&core.OAuthClientCredentialsAuthSource{TokenSource: cfg.TokenSource(context.Background())},
		client.WithHTTPClient(&http.Client{Transport: counter}))
	require.NoError(t, err)

	_, err = c.Cedar.Schema(t.Context())
	require.ErrorIs(t, err, client.ErrUnsupported)
	_, err = c.Cedar.Schema(t.Context())
	require.ErrorIs(t, err, client.ErrUnsupported)

	var unsupported *client.UnsupportedError
	require.ErrorAs(t, err, &unsupported)
	assert.Equal(t, client.ReasonNotLakekeeperPlus, unsupported.Reason)
	assert.Equal(t, client.EditionLakekeeper, unsupported.Server.Edition)
	assert.Regexp(t, `^get cedar schema: requires Lakekeeper\+; server is Lakekeeper \S+$`, err.Error())
	assert.Equal(t, 1, counter.count(), "server info requests")
}

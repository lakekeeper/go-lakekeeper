//go:build integration_plus

package plus

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

// requestCounter counts the requests a client sends, per URL path.
type requestCounter struct {
	base http.RoundTripper

	mu     sync.Mutex
	byPath map[string]int
}

func (rc *requestCounter) RoundTrip(r *http.Request) (*http.Response, error) {
	rc.mu.Lock()
	rc.byPath[r.URL.Path]++
	rc.mu.Unlock()
	return rc.base.RoundTrip(r)
}

func (rc *requestCounter) count(path string) int {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	return rc.byPath[path]
}

func (rc *requestCounter) total() int {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	n := 0
	for _, c := range rc.byPath {
		n += c
	}
	return n
}

// countingClient returns a fresh client, with its own capability cache,
// whose requests are counted.
func countingClient(t *testing.T) (*client.Client, *requestCounter) {
	t.Helper()
	cfg := clientcredentials.Config{
		ClientID:     os.Getenv("LAKEKEEPER_CLIENT_ID"),
		ClientSecret: os.Getenv("LAKEKEEPER_CLIENT_SECRET"),
		TokenURL:     os.Getenv("LAKEKEEPER_TOKEN_URL"),
		Scopes:       strings.Fields(os.Getenv("LAKEKEEPER_SCOPE")),
	}
	counter := &requestCounter{base: http.DefaultTransport, byPath: map[string]int{}}
	c, err := client.NewWithAuthSource(t.Context(), os.Getenv("LAKEKEEPER_BASE_URL"),
		&core.OAuthClientCredentialsAuthSource{TokenSource: cfg.TokenSource(context.Background())},
		client.WithHTTPClient(&http.Client{Transport: counter}))
	require.NoError(t, err)
	return c, counter
}

const (
	infoPath        = "/management/v1/info"
	cedarSchemaPath = "/management/v1/permissions/cedar/schema"
)

func TestCapabilities_ReportsLakekeeperPlus(t *testing.T) {
	info, err := sharedClient.Server.Info(t.Context())
	require.NoError(t, err)

	got, err := sharedClient.Capabilities(t.Context())

	require.NoError(t, err)
	assert.Equal(t, client.EditionLakekeeperPlus, got.Edition)
	assert.Equal(t, authzBackend, got.AuthzBackend)
	assert.NotEmpty(t, got.LakekeeperVersion)
	assert.Equal(t, info.GetLakekeeperVersion(), got.LakekeeperVersion)
	assert.NotEmpty(t, got.LakekeeperPlusVersion)
	assert.Equal(t, info.GetLakekeeperEnterpriseVersion(), got.LakekeeperPlusVersion)
	assert.Contains(t, got.Queues, client.QueueExpireSnapshots)
	assert.Equal(t, "Lakekeeper+ "+got.LakekeeperPlusVersion, got.String())
}

func TestCedarSchema_SucceedsWithOneRequest(t *testing.T) {
	if authzBackend != client.AuthzBackendCedar {
		t.Skipf("needs the cedar authorization backend, running %s", authzBackend)
	}
	c, counter := countingClient(t)

	schema, err := c.Cedar.Schema(t.Context())

	require.NoError(t, err)
	assert.NotEmpty(t, schema.GetSchemaFragments())
	assert.Equal(t, 1, counter.total(), "requests sent")
	assert.Equal(t, 1, counter.count(cedarSchemaPath), "schema requests")
}

func TestCedarSchema_OnOpenFGAIsUnsupported(t *testing.T) {
	if authzBackend != client.AuthzBackendOpenFGA {
		t.Skipf("needs the openfga authorization backend, running %s", authzBackend)
	}
	c, counter := countingClient(t)

	_, err := c.Cedar.Schema(t.Context())
	require.ErrorIs(t, err, client.ErrUnsupported)
	_, err = c.Cedar.Schema(t.Context())
	require.ErrorIs(t, err, client.ErrUnsupported)

	var unsupported *client.UnsupportedError
	require.ErrorAs(t, err, &unsupported)
	assert.Equal(t, client.ReasonAuthzBackend, unsupported.Reason)
	assert.Equal(t, "Cedar", unsupported.Feature)
	assert.Equal(t, client.AuthzBackendOpenFGA, unsupported.Server.AuthzBackend)
	assert.Equal(t, 1, counter.count(infoPath), "server info requests")
}

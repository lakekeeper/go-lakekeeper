package client_test

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/apache/iceberg-go/catalog/rest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lakekeeper/go-lakekeeper/pkg/client"
	"github.com/lakekeeper/go-lakekeeper/pkg/testutil"
)

// recordingTransport records every request it forwards to Base (or
// http.DefaultTransport when Base is nil).
type recordingTransport struct {
	Base http.RoundTripper

	mu   sync.Mutex
	reqs []*http.Request
}

func (r *recordingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	r.mu.Lock()
	r.reqs = append(r.reqs, req.Clone(req.Context()))
	r.mu.Unlock()
	base := r.Base
	if base == nil {
		base = http.DefaultTransport
	}
	return base.RoundTrip(req)
}

func (r *recordingTransport) requests() []*http.Request {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]*http.Request(nil), r.reqs...)
}

func TestNew_WithHTTPClient_TransportSeesBootstrapRequestsWithAuth(t *testing.T) {
	t.Parallel()
	mux, baseURL := testutil.NewServer(t)
	testutil.Respond(t, mux, "GET /management/v1/info", http.StatusOK, serverInfoBody(false))
	testutil.Respond(t, mux, "POST /management/v1/bootstrap", http.StatusNoContent, nil)
	rt := &recordingTransport{}

	_, err := client.New(t.Context(), baseURL, "test-token",
		client.WithHTTPClient(&http.Client{Transport: rt}),
		client.WithInitialBootstrap(true, true, nil))

	require.NoError(t, err)
	reqs := rt.requests()
	require.Len(t, reqs, 2, "info and bootstrap must both go through the supplied transport")
	assert.Equal(t, "/management/v1/info", reqs[0].URL.Path)
	assert.Equal(t, "/management/v1/bootstrap", reqs[1].URL.Path)
	for _, r := range reqs {
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
	}
}

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

func TestNew_WithHTTPClient_DoesNotMutateSuppliedClient(t *testing.T) {
	t.Parallel()
	mux, baseURL := testutil.NewServer(t)
	testutil.Respond(t, mux, "GET /management/v1/info", http.StatusOK, serverInfoBody(true))
	rt := &recordingTransport{}
	supplied := &http.Client{Transport: rt, Timeout: 7 * time.Second}

	c, err := client.New(t.Context(), baseURL, "test-token", client.WithHTTPClient(supplied))
	require.NoError(t, err)
	_, err = c.Server.Info(t.Context())
	require.NoError(t, err)

	assert.Same(t, rt, supplied.Transport, "Transport must not be replaced")
	assert.Equal(t, 7*time.Second, supplied.Timeout)
	assert.Nil(t, supplied.CheckRedirect)
	assert.Nil(t, supplied.Jar)
}

func TestNew_WithHTTPClient_NilFailsConstruction(t *testing.T) {
	t.Parallel()
	_, baseURL := testutil.NewServer(t)

	_, err := client.New(t.Context(), baseURL, "test-token", client.WithHTTPClient(nil))

	require.ErrorContains(t, err, "http client must not be nil")
}

func TestNew_WithHTTPClient_TimeoutCarriesOver(t *testing.T) {
	t.Parallel()
	mux, baseURL := testutil.NewServer(t)
	release := make(chan struct{})
	t.Cleanup(func() { close(release) })
	mux.HandleFunc("GET /management/v1/info", func(http.ResponseWriter, *http.Request) {
		select {
		case <-release:
		case <-time.After(5 * time.Second):
		}
	})
	c, err := client.New(t.Context(), baseURL, "test-token", client.WithoutRetries(),
		client.WithHTTPClient(&http.Client{Timeout: 50 * time.Millisecond}))
	require.NoError(t, err)

	_, err = c.Server.Info(t.Context())

	require.ErrorContains(t, err, "Client.Timeout exceeded")
}

func TestNew_WithHTTPClient_PrivateCATLS(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	testutil.Respond(t, mux, "GET /management/v1/info", http.StatusOK, serverInfoBody(true))
	server := httptest.NewTLSServer(mux)
	t.Cleanup(server.Close)

	t.Run("default transport rejects the certificate", func(t *testing.T) {
		t.Parallel()
		c, err := client.New(t.Context(), server.URL, "test-token", client.WithoutRetries())
		require.NoError(t, err)

		_, err = c.Server.Info(t.Context())

		var certErr *tls.CertificateVerificationError
		require.ErrorAs(t, err, &certErr)
	})

	t.Run("transport trusting the CA succeeds", func(t *testing.T) {
		t.Parallel()
		c, err := client.New(t.Context(), server.URL, "test-token", client.WithoutRetries(),
			client.WithHTTPClient(server.Client()))
		require.NoError(t, err)

		_, err = c.Server.Info(t.Context())

		require.NoError(t, err)
	})
}

// newCatalogTLSServer starts a TLS test server answering the Iceberg REST
// GET /catalog/v1/config request, and returns it with the request log.
func newCatalogTLSServer(t *testing.T) (*httptest.Server, *testutil.RequestLog) {
	t.Helper()
	mux := http.NewServeMux()
	config := testutil.Respond(t, mux, "GET /catalog/v1/config", http.StatusOK,
		map[string]any{"defaults": map[string]string{}, "overrides": map[string]string{}})
	server := httptest.NewTLSServer(mux)
	t.Cleanup(server.Close)
	return server, config
}

func TestCatalogV1_UsesSuppliedTransport(t *testing.T) {
	t.Parallel()
	server, config := newCatalogTLSServer(t)
	c, err := client.New(t.Context(), server.URL, "test-token", client.WithoutRetries(),
		client.WithHTTPClient(server.Client()))
	require.NoError(t, err)

	_, err = c.CatalogV1(t.Context(), "project-1", "warehouse-1")

	require.NoError(t, err, "the catalog must trust the private CA of the supplied transport")
	got := config.Last(t)
	assert.Equal(t, "Bearer test-token", got.Header.Get("Authorization"))
	assert.Equal(t, "project-1/warehouse-1", got.Query.Get("warehouse"))
}

func TestCatalogV1_CallerTransportWins(t *testing.T) {
	t.Parallel()
	server, _ := newCatalogTLSServer(t)
	c, err := client.New(t.Context(), server.URL, "test-token", client.WithoutRetries(),
		client.WithHTTPClient(server.Client()))
	require.NoError(t, err)
	rt := &recordingTransport{Base: server.Client().Transport}

	_, err = c.CatalogV1(t.Context(), "project-1", "warehouse-1", rest.WithCustomTransport(rt))

	require.NoError(t, err)
	assert.NotEmpty(t, rt.requests(), "the caller's rest.WithCustomTransport must take precedence")
}

func TestCatalogV1_CallerTransportFactoryWins(t *testing.T) {
	t.Parallel()
	server, _ := newCatalogTLSServer(t)
	c, err := client.New(t.Context(), server.URL, "test-token", client.WithoutRetries(),
		client.WithHTTPClient(&http.Client{}))
	require.NoError(t, err)
	rt := &recordingTransport{Base: server.Client().Transport}
	factory := func(*tls.Config) (http.RoundTripper, func()) { return rt, nil }

	_, err = c.CatalogV1(t.Context(), "project-1", "warehouse-1", rest.WithTransportFactory(factory))

	require.NoError(t, err)
	assert.NotEmpty(t, rt.requests(), "the caller's rest.WithTransportFactory must take precedence")
}

func TestCatalogV1_CallerTLSConfigHonoured(t *testing.T) {
	t.Parallel()
	server, _ := newCatalogTLSServer(t)
	c, err := client.New(t.Context(), server.URL, "test-token", client.WithoutRetries(),
		client.WithHTTPClient(&http.Client{}))
	require.NoError(t, err)
	trustsServer := server.Client().Transport.(*http.Transport).TLSClientConfig

	_, err = c.CatalogV1(t.Context(), "project-1", "warehouse-1", rest.WithTLSConfig(trustsServer))

	require.NoError(t, err, "rest.WithTLSConfig must keep working alongside WithHTTPClient")
}

func TestNew_WithHTTPClient_RetriesThroughSuppliedTransport(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name     string
		opts     []client.Option
		wantReqs int
	}{
		{
			name:     "retries on",
			opts:     []client.Option{client.WithRetryMax(1), client.WithRetryWait(time.Millisecond, time.Millisecond)},
			wantReqs: 2,
		},
		{name: "WithoutRetries", opts: []client.Option{client.WithoutRetries()}, wantReqs: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			mux, baseURL := testutil.NewServer(t)
			testutil.Respond(t, mux, "GET /management/v1/info", http.StatusServiceUnavailable, nil)
			rt := &recordingTransport{}
			opts := append([]client.Option{client.WithHTTPClient(&http.Client{Transport: rt})}, tc.opts...)
			c, err := client.New(t.Context(), baseURL, "test-token", opts...)
			require.NoError(t, err)

			_, err = c.Server.Info(t.Context())

			require.Error(t, err)
			assert.Len(t, rt.requests(), tc.wantReqs)
		})
	}
}

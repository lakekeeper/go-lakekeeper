package core

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeAuthSource implements AuthSource with values set by the test.
type fakeAuthSource struct {
	key, value string
	err        error
}

func (*fakeAuthSource) Init(context.Context) error { return nil }

func (f *fakeAuthSource) Header(context.Context) (string, string, error) {
	return f.key, f.value, f.err
}

func (f *fakeAuthSource) GetToken(context.Context) (string, error) {
	return f.value, f.err
}

// captureRT records the request it sees and returns a canned 200 response.
type captureRT struct {
	seen *http.Request
}

func (c *captureRT) RoundTrip(req *http.Request) (*http.Response, error) {
	c.seen = req
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader("")),
		Request:    req,
	}, nil
}

func TestAuthRoundTripper(t *testing.T) {
	t.Parallel()

	t.Run("injects header when missing", func(t *testing.T) {
		t.Parallel()

		capture := &captureRT{}
		rt := &AuthRoundTripper{
			Base:       capture,
			AuthSource: &fakeAuthSource{key: "Authorization", value: "Bearer x"},
		}

		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "http://example.test", http.NoBody)
		require.NoError(t, err)

		resp, err := rt.RoundTrip(req)
		require.NoError(t, err)
		_ = resp.Body.Close()

		require.NotNil(t, capture.seen)
		assert.Equal(t, "Bearer x", capture.seen.Header.Get("Authorization"))
	})

	t.Run("does not overwrite pre-existing header", func(t *testing.T) {
		t.Parallel()

		capture := &captureRT{}
		rt := &AuthRoundTripper{
			Base:       capture,
			AuthSource: &fakeAuthSource{key: "Authorization", value: "Bearer auth-source"},
		}

		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "http://example.test", http.NoBody)
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer caller-token")

		resp, err := rt.RoundTrip(req)
		require.NoError(t, err)
		_ = resp.Body.Close()

		require.NotNil(t, capture.seen)
		assert.Equal(t, "Bearer caller-token", capture.seen.Header.Get("Authorization"))
	})

	t.Run("does not mutate original request", func(t *testing.T) {
		t.Parallel()

		capture := &captureRT{}
		rt := &AuthRoundTripper{
			Base:       capture,
			AuthSource: &fakeAuthSource{key: "Authorization", value: "Bearer x"},
		}

		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "http://example.test", http.NoBody)
		require.NoError(t, err)

		resp, err := rt.RoundTrip(req)
		require.NoError(t, err)
		_ = resp.Body.Close()

		// The header injection happens on a clone; the caller's request must
		// remain pristine so that retries / outer middleware see the original.
		assert.Empty(t, req.Header.Get("Authorization"))
		require.NotNil(t, capture.seen)
		assert.NotSame(t, req, capture.seen, "captured request should be a clone of the original")
	})

	t.Run("wraps auth source error", func(t *testing.T) {
		t.Parallel()

		sentinel := errors.New("boom")
		capture := &captureRT{}
		rt := &AuthRoundTripper{
			Base:       capture,
			AuthSource: &fakeAuthSource{key: "Authorization", err: sentinel},
		}

		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "http://example.test", http.NoBody)
		require.NoError(t, err)

		resp, err := rt.RoundTrip(req)
		require.Error(t, err)
		require.ErrorIs(t, err, sentinel)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "auth source:")
		assert.Nil(t, capture.seen, "base should not be invoked when auth source errs")
	})

	t.Run("nil auth source passes through", func(t *testing.T) {
		t.Parallel()

		capture := &captureRT{}
		rt := &AuthRoundTripper{Base: capture}

		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "http://example.test", http.NoBody)
		require.NoError(t, err)

		resp, err := rt.RoundTrip(req)
		require.NoError(t, err)
		_ = resp.Body.Close()

		require.NotNil(t, capture.seen)
		assert.Same(t, req, capture.seen, "request should pass through untouched when no auth source is set")
		assert.Empty(t, capture.seen.Header.Get("Authorization"))
	})

	t.Run("nil base falls back to default transport", func(t *testing.T) {
		t.Parallel()

		var seenAuth string
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			seenAuth = r.Header.Get("Authorization")
			w.WriteHeader(http.StatusOK)
		}))
		t.Cleanup(srv.Close)

		rt := &AuthRoundTripper{
			AuthSource: &fakeAuthSource{key: "Authorization", value: "Bearer fallback"},
		}

		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, srv.URL, http.NoBody)
		require.NoError(t, err)

		resp, err := rt.RoundTrip(req)
		require.NoError(t, err)
		_ = resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "Bearer fallback", seenAuth)
	})
}

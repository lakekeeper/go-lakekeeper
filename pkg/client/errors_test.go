package client

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	managementv1 "github.com/lakekeeper/go-lakekeeper/pkg/apis/management/v1"
)

func TestWrapAPIError_NilReturnsNil(t *testing.T) {
	t.Parallel()
	require.NoError(t, wrapAPIError("noop", nil))
}

func TestWrapAPIError_NonAPIErrorWrappedWithW(t *testing.T) {
	t.Parallel()
	base := errors.New("boom")
	got := wrapAPIError("step", base)
	require.EqualError(t, got, "step: boom")
	require.ErrorIs(t, got, base, "non-API errors must remain unwrappable")
}

// Drives the generated client against a fake server that returns a non-JSON
// 4xx body — the exact shape that produced the original "undefined response
// type" symptom — and confirms wrapAPIError surfaces the body in the error
// string.
func TestWrapAPIError_SurfacesServerBody(t *testing.T) {
	t.Parallel()

	body := `boom in plain text`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()

	cfg := managementv1.NewConfiguration()
	cfg.Servers = managementv1.ServerConfigurations{{URL: srv.URL}}
	api := managementv1.NewAPIClient(cfg)

	_, _, err := api.ServerAPI.GetServerInfo(t.Context()).Execute()
	require.Error(t, err)

	wrapped := wrapAPIError("server info", err)
	require.ErrorContains(t, wrapped, "server info:")
	require.ErrorContains(t, wrapped, body)
}

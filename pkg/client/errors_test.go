package client_test

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/stretchr/testify/require"

	managementv1 "github.com/lakekeeper/go-lakekeeper/pkg/apis/management/v1"
	"github.com/lakekeeper/go-lakekeeper/pkg/client"
	"github.com/lakekeeper/go-lakekeeper/pkg/testutil"
)

const roleNotFoundBody = `{"error":{"code":404,"message":"Role 1234 not found","type":"RoleNotFound","stack":[]}}`

func TestFacadeError_JSONBodyCarriesStatus(t *testing.T) {
	t.Parallel()
	mux, c := testutil.ServerMux(t)
	mux.HandleFunc("GET /management/v1/role/{id}", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(roleNotFoundBody))
	})

	_, err := c.Roles.Get(t.Context(), "1234", client.WithProject("p"))

	var apiErr *client.APIError
	require.ErrorAs(t, err, &apiErr, "want *client.APIError, got %T: %v", err, err)
	require.Equal(t, http.StatusNotFound, apiErr.StatusCode)
}

func TestFacadeError_JSONBodyDecodedIntoErrorModel(t *testing.T) {
	t.Parallel()
	mux, c := testutil.ServerMux(t)
	mux.HandleFunc("GET /management/v1/role/{id}", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(roleNotFoundBody))
	})

	_, err := c.Roles.Get(t.Context(), "1234", client.WithProject("p"))

	var apiErr *client.APIError
	require.ErrorAs(t, err, &apiErr)
	require.NotNil(t, apiErr.Model, "a JSON error body must be decoded")
	require.Equal(t, "RoleNotFound", apiErr.Model.GetType())
	require.Equal(t, "Role 1234 not found", apiErr.Model.GetMessage())
	require.EqualValues(t, 404, apiErr.Model.GetCode())
}

func TestFacadeError_JSONBodyReachesGeneratedError(t *testing.T) {
	t.Parallel()
	mux, c := testutil.ServerMux(t)
	mux.HandleFunc("GET /management/v1/role/{id}", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(roleNotFoundBody))
	})

	_, err := c.Roles.Get(t.Context(), "1234", client.WithProject("p"))

	var genErr *managementv1.GenericOpenAPIError
	require.ErrorAs(t, err, &genErr, "errors.As must reach the generated error, got %T: %v", err, err)
	require.JSONEq(t, roleNotFoundBody, string(genErr.Body()))
}

func TestFacadeError_TextShowsOperationStatusAndServerMessage(t *testing.T) {
	t.Parallel()
	mux, c := testutil.ServerMux(t)
	mux.HandleFunc("GET /management/v1/role/{id}", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(roleNotFoundBody))
	})

	_, err := c.Roles.Get(t.Context(), "1234", client.WithProject("p"))

	require.ErrorContains(t, err, "get role: 404 Not Found")
	require.ErrorContains(t, err, "Role 1234 not found")
}

func TestFacadeError_LongBodyTruncatedInTextButKeptWhole(t *testing.T) {
	t.Parallel()
	long := strings.Repeat("x", 5000)
	mux, c := testutil.ServerMux(t)
	mux.HandleFunc("GET /management/v1/role/{id}", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(long))
	})

	_, err := c.Roles.Get(t.Context(), "1234", client.WithProject("p"))

	require.Less(t, len(err.Error()), 2000, "error text must truncate the body")
	require.ErrorContains(t, err, "...(truncated)")
	var apiErr *client.APIError
	require.ErrorAs(t, err, &apiErr)
	require.Equal(t, http.StatusBadGateway, apiErr.StatusCode)
	require.Len(t, apiErr.Body, len(long), "Body keeps the full response")
	require.Nil(t, apiErr.Model, "a non-JSON body has no error model")
}

func TestFacadeError_TruncationKeepsValidUTF8(t *testing.T) {
	t.Parallel()
	// Two ASCII bytes, then 3-byte runes: byte 1024 falls inside a rune.
	long := "xx" + strings.Repeat("€", 1000)
	mux, c := testutil.ServerMux(t)
	mux.HandleFunc("GET /management/v1/role/{id}", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(long))
	})

	_, err := c.Roles.Get(t.Context(), "1234", client.WithProject("p"))

	require.ErrorContains(t, err, "...(truncated)")
	require.True(t, utf8.ValidString(err.Error()), "error text must stay valid UTF-8")
}

func TestFacadeError_NoBody(t *testing.T) {
	t.Parallel()
	mux, c := testutil.ServerMux(t)
	mux.HandleFunc("DELETE /management/v1/role/{id}", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	})

	err := c.Roles.Delete(t.Context(), "1234", client.WithProject("p"))

	require.EqualError(t, err, "delete role: 403 Forbidden")
	var apiErr *client.APIError
	require.ErrorAs(t, err, &apiErr)
	require.Equal(t, http.StatusForbidden, apiErr.StatusCode)
	require.Empty(t, apiErr.Body)
	require.Nil(t, apiErr.Model)
	var genErr *managementv1.GenericOpenAPIError
	require.ErrorAs(t, err, &genErr, "errors.As must reach the generated error")
}

func TestFacadeError_NonJSONClientErrorKeepsDecodeCause(t *testing.T) {
	t.Parallel()
	mux, c := testutil.ServerMux(t)
	mux.HandleFunc("GET /management/v1/role/{id}", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte("<html>proxy login</html>"))
	})

	_, err := c.Roles.Get(t.Context(), "1234", client.WithProject("p"))

	var apiErr *client.APIError
	require.ErrorAs(t, err, &apiErr)
	require.Equal(t, http.StatusUnauthorized, apiErr.StatusCode)
	require.ErrorContains(t, err, "get role: 401 Unauthorized (")
	require.ErrorContains(t, err, "invalid character '<'", "the decode error must stay in the text")
	require.ErrorContains(t, err, "<html>proxy login</html>")
}

func TestFacadeError_UndecodableSuccessIsNotAPIError(t *testing.T) {
	t.Parallel()
	mux, c := testutil.ServerMux(t)
	mux.HandleFunc("GET /management/v1/role/{id}", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":42}`))
	})

	_, err := c.Roles.Get(t.Context(), "1234", client.WithProject("p"))

	require.Error(t, err)
	var apiErr *client.APIError
	require.NotErrorAs(t, err, &apiErr, "a 2xx is not an error status")
	require.ErrorContains(t, err, "get role: decode 200 OK response: ")
	var genErr *managementv1.GenericOpenAPIError
	require.ErrorAs(t, err, &genErr, "the generated error stays in the chain")
	require.JSONEq(t, `{"id":42}`, string(genErr.Body()))
}

func TestFacadeError_RetriesExhaustedStillAPIError(t *testing.T) {
	t.Parallel()
	var attempts atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		attempts.Add(1)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(`{"error":{"code":503,"message":"try later","type":"Unavailable","stack":[]}}`))
	}))
	t.Cleanup(srv.Close)

	c, err := client.New(t.Context(), srv.URL, "tok",
		client.WithRetryMax(1), client.WithRetryWait(time.Millisecond, time.Millisecond))
	require.NoError(t, err)

	_, err = c.Roles.Get(t.Context(), "1234", client.WithProject("p"))

	require.Equal(t, int32(2), attempts.Load(), "the request must have been retried")
	var apiErr *client.APIError
	require.ErrorAs(t, err, &apiErr, "want *client.APIError, got %T: %v", err, err)
	require.Equal(t, http.StatusServiceUnavailable, apiErr.StatusCode)
	require.Contains(t, string(apiErr.Body), "try later")
	require.NotNil(t, apiErr.Model, "a 5xx JSON body is decoded too")
	require.Equal(t, "Unavailable", apiErr.Model.Type)
}

func TestFacadeError_CancelledContextKeepsChain(t *testing.T) {
	t.Parallel()
	_, c := testutil.ServerMux(t)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := c.Roles.Get(ctx, "1234", client.WithProject("p"))

	require.ErrorIs(t, err, context.Canceled)
	require.ErrorContains(t, err, "get role: ")
	var apiErr *client.APIError
	require.NotErrorAs(t, err, &apiErr, "a non-API error must not become *client.APIError")
}

func TestFacadeError_TransportFailureKeepsChain(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.NotFoundHandler())
	srv.Close() // nothing listens any more
	c, err := client.New(t.Context(), srv.URL, "tok", client.WithoutRetries())
	require.NoError(t, err)

	_, err = c.Roles.Get(t.Context(), "1234", client.WithProject("p"))

	var opErr *net.OpError
	require.ErrorAs(t, err, &opErr, "transport errors must stay reachable")
	require.ErrorContains(t, err, "get role: ")
}

// generatedGetRole drives the generated client (the escape hatch) against a
// server that answers with the role-not-found body.
func generatedGetRole(t *testing.T) (*http.Response, error) {
	t.Helper()
	mux, c := testutil.ServerMux(t)
	mux.HandleFunc("GET /management/v1/role/{id}", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(roleNotFoundBody))
	})
	_, resp, err := c.RoleAPI.GetRole(t.Context(), "1234").Execute()
	require.Error(t, err)
	return resp, err
}

func TestWrapAPIError_EscapeHatchWithResponse(t *testing.T) {
	t.Parallel()
	resp, err := generatedGetRole(t)

	wrapped := client.WrapAPIError("get role", err, resp)

	var apiErr *client.APIError
	require.ErrorAs(t, wrapped, &apiErr)
	require.Equal(t, http.StatusNotFound, apiErr.StatusCode)
	require.Equal(t, "RoleNotFound", apiErr.Model.GetType())
	require.ErrorContains(t, wrapped, "get role: 404 Not Found")
}

func TestWrapAPIError_EscapeHatchWithoutResponse(t *testing.T) {
	t.Parallel()
	_, err := generatedGetRole(t)

	wrapped := client.WrapAPIError("get role", err, nil)

	var apiErr *client.APIError
	require.ErrorAs(t, wrapped, &apiErr)
	require.Zero(t, apiErr.StatusCode, "status is unknown without a response")
	require.Equal(t, "RoleNotFound", apiErr.Model.GetType())
	require.ErrorContains(t, wrapped, "Role 1234 not found")
	var genErr *managementv1.GenericOpenAPIError
	require.ErrorAs(t, wrapped, &genErr)
}

func TestWrapAPIError_NilIsNil(t *testing.T) {
	t.Parallel()
	require.NoError(t, client.WrapAPIError("noop", nil, nil))
}

func TestInitialBootstrapError_IsAPIError(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("GET /management/v1/info", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"code":401,"message":"bad token","type":"Unauthorized"}}`))
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	_, err := client.New(t.Context(), srv.URL, "tok", client.WithoutRetries(),
		client.WithInitialBootstrap(true, true, nil))

	var apiErr *client.APIError
	require.ErrorAs(t, err, &apiErr)
	require.Equal(t, http.StatusUnauthorized, apiErr.StatusCode)
	require.ErrorContains(t, err, "bad token")
	require.True(t, strings.HasPrefix(err.Error(), "bootstrap: get server info: 401 Unauthorized"), "got %q", err)
}

func TestInitialBootstrapError_SinglePrefix(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("GET /management/v1/info", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// Marshal through the SDK so the body has every required field.
		_ = json.NewEncoder(w).Encode(managementv1.ServerInfo{
			ServerId: "srv", Version: "0.15.0", LakekeeperVersion: "0.15.0",
			AuthzBackend: "openfga", ManagedRoleProviders: []string{}, Queues: []string{},
		})
	})
	mux.HandleFunc("POST /management/v1/bootstrap", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte(`{"error":{"code":409,"message":"already bootstrapped","type":"Conflict"}}`))
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	_, err := client.New(t.Context(), srv.URL, "tok", client.WithoutRetries(),
		client.WithInitialBootstrap(true, true, nil))

	require.Error(t, err)
	require.True(t, strings.HasPrefix(err.Error(), "bootstrap: 409 Conflict"), "got %q", err)
	require.NotContains(t, err.Error(), "bootstrap: bootstrap:")
}

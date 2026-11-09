package client_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	managementv1 "github.com/lakekeeper/go-lakekeeper/pkg/apis/management/v1"
	"github.com/lakekeeper/go-lakekeeper/pkg/client"
	"github.com/lakekeeper/go-lakekeeper/pkg/testutil"
)

func TestServer_Info_ErrorIsAPIError(t *testing.T) {
	t.Parallel()
	mux, c := testutil.ServerMux(t)
	testutil.Respond(t, mux, "GET /management/v1/info", http.StatusUnauthorized,
		`{"error":{"code":401,"message":"bad token","type":"Unauthorized","stack":[]}}`)

	_, err := c.Server.Info(t.Context())

	require.ErrorContains(t, err, "get server info: 401 Unauthorized")
	var apiErr *client.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, http.StatusUnauthorized, apiErr.StatusCode)
	require.NotNil(t, apiErr.Model)
	assert.Equal(t, "Unauthorized", apiErr.Model.GetType())
	var genErr *managementv1.GenericOpenAPIError
	require.ErrorAs(t, err, &genErr, "errors.As must reach the generated error")
}

// serverInfoBody returns a minimal GET /management/v1/info response; every
// required ServerInfo property is present.
func serverInfoBody(bootstrapped bool) map[string]any {
	return map[string]any{
		"authz-backend":                   "openfga",
		"aws-system-identities-enabled":   false,
		"azure-system-identities-enabled": false,
		"bootstrapped":                    bootstrapped,
		"default-project-id":              "00000000-0000-0000-0000-000000000000",
		"gcp-system-identities-enabled":   true,
		"lakekeeper-version":              "0.15.0",
		"license-status":                  map[string]any{"license-type": "none", "valid": true},
		"managed-role-providers":          []string{},
		"queues":                          []string{"tabular_expiration"},
		"server-id":                       "srv-1",
		"version":                         "0.15.0",
	}
}

func TestServer_Info_ReturnsServerInfo(t *testing.T) {
	t.Parallel()
	mux, c := testutil.ServerMux(t)
	reqs := testutil.Respond(t, mux, "GET /management/v1/info", http.StatusOK, serverInfoBody(true))

	got, err := c.Server.Info(t.Context())

	require.NoError(t, err)
	assert.NotContains(t, reqs.Last(t).Header, "X-Project-Id")
	assert.Equal(t, "srv-1", got.GetServerId())
	assert.True(t, got.GetBootstrapped())
	assert.Equal(t, "openfga", got.GetAuthzBackend())
	assert.Equal(t, "00000000-0000-0000-0000-000000000000", got.GetDefaultProjectId())
	assert.True(t, got.GetGcpSystemIdentitiesEnabled())
	assert.Equal(t, []string{"tabular_expiration"}, got.GetQueues())
}

func TestServer_Bootstrap_SendsRequest(t *testing.T) {
	t.Parallel()
	mux, c := testutil.ServerMux(t)
	reqs := testutil.Respond(t, mux, "POST /management/v1/bootstrap", http.StatusNoContent, nil)
	req := managementv1.NewBootstrapRequest(true)
	req.SetIsOperator(true)
	req.SetUserType(managementv1.USERTYPE_APPLICATION)

	err := c.Server.Bootstrap(t.Context(), req)

	require.NoError(t, err)
	got := reqs.Last(t)
	assert.NotContains(t, got.Header, "X-Project-Id")
	assert.JSONEq(t, `{"accept-terms-of-use":true,"is-operator":true,"user-type":"application"}`, string(got.Body))
}

func TestServer_Bootstrap_NilRequestRejected(t *testing.T) {
	t.Parallel()
	mux, c := testutil.ServerMux(t)
	reqs := testutil.Respond(t, mux, "POST /management/v1/bootstrap", http.StatusNoContent, nil)

	err := c.Server.Bootstrap(t.Context(), nil)

	require.ErrorContains(t, err, "bootstrap: request must not be nil")
	assert.Equal(t, 0, reqs.Len(), "no request may reach the server")
}

func TestServer_Bootstrap_ErrorIsAPIError(t *testing.T) {
	t.Parallel()
	mux, c := testutil.ServerMux(t)
	testutil.Respond(t, mux, "POST /management/v1/bootstrap", http.StatusBadRequest,
		`{"error":{"code":400,"message":"Catalog already bootstrapped","type":"CatalogAlreadyBootstrapped","stack":[]}}`)

	err := c.Server.Bootstrap(t.Context(), managementv1.NewBootstrapRequest(true))

	require.ErrorContains(t, err, "bootstrap: 400 Bad Request")
	var apiErr *client.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, http.StatusBadRequest, apiErr.StatusCode)
	require.NotNil(t, apiErr.Model)
	assert.Equal(t, "CatalogAlreadyBootstrapped", apiErr.Model.GetType())
}

func TestNew_InitialBootstrap_SkipsBootstrappedServer(t *testing.T) {
	t.Parallel()
	mux, baseURL := testutil.NewServer(t)
	info := testutil.Respond(t, mux, "GET /management/v1/info", http.StatusOK, serverInfoBody(true))
	boot := testutil.Respond(t, mux, "POST /management/v1/bootstrap", http.StatusNoContent, nil)

	_, err := client.New(t.Context(), baseURL, "test-token", client.WithoutRetries(),
		client.WithInitialBootstrap(true, true, nil))

	require.NoError(t, err)
	assert.Equal(t, 1, info.Len())
	assert.Equal(t, 0, boot.Len(), "an already bootstrapped server must not be bootstrapped again")
}

func TestNew_InitialBootstrap_BootstrapsFreshServer(t *testing.T) {
	t.Parallel()
	mux, baseURL := testutil.NewServer(t)
	testutil.Respond(t, mux, "GET /management/v1/info", http.StatusOK, serverInfoBody(false))
	boot := testutil.Respond(t, mux, "POST /management/v1/bootstrap", http.StatusNoContent, nil)
	userType := managementv1.USERTYPE_HUMAN

	_, err := client.New(t.Context(), baseURL, "test-token", client.WithoutRetries(),
		client.WithInitialBootstrap(true, false, &userType))

	require.NoError(t, err)
	require.Equal(t, 1, boot.Len())
	assert.JSONEq(t, `{"accept-terms-of-use":true,"is-operator":false,"user-type":"human"}`,
		string(boot.Last(t).Body))
}

func TestNew_InitialBootstrap_BootstrapErrorFailsConstruction(t *testing.T) {
	t.Parallel()
	mux, baseURL := testutil.NewServer(t)
	testutil.Respond(t, mux, "GET /management/v1/info", http.StatusOK, serverInfoBody(false))
	testutil.Respond(t, mux, "POST /management/v1/bootstrap", http.StatusForbidden,
		`{"error":{"code":403,"message":"no","type":"Forbidden","stack":[]}}`)

	_, err := client.New(t.Context(), baseURL, "test-token", client.WithoutRetries(),
		client.WithInitialBootstrap(true, true, nil))

	require.ErrorContains(t, err, "bootstrap: bootstrap: 403 Forbidden")
	var apiErr *client.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, http.StatusForbidden, apiErr.StatusCode)
}

func TestNew_InitialBootstrap_TermsNotAcceptedSkipsBootstrap(t *testing.T) {
	t.Parallel()
	mux, baseURL := testutil.NewServer(t)
	info := testutil.Respond(t, mux, "GET /management/v1/info", http.StatusOK, serverInfoBody(false))
	boot := testutil.Respond(t, mux, "POST /management/v1/bootstrap", http.StatusNoContent, nil)

	_, err := client.New(t.Context(), baseURL, "test-token", client.WithoutRetries(),
		client.WithInitialBootstrap(false, true, nil))

	require.NoError(t, err)
	assert.Equal(t, 0, info.Len())
	assert.Equal(t, 0, boot.Len())
}

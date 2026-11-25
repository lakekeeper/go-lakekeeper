package client_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lakekeeper/go-lakekeeper/pkg/client"
	"github.com/lakekeeper/go-lakekeeper/pkg/testutil"
)

// lakekeeperInfo is the server info of a Lakekeeper 0.15.1 server running
// OpenFGA: lakekeeper-enterprise-version is null.
func lakekeeperInfo() map[string]any {
	info := serverInfoBody(true)
	info["version"] = "0.15.1"
	info["lakekeeper-version"] = "0.15.1"
	info["lakekeeper-enterprise-version"] = nil
	return info
}

// lakekeeperPlusInfo is the server info of a Lakekeeper+ 0.15.2 server, on
// Lakekeeper 0.15.1, running the given authorization backend.
func lakekeeperPlusInfo(authzBackend string) map[string]any {
	info := serverInfoBody(true)
	info["version"] = "0.15.1"
	info["lakekeeper-version"] = "0.15.1"
	info["lakekeeper-enterprise-version"] = "0.15.2"
	info["authz-backend"] = authzBackend
	info["queues"] = []string{"expire_snapshots", "tabular_expiration"}
	return info
}

func TestCapabilities_Lakekeeper(t *testing.T) {
	t.Parallel()
	mux, c := testutil.ServerMux(t)
	testutil.Respond(t, mux, "GET /management/v1/info", http.StatusOK, lakekeeperInfo())

	got, err := c.Capabilities(t.Context())

	require.NoError(t, err)
	assert.Equal(t, &client.Capabilities{
		Edition:           client.EditionLakekeeper,
		LakekeeperVersion: "0.15.1",
		AuthzBackend:      "openfga",
		Queues:            []string{"tabular_expiration"},
	}, got)
	assert.Equal(t, "Lakekeeper 0.15.1", got.String())
}

func TestCapabilities_LakekeeperPlus(t *testing.T) {
	t.Parallel()
	mux, c := testutil.ServerMux(t)
	testutil.Respond(t, mux, "GET /management/v1/info", http.StatusOK, lakekeeperPlusInfo("cedar"))

	got, err := c.Capabilities(t.Context())

	require.NoError(t, err)
	assert.Equal(t, &client.Capabilities{
		Edition:               client.EditionLakekeeperPlus,
		LakekeeperVersion:     "0.15.1",
		LakekeeperPlusVersion: "0.15.2",
		AuthzBackend:          client.AuthzBackendCedar,
		Queues:                []string{"expire_snapshots", "tabular_expiration"},
	}, got)
	assert.Equal(t, "Lakekeeper+ 0.15.2", got.String())
}

func TestCapabilities_FetchedOncePerClient(t *testing.T) {
	t.Parallel()
	mux, c := testutil.ServerMux(t)
	info := testutil.Respond(t, mux, "GET /management/v1/info", http.StatusOK, lakekeeperInfo())

	first, err := c.Capabilities(t.Context())
	require.NoError(t, err)
	first.Queues[0] = "changed by the caller"
	second, err := c.Capabilities(t.Context())
	require.NoError(t, err)

	assert.Equal(t, 1, info.Len(), "server info requests")
	assert.Equal(t, []string{"tabular_expiration"}, second.Queues, "the cached value is not shared")
}

func TestCapabilities_FailedFetchIsNotCached(t *testing.T) {
	t.Parallel()
	mux, c := testutil.ServerMux(t)
	calls := 0
	mux.HandleFunc("GET /management/v1/info", func(w http.ResponseWriter, _ *http.Request) {
		calls++
		if calls == 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		assert.NoError(t, json.NewEncoder(w).Encode(lakekeeperInfo()))
	})

	_, err := c.Capabilities(t.Context())
	require.ErrorContains(t, err, "capabilities: get server info: 503")
	got, err := c.Capabilities(t.Context())

	require.NoError(t, err)
	assert.Equal(t, client.EditionLakekeeper, got.Edition)
}

// An older server's info can lack a field the generated model requires. The
// capabilities do not need it, so they are still read.
func TestCapabilities_FromServerInfoTheModelCannotDecode(t *testing.T) {
	t.Parallel()
	mux, c := testutil.ServerMux(t)
	info := testutil.Respond(t, mux, "GET /management/v1/info", http.StatusOK,
		`{"version":"0.13.9","lakekeeper-version":"0.13.9","authz-backend":"openfga","queues":["tabular_purge"]}`)

	_, infoErr := c.Server.Info(t.Context())
	got, err := c.Capabilities(t.Context())

	require.Error(t, infoErr, "the model still rejects the info")
	require.NoError(t, err)
	assert.Equal(t, &client.Capabilities{
		Edition:           client.EditionLakekeeper,
		LakekeeperVersion: "0.13.9",
		AuthzBackend:      "openfga",
		Queues:            []string{"tabular_purge"},
	}, got)
	assert.Equal(t, 1, info.Len(), "Server.Info fills the capabilities; no second request")
}

// Server.Info fills the capabilities, so a later Capabilities call or
// diagnosis costs no request.
func TestCapabilities_AfterServerInfoCostNoRequest(t *testing.T) {
	t.Parallel()
	mux, c := testutil.ServerMux(t)
	info := testutil.Respond(t, mux, "GET /management/v1/info", http.StatusOK, lakekeeperInfo())

	_, err := c.Server.Info(t.Context())
	require.NoError(t, err)
	got, err := c.Capabilities(t.Context())

	require.NoError(t, err)
	assert.Equal(t, "Lakekeeper 0.15.1", got.String())
	assert.Equal(t, 1, info.Len(), "server info requests")
}

// CachedCapabilities returns what an earlier server info fetch read, and
// sends no request of its own.
func TestCachedCapabilities_AfterServerInfoWithoutARequest(t *testing.T) {
	t.Parallel()
	mux, c := testutil.ServerMux(t)
	info := testutil.Respond(t, mux, "GET /management/v1/info", http.StatusOK, lakekeeperInfo())

	_, err := c.Server.Info(t.Context())
	require.NoError(t, err)
	got, ok := c.CachedCapabilities()

	require.True(t, ok)
	assert.Equal(t, "Lakekeeper 0.15.1", got.String())
	assert.Equal(t, 1, info.Len(), "server info requests")
}

func TestCachedCapabilities_NoneBeforeAServerInfoFetch(t *testing.T) {
	t.Parallel()
	mux, c := testutil.ServerMux(t)
	info := testutil.Respond(t, mux, "GET /management/v1/info", http.StatusOK, lakekeeperInfo())
	testutil.Respond(t, mux, "GET /management/v1/project-list", http.StatusOK, `{"projects":[]}`)

	_, err := c.Projects.List(t.Context())
	require.NoError(t, err)
	got, ok := c.CachedCapabilities()

	assert.False(t, ok)
	assert.Nil(t, got)
	assert.Zero(t, info.Len(), "server info requests")
}

// A server info body the SDK cannot decode still names the server.
func TestCachedCapabilities_FromAnUndecodableServerInfo(t *testing.T) {
	t.Parallel()
	mux, c := testutil.ServerMux(t)
	testutil.Respond(t, mux, "GET /management/v1/info", http.StatusOK,
		`{"version":"0.13.9","lakekeeper-version":"0.13.9","bootstrapped":true}`)

	_, err := c.Server.Info(t.Context())
	require.Error(t, err)
	got, ok := c.CachedCapabilities()

	require.True(t, ok)
	assert.Equal(t, "Lakekeeper 0.13.9", got.String())
}

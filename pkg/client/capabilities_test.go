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

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

// lakekeeperPlusInfoVersion is lakekeeperPlusInfo with the Cedar backend on
// the given Lakekeeper+ version.
func lakekeeperPlusInfoVersion(plusVersion string) map[string]any {
	info := lakekeeperPlusInfo(client.AuthzBackendCedar)
	info["lakekeeper-enterprise-version"] = plusVersion
	return info
}

// A server that meets every other requirement but is older than the release
// of the spec the operation came from answers an unknown route with a 404
// and no error body: the server is too old for the operation.
func TestCedar_Schema_OnOlderServerIsServerTooOld(t *testing.T) {
	t.Parallel()
	mux, c := testutil.ServerMux(t)
	testutil.Respond(t, mux, cedarSchemaPath, http.StatusNotFound, nil)
	testutil.Respond(t, mux, "GET /management/v1/info", http.StatusOK, lakekeeperPlusInfoVersion("0.1.0"))

	_, err := c.Cedar.Schema(t.Context())

	require.ErrorIs(t, err, client.ErrUnsupported)
	require.EqualError(t, err, "get cedar schema: requires Lakekeeper+ "+managementv1.LakekeeperPlusSpecRelease+
		" or later; server is Lakekeeper+ 0.1.0")
	var unsupported *client.UnsupportedError
	require.ErrorAs(t, err, &unsupported)
	assert.Equal(t, client.ReasonServerTooOld, unsupported.Reason)
	assert.Equal(t, "0.1.0", unsupported.Server.LakekeeperPlusVersion)
}

// A 404 with a Lakekeeper error body is the server answering a route it
// knows: a genuine not-found, even on an older server.
func TestCedar_Schema_NotFoundWithErrorBodyOnOlderServerStaysAPIError(t *testing.T) {
	t.Parallel()
	mux, c := testutil.ServerMux(t)
	testutil.Respond(t, mux, cedarSchemaPath, http.StatusNotFound,
		`{"error":{"code":404,"message":"not found","type":"NotFound","stack":[]}}`)
	testutil.Respond(t, mux, "GET /management/v1/info", http.StatusOK, lakekeeperPlusInfoVersion("0.1.0"))

	_, err := c.Cedar.Schema(t.Context())

	require.NotErrorIs(t, err, client.ErrUnsupported)
	var apiErr *client.APIError
	require.ErrorAs(t, err, &apiErr)
}

// The spec release itself is not too old: an empty 404 there stays as it is.
func TestCedar_Schema_NotFoundOnSpecReleaseIsNotServerTooOld(t *testing.T) {
	t.Parallel()
	mux, c := testutil.ServerMux(t)
	testutil.Respond(t, mux, cedarSchemaPath, http.StatusNotFound, nil)
	testutil.Respond(t, mux, "GET /management/v1/info", http.StatusOK,
		lakekeeperPlusInfoVersion(managementv1.LakekeeperPlusSpecRelease))

	_, err := c.Cedar.Schema(t.Context())

	require.NotErrorIs(t, err, client.ErrUnsupported)
}

const cedarSchemaPath = "GET /management/v1/permissions/cedar/schema"

func TestCedar_Schema_ReturnsSchemaWithoutConsultingCapabilities(t *testing.T) {
	t.Parallel()
	mux, c := testutil.ServerMux(t)
	schema := testutil.Respond(t, mux, cedarSchemaPath, http.StatusOK,
		`{"schema-fragments":{"/etc/lakekeeper/schema.cedarschema":"namespace Lakekeeper {}"}}`)
	info := testutil.Respond(t, mux, "GET /management/v1/info", http.StatusOK, lakekeeperPlusInfo("cedar"))

	got, err := c.Cedar.Schema(t.Context())

	require.NoError(t, err)
	assert.Equal(t, map[string]string{"/etc/lakekeeper/schema.cedarschema": "namespace Lakekeeper {}"},
		got.GetSchemaFragments())
	assert.Equal(t, 1, schema.Len(), "schema requests")
	assert.Zero(t, info.Len(), "server info requests")
}

func TestCedar_Schema_OnLakekeeperIsUnsupported(t *testing.T) {
	t.Parallel()
	mux, c := testutil.ServerMux(t)
	testutil.Respond(t, mux, cedarSchemaPath, http.StatusNotFound, nil)
	testutil.Respond(t, mux, "GET /management/v1/info", http.StatusOK, lakekeeperInfo())

	_, err := c.Cedar.Schema(t.Context())

	require.ErrorIs(t, err, client.ErrUnsupported)
	require.EqualError(t, err, "get cedar schema: requires Lakekeeper+; server is Lakekeeper 0.15.1")
	var unsupported *client.UnsupportedError
	require.ErrorAs(t, err, &unsupported)
	assert.Equal(t, client.ReasonNotLakekeeperPlus, unsupported.Reason)
	assert.Equal(t, "Cedar", unsupported.Feature)
	assert.Equal(t, client.EditionLakekeeper, unsupported.Server.Edition)
	var apiErr *client.APIError
	assert.NotErrorAs(t, err, &apiErr, "an unsupported operation must not read as a not-found")
}

func TestCedar_Schema_OnOtherAuthzBackendIsUnsupported(t *testing.T) {
	t.Parallel()
	mux, c := testutil.ServerMux(t)
	testutil.Respond(t, mux, cedarSchemaPath, http.StatusNotFound, nil)
	testutil.Respond(t, mux, "GET /management/v1/info", http.StatusOK, lakekeeperPlusInfo("openfga"))

	_, err := c.Cedar.Schema(t.Context())

	require.EqualError(t, err, "get cedar schema: requires the cedar authorization backend; "+
		"server is Lakekeeper+ 0.15.2 with the openfga authorization backend")
	var unsupported *client.UnsupportedError
	require.ErrorAs(t, err, &unsupported)
	assert.Equal(t, client.ReasonAuthzBackend, unsupported.Reason)
}

func TestCedar_Schema_MethodNotAllowedIsDiagnosedToo(t *testing.T) {
	t.Parallel()
	mux, c := testutil.ServerMux(t)
	testutil.Respond(t, mux, cedarSchemaPath, http.StatusMethodNotAllowed, nil)
	testutil.Respond(t, mux, "GET /management/v1/info", http.StatusOK, lakekeeperInfo())

	_, err := c.Cedar.Schema(t.Context())

	require.ErrorIs(t, err, client.ErrUnsupported)
}

// A 404 the capabilities do not explain is a genuine not-found: the server
// has everything the operation needs.
func TestCedar_Schema_NotFoundOnCapableServerStaysAPIError(t *testing.T) {
	t.Parallel()
	mux, c := testutil.ServerMux(t)
	testutil.Respond(t, mux, cedarSchemaPath, http.StatusNotFound,
		`{"error":{"code":404,"message":"not found","type":"NotFound","stack":[]}}`)
	testutil.Respond(t, mux, "GET /management/v1/info", http.StatusOK, lakekeeperPlusInfo("cedar"))

	_, err := c.Cedar.Schema(t.Context())

	require.NotErrorIs(t, err, client.ErrUnsupported)
	var apiErr *client.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, http.StatusNotFound, apiErr.StatusCode)
}

func TestCedar_Schema_OtherStatusIsNotDiagnosed(t *testing.T) {
	t.Parallel()
	mux, c := testutil.ServerMux(t)
	testutil.Respond(t, mux, cedarSchemaPath, http.StatusForbidden, nil)
	info := testutil.Respond(t, mux, "GET /management/v1/info", http.StatusOK, lakekeeperInfo())

	_, err := c.Cedar.Schema(t.Context())

	var apiErr *client.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, http.StatusForbidden, apiErr.StatusCode)
	assert.Zero(t, info.Len(), "server info requests")
}

// When the capabilities cannot be fetched, the caller gets the original
// error: diagnosis never replaces it with a different failure.
func TestCedar_Schema_FailedDiagnosisKeepsOriginalError(t *testing.T) {
	t.Parallel()
	mux, c := testutil.ServerMux(t)
	testutil.Respond(t, mux, cedarSchemaPath, http.StatusNotFound, nil)
	testutil.Respond(t, mux, "GET /management/v1/info", http.StatusInternalServerError, nil)

	_, err := c.Cedar.Schema(t.Context())

	var apiErr *client.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, http.StatusNotFound, apiErr.StatusCode)
}

func TestCedar_Schema_DiagnosesWithOneServerInfoFetch(t *testing.T) {
	t.Parallel()
	mux, c := testutil.ServerMux(t)
	testutil.Respond(t, mux, cedarSchemaPath, http.StatusNotFound, nil)
	info := testutil.Respond(t, mux, "GET /management/v1/info", http.StatusOK, lakekeeperInfo())

	_, err := c.Cedar.Schema(t.Context())
	require.ErrorIs(t, err, client.ErrUnsupported)
	_, err = c.Cedar.Schema(t.Context())
	require.ErrorIs(t, err, client.ErrUnsupported)

	assert.Equal(t, 1, info.Len(), "server info requests")
}

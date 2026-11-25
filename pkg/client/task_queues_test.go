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

const (
	whQueuePath      = "/management/v1/warehouse/wh-1/task-queue/"
	optimizeConfig   = whQueuePath + "optimize/config"
	expireSnapConfig = whQueuePath + "expire_snapshots/config"
)

// lakekeeperPlusInfo runs expire_snapshots but not optimize.
func TestTaskQueues_OptimizeConfig_QueueNotConfigured(t *testing.T) {
	t.Parallel()
	mux, c := testutil.ServerMux(t)
	testutil.Respond(t, mux, "GET "+optimizeConfig, http.StatusNotFound, nil)
	testutil.Respond(t, mux, "GET /management/v1/info", http.StatusOK, lakekeeperPlusInfo("cedar"))

	_, err := c.TaskQueues.OptimizeConfig(t.Context(), "wh-1")

	require.ErrorIs(t, err, client.ErrUnsupported)
	require.EqualError(t, err, "get optimize queue config: requires the optimize task queue; "+
		"server is Lakekeeper+ 0.15.2 without it")
	var unsupported *client.UnsupportedError
	require.ErrorAs(t, err, &unsupported)
	assert.Equal(t, client.ReasonQueueNotConfigured, unsupported.Reason)
	assert.Equal(t, "task queue optimize", unsupported.Feature)
}

func TestTaskQueues_OptimizeConfig_OnLakekeeperIsUnsupported(t *testing.T) {
	t.Parallel()
	mux, c := testutil.ServerMux(t)
	testutil.Respond(t, mux, "GET "+optimizeConfig, http.StatusNotFound, nil)
	testutil.Respond(t, mux, "GET /management/v1/info", http.StatusOK, lakekeeperInfo())

	_, err := c.TaskQueues.OptimizeConfig(t.Context(), "wh-1")

	require.EqualError(t, err, "get optimize queue config: requires Lakekeeper+; server is Lakekeeper 0.15.1")
	var unsupported *client.UnsupportedError
	require.ErrorAs(t, err, &unsupported)
	assert.Equal(t, client.ReasonNotLakekeeperPlus, unsupported.Reason)
}

// The server runs expire_snapshots, so its 404 is a missing warehouse.
func TestTaskQueues_ExpireSnapshotsConfig_MissingWarehouseStaysNotFound(t *testing.T) {
	t.Parallel()
	mux, c := testutil.ServerMux(t)
	testutil.Respond(t, mux, "GET "+expireSnapConfig, http.StatusNotFound,
		`{"error":{"code":404,"message":"Warehouse not found","type":"WarehouseNotFound","stack":[]}}`)
	testutil.Respond(t, mux, "GET /management/v1/info", http.StatusOK, lakekeeperPlusInfo("cedar"))

	_, err := c.TaskQueues.ExpireSnapshotsConfig(t.Context(), "wh-1")

	require.NotErrorIs(t, err, client.ErrUnsupported)
	var apiErr *client.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, http.StatusNotFound, apiErr.StatusCode)
}

func TestTaskQueues_SetExpireSnapshotsConfig_SendsOnlyTheFieldsSet(t *testing.T) {
	t.Parallel()
	mux, c := testutil.ServerMux(t)
	set := testutil.Respond(t, mux, "POST "+expireSnapConfig, http.StatusNoContent, nil)
	info := testutil.Respond(t, mux, "GET /management/v1/info", http.StatusOK, lakekeeperPlusInfo("cedar"))

	cfg := managementv1.ExpireSnapshotsQueueConfig{}
	cfg.SetMaxSnapshotAgeMs(604800000)
	err := c.TaskQueues.SetExpireSnapshotsConfig(t.Context(), "wh-1",
		&managementv1.SetExpireSnapshotsQueueConfig{QueueConfig: cfg})

	require.NoError(t, err)
	assert.JSONEq(t, `{"queue-config":{"max-snapshot-age-ms":604800000}}`, string(set.Last(t).Body))
	assert.Zero(t, info.Len(), "server info requests")
}

func TestTaskQueues_SetRemoveOrphanFilesConfig_QueueNotConfigured(t *testing.T) {
	t.Parallel()
	mux, c := testutil.ServerMux(t)
	testutil.Respond(t, mux, "POST "+whQueuePath+"remove_orphan_files/config", http.StatusNotFound, nil)
	testutil.Respond(t, mux, "GET /management/v1/info", http.StatusOK, lakekeeperPlusInfo("cedar"))

	err := c.TaskQueues.SetRemoveOrphanFilesConfig(t.Context(), "wh-1",
		&managementv1.SetRemoveOrphanFilesQueueConfig{})

	require.EqualError(t, err, "set remove_orphan_files queue config: requires the remove_orphan_files task queue; "+
		"server is Lakekeeper+ 0.15.2 without it")
}

// Lakekeeper answers a set for a queue it does not know with 400
// QueueNotFound rather than 404.
func TestTaskQueues_SetOptimizeConfig_QueueNotFoundOnLakekeeperIsUnsupported(t *testing.T) {
	t.Parallel()
	mux, c := testutil.ServerMux(t)
	testutil.Respond(t, mux, "POST "+optimizeConfig, http.StatusBadRequest,
		`{"error":{"message":"Queue 'optimize' not found! Configurable queues: [soft_deletion, tabular_purge, task_log_cleanup]",`+
			`"type":"QueueNotFound","code":400,"stack":[]}}`)
	testutil.Respond(t, mux, "GET /management/v1/info", http.StatusOK, lakekeeperInfo())

	err := c.TaskQueues.SetOptimizeConfig(t.Context(), "wh-1", &managementv1.SetOptimizeQueueConfig{})

	require.EqualError(t, err, "set optimize queue config: requires Lakekeeper+; server is Lakekeeper 0.15.1")
}

// Other 400s are the caller's error, not an unsupported operation.
func TestTaskQueues_SetOptimizeConfig_OtherBadRequestIsNotDiagnosed(t *testing.T) {
	t.Parallel()
	mux, c := testutil.ServerMux(t)
	testutil.Respond(t, mux, "POST "+optimizeConfig, http.StatusBadRequest,
		`{"error":{"message":"bad config","type":"InvalidQueueConfig","code":400,"stack":[]}}`)
	info := testutil.Respond(t, mux, "GET /management/v1/info", http.StatusOK, lakekeeperInfo())

	err := c.TaskQueues.SetOptimizeConfig(t.Context(), "wh-1", &managementv1.SetOptimizeQueueConfig{})

	var apiErr *client.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, http.StatusBadRequest, apiErr.StatusCode)
	assert.Zero(t, info.Len(), "server info requests")
}

func TestTaskQueues_SetNilRequestFailsWithoutRequest(t *testing.T) {
	t.Parallel()
	_, c := testutil.ServerMux(t)

	require.EqualError(t, c.TaskQueues.SetOptimizeConfig(t.Context(), "wh-1", nil),
		"set optimize queue config: request must not be nil")
}

// Lakekeeper queues are not gated: a 404 is returned as is, without a
// server info request.
func TestTaskQueues_TabularPurgeConfig_NotFoundIsNotDiagnosed(t *testing.T) {
	t.Parallel()
	mux, c := testutil.ServerMux(t)
	testutil.Respond(t, mux, "GET "+whQueuePath+"tabular_purge/config", http.StatusNotFound, nil)
	info := testutil.Respond(t, mux, "GET /management/v1/info", http.StatusOK, lakekeeperInfo())

	_, err := c.TaskQueues.TabularPurgeConfig(t.Context(), "wh-1")

	var apiErr *client.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, http.StatusNotFound, apiErr.StatusCode)
	assert.Zero(t, info.Len(), "server info requests")
}

func TestTaskQueues_SoftDeletionConfig_ReturnsConfig(t *testing.T) {
	t.Parallel()
	mux, c := testutil.ServerMux(t)
	testutil.Respond(t, mux, "GET "+whQueuePath+"soft_deletion/config", http.StatusOK,
		`{"max-seconds-since-last-heartbeat":3600,"queue-config":{}}`)

	got, err := c.TaskQueues.SoftDeletionConfig(t.Context(), "wh-1")

	require.NoError(t, err)
	assert.Equal(t, int64(3600), got.GetMaxSecondsSinceLastHeartbeat())
}

func TestTaskQueues_TaskLogCleanupConfig_IsProjectScoped(t *testing.T) {
	t.Parallel()
	mux, c := testutil.ServerMux(t)
	const path = "/management/v1/project/task-queue/task_log_cleanup/config"
	get := testutil.Respond(t, mux, "GET "+path, http.StatusOK,
		`{"queue-config":{"retention-period":"P30D"}}`)
	set := testutil.Respond(t, mux, "POST "+path, http.StatusNoContent, nil)

	got, err := c.TaskQueues.TaskLogCleanupConfig(t.Context(), client.WithProject("prj-1"))
	require.NoError(t, err)
	assert.Equal(t, "P30D", got.QueueConfig.GetRetentionPeriod())
	assert.Equal(t, "prj-1", get.Last(t).Header.Get("X-Project-Id"))

	cfg := managementv1.TaskLogCleanupConfig{}
	cfg.SetCleanupPeriod("P2D")
	err = c.TaskQueues.SetTaskLogCleanupConfig(t.Context(),
		&managementv1.SetTaskLogCleanupConfig{QueueConfig: cfg}, client.WithProject("prj-1"))
	require.NoError(t, err)
	assert.Equal(t, "prj-1", set.Last(t).Header.Get("X-Project-Id"))
	assert.JSONEq(t, `{"queue-config":{"cleanup-period":"P2D"}}`, string(set.Last(t).Body))
}

func tableEntity() managementv1.WarehouseTaskEntityId {
	return managementv1.WarehouseTaskEntityIdTableAsWarehouseTaskEntityId(
		managementv1.NewWarehouseTaskEntityIdTable("tbl-1", "table"))
}

func TestTaskQueues_ScheduleExpireSnapshots_ReturnsTaskID(t *testing.T) {
	t.Parallel()
	mux, c := testutil.ServerMux(t)
	schedule := testutil.Respond(t, mux, "POST "+whQueuePath+"expire_snapshots/schedule", http.StatusOK,
		`{"task-id":"task-1"}`)
	info := testutil.Respond(t, mux, "GET /management/v1/info", http.StatusOK, lakekeeperPlusInfo("cedar"))

	got, err := c.TaskQueues.ScheduleExpireSnapshots(t.Context(), "wh-1",
		managementv1.NewScheduleExpireSnapshotsTaskRequest(tableEntity()))

	require.NoError(t, err)
	assert.Equal(t, "task-1", got.TaskId)
	assert.JSONEq(t, `{"entity":{"type":"table","table-id":"tbl-1"}}`, string(schedule.Last(t).Body))
	assert.Zero(t, info.Len(), "server info requests")
}

func TestTaskQueues_ScheduleOptimize_SendsSteps(t *testing.T) {
	t.Parallel()
	mux, c := testutil.ServerMux(t)
	schedule := testutil.Respond(t, mux, "POST "+whQueuePath+"optimize/schedule", http.StatusOK,
		`{"task-id":"task-2"}`)

	req := managementv1.NewScheduleOptimizeTaskRequest(tableEntity())
	req.SetPayload(managementv1.ScheduleOptimizeTaskRequestPayload{
		Steps: []managementv1.OptimizeStep{managementv1.OPTIMIZESTEP_COMPACTION, managementv1.OPTIMIZESTEP_REWRITE_MANIFESTS},
	})
	got, err := c.TaskQueues.ScheduleOptimize(t.Context(), "wh-1", req)

	require.NoError(t, err)
	assert.Equal(t, "task-2", got.TaskId)
	assert.JSONEq(t,
		`{"entity":{"type":"table","table-id":"tbl-1"},"payload":{"steps":["compaction","rewrite-manifests"]}}`,
		string(schedule.Last(t).Body))
}

func TestTaskQueues_ScheduleOptimize_QueueNotConfigured(t *testing.T) {
	t.Parallel()
	mux, c := testutil.ServerMux(t)
	testutil.Respond(t, mux, "POST "+whQueuePath+"optimize/schedule", http.StatusNotFound, nil)
	testutil.Respond(t, mux, "GET /management/v1/info", http.StatusOK, lakekeeperPlusInfo("cedar"))

	_, err := c.TaskQueues.ScheduleOptimize(t.Context(), "wh-1",
		managementv1.NewScheduleOptimizeTaskRequest(tableEntity()))

	require.EqualError(t, err, "schedule optimize task: requires the optimize task queue; "+
		"server is Lakekeeper+ 0.15.2 without it")
}

func TestTaskQueues_ScheduleRemoveOrphanFiles_OnLakekeeperIsUnsupported(t *testing.T) {
	t.Parallel()
	mux, c := testutil.ServerMux(t)
	testutil.Respond(t, mux, "POST "+whQueuePath+"remove_orphan_files/schedule", http.StatusNotFound, nil)
	testutil.Respond(t, mux, "GET /management/v1/info", http.StatusOK, lakekeeperInfo())

	_, err := c.TaskQueues.ScheduleRemoveOrphanFiles(t.Context(), "wh-1",
		managementv1.NewScheduleRemoveOrphanFilesTaskRequest(tableEntity()))

	require.ErrorIs(t, err, client.ErrUnsupported)
	require.EqualError(t, err, "schedule remove_orphan_files task: requires Lakekeeper+; server is Lakekeeper 0.15.1")
}

// The server runs expire_snapshots, so its 404 is a missing table.
func TestTaskQueues_ScheduleExpireSnapshots_MissingTableStaysNotFound(t *testing.T) {
	t.Parallel()
	mux, c := testutil.ServerMux(t)
	testutil.Respond(t, mux, "POST "+whQueuePath+"expire_snapshots/schedule", http.StatusNotFound,
		`{"error":{"code":404,"message":"Table not found","type":"NoSuchTableException","stack":[]}}`)
	testutil.Respond(t, mux, "GET /management/v1/info", http.StatusOK, lakekeeperPlusInfo("cedar"))

	_, err := c.TaskQueues.ScheduleExpireSnapshots(t.Context(), "wh-1",
		managementv1.NewScheduleExpireSnapshotsTaskRequest(tableEntity()))

	require.NotErrorIs(t, err, client.ErrUnsupported)
	var apiErr *client.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, http.StatusNotFound, apiErr.StatusCode)
}

func TestTaskQueues_ScheduleNilRequestFailsWithoutRequest(t *testing.T) {
	t.Parallel()
	_, c := testutil.ServerMux(t)

	_, err := c.TaskQueues.ScheduleExpireSnapshots(t.Context(), "wh-1", nil)
	require.EqualError(t, err, "schedule expire_snapshots task: request must not be nil")
}

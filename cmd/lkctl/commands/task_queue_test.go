package commands

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lakekeeper/go-lakekeeper/pkg/testutil"
)

const whQueue = "/management/v1/warehouse/wh-1/task-queue/"

func TestTaskQueueHelp_MarksLakekeeperPlusQueuesWithoutContactingServer(t *testing.T) {
	t.Parallel()
	baseURL := noServer(t)

	help, err := runLkctl(t, baseURL, "task-queue", "--help")
	require.NoError(t, err)

	for _, plus := range []string{"expire-snapshots", "optimize", "remove-orphan-files"} {
		assert.Regexp(t, plus+`\s+.*\(Lakekeeper\+\)`, help)
	}
	for _, lakekeeper := range []string{"tabular-purge", "soft-deletion", "task-log-cleanup"} {
		assert.Regexp(t, lakekeeper+`\s+`, help)
		assert.NotRegexp(t, lakekeeper+`\s+.*\(Lakekeeper\+\)`, help)
	}
}

func TestTaskQueueSet_SendsOnlyTheFlagsGiven(t *testing.T) {
	t.Parallel()
	mux, baseURL := testutil.NewServer(t)
	set := testutil.Respond(t, mux, "POST "+whQueue+"expire_snapshots/config", http.StatusNoContent, nil)

	out, err := runLkctl(t, baseURL, "task-queue", "expire-snapshots", "set", "wh-1",
		"--max-snapshot-age-ms", "604800000", "--enable-expire-snapshots=false")

	require.NoError(t, err)
	assert.JSONEq(t,
		`{"queue-config":{"max-snapshot-age-ms":604800000,"enable-expire-snapshots":false}}`,
		string(set.Last(t).Body))
	assert.Equal(t, "Task queue expire_snapshots configuration set for warehouse wh-1\n", out)
}

func TestTaskQueueSet_Heartbeat(t *testing.T) {
	t.Parallel()
	mux, baseURL := testutil.NewServer(t)
	set := testutil.Respond(t, mux, "POST "+whQueue+"tabular_purge/config", http.StatusNoContent, nil)

	_, err := runLkctl(t, baseURL, "task-queue", "tabular-purge", "set", "wh-1",
		"--max-seconds-since-last-heartbeat", "3600")

	require.NoError(t, err)
	assert.JSONEq(t, `{"queue-config":{},"max-seconds-since-last-heartbeat":3600}`, string(set.Last(t).Body))
}

func TestTaskQueueGet_TextShowsUnsetFields(t *testing.T) {
	t.Parallel()
	mux, baseURL := testutil.NewServer(t)
	testutil.Respond(t, mux, "GET "+whQueue+"optimize/config", http.StatusOK,
		`{"queue-config":{"enable-compaction":true,"enable-optimize":false,"target-file-size-bytes":536870912,
		  "target-manifest-size-bytes":8388608,"queue-name":"optimize"},
		  "max-seconds-since-last-heartbeat":null}`)

	out, err := runLkctl(t, baseURL, "task-queue", "optimize", "get", "wh-1")

	require.NoError(t, err)
	assert.Equal(t, `max-seconds-since-last-heartbeat: unset
enable-compaction: true
enable-optimize: false
enable-rewrite-manifests: unset
target-file-size-bytes: 536870912
target-manifest-size-bytes: 8388608
`, out)
}

func TestTaskQueueTaskLogCleanup_IsProjectScoped(t *testing.T) {
	t.Parallel()
	mux, baseURL := testutil.NewServer(t)
	const path = "/management/v1/project/task-queue/task_log_cleanup/config"
	set := testutil.Respond(t, mux, "POST "+path, http.StatusNoContent, nil)

	out, err := runLkctl(t, baseURL, "task-queue", "task-log-cleanup", "set", "-p", "prj-1",
		"--retention-period", "P30D")

	require.NoError(t, err)
	assert.Equal(t, "prj-1", set.Last(t).Header.Get("X-Project-Id"))
	assert.JSONEq(t, `{"queue-config":{"retention-period":"P30D"}}`, string(set.Last(t).Body))
	assert.Equal(t, "Task queue task_log_cleanup configuration set\n", out)
}

func TestTaskQueueOnLakekeeper_LakekeeperPlusQueueIsUnsupported(t *testing.T) {
	t.Parallel()
	mux, baseURL := testutil.NewServer(t)
	testutil.Respond(t, mux, "GET "+whQueue+"remove_orphan_files/config", http.StatusNotFound, nil)
	testutil.Respond(t, mux, "GET /management/v1/info", http.StatusOK, serverTestInfoBody)

	_, err := runLkctl(t, baseURL, "task-queue", "remove-orphan-files", "get", "wh-1")

	require.EqualError(t, err, "get remove_orphan_files queue config: requires Lakekeeper+; server is Lakekeeper 0.15.0")
}

func TestTaskQueueHelp_ScheduleOnlyOnLakekeeperPlusQueues(t *testing.T) {
	t.Parallel()
	baseURL := noServer(t)

	for _, plus := range []string{"expire-snapshots", "optimize", "remove-orphan-files"} {
		help, err := runLkctl(t, baseURL, "task-queue", plus, "--help")
		require.NoError(t, err)
		assert.Regexp(t, `schedule\s+.*\(Lakekeeper\+\)`, help)
	}
	help, err := runLkctl(t, baseURL, "task-queue", "tabular-purge", "--help")
	require.NoError(t, err)
	assert.NotContains(t, help, "schedule")
}

func TestTaskQueueSchedule_SchedulesTableTask(t *testing.T) {
	t.Parallel()
	mux, baseURL := testutil.NewServer(t)
	schedule := testutil.Respond(t, mux, "POST "+whQueue+"expire_snapshots/schedule", http.StatusOK,
		`{"task-id":"task-1"}`)

	out, err := runLkctl(t, baseURL, "task-queue", "expire-snapshots", "schedule", "wh-1", "tbl-1",
		"--scheduled-for", "2030-01-02T03:04:05Z")

	require.NoError(t, err)
	assert.JSONEq(t,
		`{"entity":{"type":"table","table-id":"tbl-1"},"scheduled-for":"2030-01-02T03:04:05Z"}`,
		string(schedule.Last(t).Body))
	assert.Equal(t, "Task task-1 scheduled on queue expire_snapshots for table tbl-1\n", out)
}

func TestTaskQueueSchedule_JSONPrintsResponse(t *testing.T) {
	t.Parallel()
	mux, baseURL := testutil.NewServer(t)
	testutil.Respond(t, mux, "POST "+whQueue+"remove_orphan_files/schedule", http.StatusOK, `{"task-id":"task-3"}`)

	out, err := runLkctl(t, baseURL, "task-queue", "remove-orphan-files", "schedule", "wh-1", "tbl-1", "-o", "json")

	require.NoError(t, err)
	assert.JSONEq(t, `{"task-id":"task-3"}`, out)
}

func TestTaskQueueSchedule_OptimizeSteps(t *testing.T) {
	t.Parallel()
	mux, baseURL := testutil.NewServer(t)
	schedule := testutil.Respond(t, mux, "POST "+whQueue+"optimize/schedule", http.StatusOK, `{"task-id":"task-2"}`)

	_, err := runLkctl(t, baseURL, "task-queue", "optimize", "schedule", "wh-1", "tbl-1",
		"--step", "compaction", "--step", "rewrite-manifests")

	require.NoError(t, err)
	assert.JSONEq(t,
		`{"entity":{"type":"table","table-id":"tbl-1"},"payload":{"steps":["compaction","rewrite-manifests"]}}`,
		string(schedule.Last(t).Body))
}

func TestTaskQueueSchedule_RejectsBadInputWithoutRequest(t *testing.T) {
	t.Parallel()
	baseURL := noServer(t)

	_, err := runLkctl(t, baseURL, "task-queue", "optimize", "schedule", "wh-1", "tbl-1", "--step", "vacuum")
	require.ErrorContains(t, err, `invalid --step "vacuum"`)

	_, err = runLkctl(t, baseURL, "task-queue", "expire-snapshots", "schedule", "wh-1", "tbl-1",
		"--scheduled-for", "tomorrow")
	require.ErrorContains(t, err, "invalid --scheduled-for")

	_, err = runLkctl(t, baseURL, "task-queue", "expire-snapshots", "schedule", "wh-1", "tbl-1", "--step", "compaction")
	require.ErrorContains(t, err, "unknown flag: --step")
}

func TestTaskQueueScheduleOnLakekeeper_IsUnsupported(t *testing.T) {
	t.Parallel()
	mux, baseURL := testutil.NewServer(t)
	testutil.Respond(t, mux, "POST "+whQueue+"optimize/schedule", http.StatusNotFound, nil)
	testutil.Respond(t, mux, "GET /management/v1/info", http.StatusOK, serverTestInfoBody)

	_, err := runLkctl(t, baseURL, "task-queue", "optimize", "schedule", "wh-1", "tbl-1")

	require.EqualError(t, err, "schedule optimize task: requires Lakekeeper+; server is Lakekeeper 0.15.0")
}

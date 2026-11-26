//go:build e2e_cli && e2e_plus

package clie2e

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Setting one field of a Lakekeeper+ queue sends only that field, so the
// fields the user did not set stay unset in the stored configuration and
// keep inheriting the server default.

func TestPlusTaskQueueExpireSnapshotsGetAndSet(t *testing.T) {
	requireBackend(t, BackendComposePlus)
	t.Parallel()

	id, _ := MustProvisionWarehouse(t)

	out := runOK(t, "task-queue", "expire-snapshots", "set", id, "--max-snapshot-age-ms", "604800000")
	assert.Contains(t, string(out), "configuration set")

	got := getQueueConfig(t, "task-queue", "expire-snapshots", "get", id)
	assert.EqualValues(t, 604800000, got.QueueConfig["max-snapshot-age-ms"])
	for _, unset := range []string{
		"enable-expire-snapshots", "max-ref-age-ms", "min-snapshots-to-expire", "min-snapshots-to-keep",
	} {
		assert.Nil(t, got.QueueConfig[unset], unset)
	}
	assert.Nil(t, got.MaxSecondsSinceLastHeartbeat)

	text := string(runOK(t, "task-queue", "expire-snapshots", "get", id))
	assert.Contains(t, text, "max-snapshot-age-ms: 604800000\n")
	assert.Contains(t, text, "enable-expire-snapshots: unset\n")
}

// serverRunsQueue reports whether the server info lists the task queue.
func serverRunsQueue(t *testing.T, queue string) bool {
	t.Helper()
	var info struct {
		Queues []string `json:"queues"`
	}
	decodeJSON(t, runOK(t, "server", "info", "--output", "json"), &info)
	return slices.Contains(info.Queues, queue)
}

// The Lakekeeper+ 0.15.1 image does not run the optimize queue, which makes
// it the queue for the queue-not-configured sentence.
func TestPlusTaskQueueOptimizeQueueNotConfigured(t *testing.T) {
	requireBackend(t, BackendComposePlus)
	t.Parallel()
	if serverRunsQueue(t, "optimize") {
		t.Skip("the server runs the optimize queue")
	}

	id, _ := MustProvisionWarehouse(t)

	_, stderr, _ := runFail(t, "task-queue", "optimize", "get", id)
	assert.Regexp(t,
		`^Error: get optimize queue config: requires the optimize task queue; server is Lakekeeper\+ \S+ without it\n$`,
		string(stderr))

	_, stderr, _ = runFail(t, "task-queue", "optimize", "set", id, "--enable-compaction=true")
	assert.Regexp(t,
		`^Error: set optimize queue config: requires the optimize task queue; server is Lakekeeper\+ \S+ without it\n$`,
		string(stderr))
}

func TestPlusTaskQueueOptimizeGetAndSet(t *testing.T) {
	requireBackend(t, BackendComposePlus)
	t.Parallel()
	if !serverRunsQueue(t, "optimize") {
		t.Skip("the server does not run the optimize queue")
	}

	id, _ := MustProvisionWarehouse(t)

	runOK(t, "task-queue", "optimize", "set", id, "--enable-compaction=true")

	got := getQueueConfig(t, "task-queue", "optimize", "get", id)
	assert.Equal(t, true, got.QueueConfig["enable-compaction"])
	assert.Nil(t, got.QueueConfig["enable-rewrite-manifests"], "enable-rewrite-manifests")
}

func TestPlusTaskQueueRemoveOrphanFilesGetAndSet(t *testing.T) {
	requireBackend(t, BackendComposePlus)
	t.Parallel()

	id, _ := MustProvisionWarehouse(t)

	runOK(t, "task-queue", "remove-orphan-files", "set", id,
		"--dry-run=true", "--max-seconds-since-last-heartbeat", "600")

	got := getQueueConfig(t, "task-queue", "remove-orphan-files", "get", id)
	assert.Equal(t, true, got.QueueConfig["dry-run"])
	if assert.NotNil(t, got.MaxSecondsSinceLastHeartbeat) {
		assert.Equal(t, int64(600), *got.MaxSecondsSinceLastHeartbeat)
	}
}

// A missing warehouse is a not-found, not an unsupported queue: the server
// runs the queue.
func TestPlusTaskQueueMissingWarehouseIsNotFound(t *testing.T) {
	requireBackend(t, BackendComposePlus)
	t.Parallel()

	_, stderr, _ := runFail(t, "task-queue", "expire-snapshots", "get", randomUUID())

	require.Contains(t, string(stderr), "Error: get expire_snapshots queue config: 404")
	assert.NotContains(t, string(stderr), "requires")
}

// scheduleJSON is the JSON output of `lkctl task-queue <queue> schedule`.
type scheduleJSON struct {
	TaskID string `json:"task-id"`
}

// Scheduling a task for a table puts it in the warehouse's task list. The
// server rejects scheduling on a queue the warehouse has not enabled
// (400 QueueDisabledForWarehouse), so each case enables it first.
func TestPlusTaskQueueScheduleShowsInTaskList(t *testing.T) {
	requireBackend(t, BackendComposePlus)
	t.Parallel()

	cases := []struct {
		cmd, queue, enable string
		extra              []string
	}{
		{"expire-snapshots", "expire_snapshots", "--enable-expire-snapshots=true", nil},
		{"optimize", "optimize", "--enable-optimize=true", []string{"--step", "compaction", "--step", "rewrite-manifests"}},
		{"remove-orphan-files", "remove_orphan_files", "--enable-remove-orphan-files=true", nil},
	}
	for _, tc := range cases {
		t.Run(tc.cmd, func(t *testing.T) {
			t.Parallel()
			if !serverRunsQueue(t, tc.queue) {
				t.Skip("the server does not run the " + tc.queue + " queue")
			}
			id, name := MustProvisionWarehouse(t)
			tableID := MustProvisionTable(t, name)
			runOK(t, "task-queue", tc.cmd, "set", id, tc.enable)

			args := append([]string{"task-queue", tc.cmd, "schedule", id, tableID, "--output", "json"}, tc.extra...)
			var got scheduleJSON
			decodeJSON(t, runOK(t, args...), &got)
			require.NotEmpty(t, got.TaskID)

			var listed []string
			for _, task := range warehouseTasks(t, id, tc.queue) {
				listed = append(listed, task.TaskId)
			}
			assert.Contains(t, listed, got.TaskID)
		})
	}
}

func TestPlusTaskQueueScheduleText(t *testing.T) {
	requireBackend(t, BackendComposePlus)
	t.Parallel()

	id, name := MustProvisionWarehouse(t)
	tableID := MustProvisionTable(t, name)
	runOK(t, "task-queue", "expire-snapshots", "set", id, "--enable-expire-snapshots=true")

	out := runOK(t, "task-queue", "expire-snapshots", "schedule", id, tableID)
	assert.Regexp(t, `^Task \S+ scheduled on queue expire_snapshots for table `+tableID+`\n$`, string(out))
}

func TestPlusTaskQueueScheduleOptimizeQueueNotConfigured(t *testing.T) {
	requireBackend(t, BackendComposePlus)
	t.Parallel()
	if serverRunsQueue(t, "optimize") {
		t.Skip("the server runs the optimize queue")
	}

	id, name := MustProvisionWarehouse(t)
	tableID := MustProvisionTable(t, name)

	_, stderr, _ := runFail(t, "task-queue", "optimize", "schedule", id, tableID)
	assert.Regexp(t,
		`^Error: schedule optimize task: requires the optimize task queue; server is Lakekeeper\+ \S+ without it\n$`,
		string(stderr))
}

// A missing table or warehouse is a not-found, not an unsupported queue.
func TestPlusTaskQueueScheduleMissingTargetIsNotFound(t *testing.T) {
	requireBackend(t, BackendComposePlus)
	t.Parallel()

	id, _ := MustProvisionWarehouse(t)
	for name, wh := range map[string]string{"table": id, "warehouse": randomUUID()} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			_, stderr, _ := runFail(t, "task-queue", "expire-snapshots", "schedule", wh, randomUUID())
			require.Contains(t, string(stderr), "Error: schedule expire_snapshots task: 404")
			assert.NotContains(t, string(stderr), "requires")
		})
	}
}

//go:build e2e_cli

// Task-queue configuration of the Lakekeeper queues, and the Lakekeeper+
// queue commands against Lakekeeper. The Lakekeeper+ queues against
// Lakekeeper+ are in plus_task_queue_test.go.

package clie2e

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// queueConfigJSON is the JSON output of `lkctl task-queue <queue> get`.
type queueConfigJSON struct {
	MaxSecondsSinceLastHeartbeat *int64         `json:"max-seconds-since-last-heartbeat"`
	QueueConfig                  map[string]any `json:"queue-config"`
}

func getQueueConfig(t *testing.T, args ...string) queueConfigJSON {
	t.Helper()
	var got queueConfigJSON
	decodeJSON(t, runOK(t, append(args, "--output", "json")...), &got)
	return got
}

func TestTaskQueueWarehouseQueuesGetAndSet(t *testing.T) {
	requireBackend(t, BackendCompose)
	t.Parallel()

	for _, queue := range []string{"tabular-purge", "soft-deletion"} {
		t.Run(queue, func(t *testing.T) {
			t.Parallel()
			id, _ := MustProvisionWarehouse(t)

			out := runOK(t, "task-queue", queue, "set", id, "--max-seconds-since-last-heartbeat", "3600")
			assert.Contains(t, string(out), "configuration set")

			got := getQueueConfig(t, "task-queue", queue, "get", id)
			if assert.NotNil(t, got.MaxSecondsSinceLastHeartbeat) {
				assert.Equal(t, int64(3600), *got.MaxSecondsSinceLastHeartbeat)
			}

			text := string(runOK(t, "task-queue", queue, "get", id))
			assert.Contains(t, text, "max-seconds-since-last-heartbeat: 3600\n")
		})
	}
}

// task_log_cleanup is configured per project; the test uses the server's
// default project and is the only test that touches its configuration.
// Lakekeeper 0.15 requires cleanup-period in a set, although the spec marks
// it nullable.
func TestTaskQueueTaskLogCleanupGetAndSet(t *testing.T) {
	requireBackend(t, BackendCompose)
	t.Parallel()

	runOK(t, "task-queue", "task-log-cleanup", "set", "--cleanup-period", "P2D", "--retention-period", "P30D")

	got := getQueueConfig(t, "task-queue", "task-log-cleanup", "get")
	assert.Equal(t, "P30D", got.QueueConfig["retention-period"])
	assert.Equal(t, "P2D", got.QueueConfig["cleanup-period"])
	assert.Nil(t, got.MaxSecondsSinceLastHeartbeat)

	text := string(runOK(t, "task-queue", "task-log-cleanup", "get"))
	assert.Contains(t, text, "retention-period: P30D\n")
	assert.Contains(t, text, "max-seconds-since-last-heartbeat: unset\n")
}

func TestTaskQueueLakekeeperPlusQueuesRequireLakekeeperPlus(t *testing.T) {
	requireBackend(t, BackendCompose)
	t.Parallel()

	id, _ := MustProvisionWarehouse(t)

	for _, queue := range []string{"expire_snapshots", "optimize", "remove_orphan_files"} {
		cmdName := map[string]string{
			"expire_snapshots":    "expire-snapshots",
			"optimize":            "optimize",
			"remove_orphan_files": "remove-orphan-files",
		}[queue]
		t.Run(cmdName, func(t *testing.T) {
			t.Parallel()

			_, stderr, _ := runFail(t, "task-queue", cmdName, "get", id)
			assert.Regexp(t,
				`^Error: get `+queue+` queue config: requires Lakekeeper\+; server is Lakekeeper \S+\n$`,
				string(stderr))

			_, stderr, _ = runFail(t, "task-queue", cmdName, "set", id, "--max-seconds-since-last-heartbeat", "60")
			require.Regexp(t,
				`^Error: set `+queue+` queue config: requires Lakekeeper\+; server is Lakekeeper \S+\n$`,
				string(stderr))
		})
	}
}

func TestTaskQueueScheduleRequiresLakekeeperPlus(t *testing.T) {
	requireBackend(t, BackendCompose)
	t.Parallel()

	id, name := MustProvisionWarehouse(t)
	tableID := MustProvisionTable(t, name)

	for cmdName, queue := range map[string]string{
		"expire-snapshots":    "expire_snapshots",
		"optimize":            "optimize",
		"remove-orphan-files": "remove_orphan_files",
	} {
		t.Run(cmdName, func(t *testing.T) {
			t.Parallel()
			_, stderr, _ := runFail(t, "task-queue", cmdName, "schedule", id, tableID)
			assert.Regexp(t,
				`^Error: schedule `+queue+` task: requires Lakekeeper\+; server is Lakekeeper \S+\n$`,
				string(stderr))
		})
	}
}

package client

import (
	"context"
	"errors"

	managementv1 "github.com/lakekeeper/go-lakekeeper/pkg/apis/management/v1"
)

// Lakekeeper+ task queues, as the server info names them in its queues.
const (
	QueueExpireSnapshots   = "expire_snapshots"
	QueueOptimize          = "optimize"
	QueueRemoveOrphanFiles = "remove_orphan_files"
)

// plusQueueRequirement is what the operations of a Lakekeeper+ task queue
// need: Lakekeeper+ running that queue.
func plusQueueRequirement(queue string) requirement {
	return requirement{feature: "task queue " + queue, lakekeeperPlus: true, queue: queue}
}

// TaskQueues is a one-call façade over the task-queue operations of
// TasksAPIService: it reads and sets the configuration of every task queue,
// and schedules tasks on the Lakekeeper+ queues.
//
// The Lakekeeper queues are tabular_purge and soft_deletion (per warehouse)
// and task_log_cleanup (per project). The Lakekeeper+ queues are
// expire_snapshots, optimize and remove_orphan_files (per warehouse); on a
// server that is not Lakekeeper+ or does not run the queue, their methods
// fail with an *UnsupportedError (see ErrUnsupported).
//
// The Set methods send the request as given: a field left unset in the
// queue configuration stays unset on the server, so the warehouse keeps
// inheriting the server default. Build the configuration from its zero
// value rather than from the generated New… constructors where those set
// defaults explicitly. Setting a configuration replaces the stored one.
type TaskQueues struct {
	api  *managementv1.TasksAPIService
	caps *capabilityCache
}

// TabularPurgeConfig returns the tabular_purge queue configuration of a
// warehouse.
func (q *TaskQueues) TabularPurgeConfig(ctx context.Context, warehouseID string) (*managementv1.GetPurgeQueueConfig, error) {
	out, resp, err := q.api.GetTaskQueueConfigTabularPurge(ctx, warehouseID).Execute()
	return entityResult("get tabular_purge queue config", out, resp, err)
}

// SetTabularPurgeConfig sets the tabular_purge queue configuration of a
// warehouse.
func (q *TaskQueues) SetTabularPurgeConfig(ctx context.Context, warehouseID string, req *managementv1.SetPurgeQueueConfig) error {
	const op = "set tabular_purge queue config"
	if req == nil {
		return nilRequest(op)
	}
	resp, err := q.api.SetTaskQueueConfigTabularPurge(ctx, warehouseID).SetPurgeQueueConfig(*req).Execute()
	return WrapAPIError(op, err, resp)
}

// SoftDeletionConfig returns the soft_deletion queue configuration of a
// warehouse.
func (q *TaskQueues) SoftDeletionConfig(ctx context.Context, warehouseID string) (*managementv1.GetSoftDeletionQueueConfig, error) {
	out, resp, err := q.api.GetTaskQueueConfigSoftDeletion(ctx, warehouseID).Execute()
	return entityResult("get soft_deletion queue config", out, resp, err)
}

// SetSoftDeletionConfig sets the soft_deletion queue configuration of a
// warehouse.
func (q *TaskQueues) SetSoftDeletionConfig(ctx context.Context, warehouseID string, req *managementv1.SetSoftDeletionQueueConfig) error {
	const op = "set soft_deletion queue config"
	if req == nil {
		return nilRequest(op)
	}
	resp, err := q.api.SetTaskQueueConfigSoftDeletion(ctx, warehouseID).SetSoftDeletionQueueConfig(*req).Execute()
	return WrapAPIError(op, err, resp)
}

// TaskLogCleanupConfig returns the task_log_cleanup queue configuration of
// a project. Name the project with WithProject; without it the server
// applies its default project.
func (q *TaskQueues) TaskLogCleanupConfig(ctx context.Context, callOpts ...CallOption) (*managementv1.GetTaskLogCleanupConfig, error) {
	const op = "get task_log_cleanup queue config"
	req, err := withProject(op, q.api.GetProjectTaskQueueConfigTaskLogCleanup(ctx), callOpts)
	if err != nil {
		return nil, err
	}
	out, resp, err := req.Execute()
	return entityResult(op, out, resp, err)
}

// SetTaskLogCleanupConfig sets the task_log_cleanup queue configuration of
// a project. Name the project with WithProject; without it the server
// applies its default project.
func (q *TaskQueues) SetTaskLogCleanupConfig(ctx context.Context, req *managementv1.SetTaskLogCleanupConfig, callOpts ...CallOption) error {
	const op = "set task_log_cleanup queue config"
	if req == nil {
		return nilRequest(op)
	}
	call, err := withProject(op, q.api.SetProjectTaskQueueConfigTaskLogCleanup(ctx), callOpts)
	if err != nil {
		return err
	}
	resp, err := call.SetTaskLogCleanupConfig(*req).Execute()
	return WrapAPIError(op, err, resp)
}

// ExpireSnapshotsConfig returns the expire_snapshots queue configuration of
// a warehouse (Lakekeeper+).
func (q *TaskQueues) ExpireSnapshotsConfig(ctx context.Context, warehouseID string) (*managementv1.GetExpireSnapshotsQueueConfig, error) {
	const op = "get expire_snapshots queue config"
	out, resp, err := q.api.GetTaskQueueConfigExpireSnapshots(ctx, warehouseID).Execute()
	out, err = entityResult(op, out, resp, err)
	return out, q.caps.diagnose(ctx, plusQueueRequirement(QueueExpireSnapshots), op, err)
}

// SetExpireSnapshotsConfig sets the expire_snapshots queue configuration of
// a warehouse (Lakekeeper+).
func (q *TaskQueues) SetExpireSnapshotsConfig(ctx context.Context, warehouseID string, req *managementv1.SetExpireSnapshotsQueueConfig) error {
	const op = "set expire_snapshots queue config"
	if req == nil {
		return nilRequest(op)
	}
	resp, err := q.api.SetTaskQueueConfigExpireSnapshots(ctx, warehouseID).SetExpireSnapshotsQueueConfig(*req).Execute()
	return q.caps.diagnose(ctx, plusQueueRequirement(QueueExpireSnapshots), op, WrapAPIError(op, err, resp))
}

// OptimizeConfig returns the optimize queue configuration of a warehouse
// (Lakekeeper+).
func (q *TaskQueues) OptimizeConfig(ctx context.Context, warehouseID string) (*managementv1.GetOptimizeQueueConfig, error) {
	const op = "get optimize queue config"
	out, resp, err := q.api.GetTaskQueueConfigOptimize(ctx, warehouseID).Execute()
	out, err = entityResult(op, out, resp, err)
	return out, q.caps.diagnose(ctx, plusQueueRequirement(QueueOptimize), op, err)
}

// SetOptimizeConfig sets the optimize queue configuration of a warehouse
// (Lakekeeper+).
func (q *TaskQueues) SetOptimizeConfig(ctx context.Context, warehouseID string, req *managementv1.SetOptimizeQueueConfig) error {
	const op = "set optimize queue config"
	if req == nil {
		return nilRequest(op)
	}
	resp, err := q.api.SetTaskQueueConfigOptimize(ctx, warehouseID).SetOptimizeQueueConfig(*req).Execute()
	return q.caps.diagnose(ctx, plusQueueRequirement(QueueOptimize), op, WrapAPIError(op, err, resp))
}

// RemoveOrphanFilesConfig returns the remove_orphan_files queue
// configuration of a warehouse (Lakekeeper+).
func (q *TaskQueues) RemoveOrphanFilesConfig(ctx context.Context, warehouseID string) (*managementv1.GetRemoveOrphanFilesQueueConfig, error) {
	const op = "get remove_orphan_files queue config"
	out, resp, err := q.api.GetTaskQueueConfigRemoveOrphanFiles(ctx, warehouseID).Execute()
	out, err = entityResult(op, out, resp, err)
	return out, q.caps.diagnose(ctx, plusQueueRequirement(QueueRemoveOrphanFiles), op, err)
}

// SetRemoveOrphanFilesConfig sets the remove_orphan_files queue
// configuration of a warehouse (Lakekeeper+).
func (q *TaskQueues) SetRemoveOrphanFilesConfig(ctx context.Context, warehouseID string, req *managementv1.SetRemoveOrphanFilesQueueConfig) error {
	const op = "set remove_orphan_files queue config"
	if req == nil {
		return nilRequest(op)
	}
	resp, err := q.api.SetTaskQueueConfigRemoveOrphanFiles(ctx, warehouseID).SetRemoveOrphanFilesQueueConfig(*req).Execute()
	return q.caps.diagnose(ctx, plusQueueRequirement(QueueRemoveOrphanFiles), op, WrapAPIError(op, err, resp))
}

// ScheduleExpireSnapshots schedules an expire_snapshots task for an entity
// of a warehouse (Lakekeeper+) and returns the task's id. A missing
// warehouse or entity is a not-found *APIError.
func (q *TaskQueues) ScheduleExpireSnapshots(ctx context.Context, warehouseID string, req *managementv1.ScheduleExpireSnapshotsTaskRequest) (*managementv1.ScheduleTaskResponse, error) {
	const op = "schedule expire_snapshots task"
	if req == nil {
		return nil, nilRequest(op)
	}
	out, resp, err := q.api.ScheduleTaskExpireSnapshots(ctx, warehouseID).ScheduleExpireSnapshotsTaskRequest(*req).Execute()
	out, err = entityResult(op, out, resp, err)
	return out, q.caps.diagnose(ctx, plusQueueRequirement(QueueExpireSnapshots), op, err)
}

// ScheduleOptimize schedules an optimize task for an entity of a warehouse
// (Lakekeeper+) and returns the task's id. The payload's steps take the
// managementv1.OptimizeStep constants; without steps the server runs its
// default steps. A missing warehouse or entity is a not-found *APIError.
func (q *TaskQueues) ScheduleOptimize(ctx context.Context, warehouseID string, req *managementv1.ScheduleOptimizeTaskRequest) (*managementv1.ScheduleTaskResponse, error) {
	const op = "schedule optimize task"
	if req == nil {
		return nil, nilRequest(op)
	}
	out, resp, err := q.api.ScheduleTaskOptimize(ctx, warehouseID).ScheduleOptimizeTaskRequest(*req).Execute()
	out, err = entityResult(op, out, resp, err)
	return out, q.caps.diagnose(ctx, plusQueueRequirement(QueueOptimize), op, err)
}

// ScheduleRemoveOrphanFiles schedules a remove_orphan_files task for an
// entity of a warehouse (Lakekeeper+) and returns the task's id. A missing
// warehouse or entity is a not-found *APIError.
func (q *TaskQueues) ScheduleRemoveOrphanFiles(ctx context.Context, warehouseID string, req *managementv1.ScheduleRemoveOrphanFilesTaskRequest) (*managementv1.ScheduleTaskResponse, error) {
	const op = "schedule remove_orphan_files task"
	if req == nil {
		return nil, nilRequest(op)
	}
	out, resp, err := q.api.ScheduleTaskRemoveOrphanFiles(ctx, warehouseID).ScheduleRemoveOrphanFilesTaskRequest(*req).Execute()
	out, err = entityResult(op, out, resp, err)
	return out, q.caps.diagnose(ctx, plusQueueRequirement(QueueRemoveOrphanFiles), op, err)
}

func nilRequest(op string) error {
	return errors.New(op + ": request must not be nil")
}

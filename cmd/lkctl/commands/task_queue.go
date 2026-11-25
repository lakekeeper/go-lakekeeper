package commands

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"slices"
	"strings"
	"time"

	"github.com/spf13/cobra"

	managementv1 "github.com/lakekeeper/go-lakekeeper/pkg/apis/management/v1"
	"github.com/lakekeeper/go-lakekeeper/pkg/client"
)

// heartbeatField is the field every queue's configuration has next to its
// queue-config.
const heartbeatField = "max-seconds-since-last-heartbeat"

// queueFieldKind is the type of a queue-config field: how its `set` flag is
// defined, and how the flag's value is read back.
type queueFieldKind struct {
	define func(cmd *cobra.Command, name, usage string)
	read   func(cmd *cobra.Command, name string) (any, error)
}

var (
	boolField = queueFieldKind{
		define: func(cmd *cobra.Command, name, usage string) { cmd.Flags().Bool(name, false, usage) },
		read:   func(cmd *cobra.Command, name string) (any, error) { return cmd.Flags().GetBool(name) },
	}
	int64Field = queueFieldKind{
		define: func(cmd *cobra.Command, name, usage string) { cmd.Flags().Int64(name, 0, usage) },
		read:   func(cmd *cobra.Command, name string) (any, error) { return cmd.Flags().GetInt64(name) },
	}
	stringField = queueFieldKind{
		define: func(cmd *cobra.Command, name, usage string) { cmd.Flags().String(name, "", usage) },
		read:   func(cmd *cobra.Command, name string) (any, error) { return cmd.Flags().GetString(name) },
	}
)

// queueField is one field of a queue-config. Its name is the JSON name, and
// also the name of its `set` flag.
type queueField struct {
	name  string
	kind  queueFieldKind
	usage string
}

// queueTarget is what a queue command acts on: the client, and the
// warehouse of a warehouse queue or the call options (-p) of a project
// queue.
type queueTarget struct {
	c           *client.Client
	warehouseID string
	callOpts    []client.CallOption
}

// taskQueue describes one task queue for the `task-queue` commands.
type taskQueue struct {
	use   string       // command name
	queue client.Queue // the server's name for the queue
	short string
	plus  bool // a Lakekeeper+ queue
	// project: the queue is configured per project (-p), not per warehouse.
	project bool
	fields  []queueField
	get     func(ctx context.Context, t queueTarget) (any, error)
	// set decodes body, the JSON of the queue's Set… request, and sends it.
	set func(ctx context.Context, t queueTarget, body []byte) error
	// schedule schedules a task on the queue; nil for a queue without
	// scheduling.
	schedule func(ctx context.Context, t queueTarget, s queueSchedule) (*managementv1.ScheduleTaskResponse, error)
	// steps: schedule takes optimize steps (--step).
	steps bool
}

// queueSchedule is what `schedule` sends: the entity, and optionally when
// the task runs and which optimize steps it runs.
type queueSchedule struct {
	entity       managementv1.WarehouseTaskEntityId
	scheduledFor *time.Time
	steps        []managementv1.OptimizeStep
}

var taskQueues = []taskQueue{
	{
		use: "tabular-purge", queue: client.QueueTabularPurge,
		short: "Configure the tabular_purge queue of a warehouse",
		get:   warehouseGet((*client.TaskQueues).TabularPurgeConfig),
		set:   warehouseSet((*client.TaskQueues).SetTabularPurgeConfig),
	},
	{
		use: "soft-deletion", queue: client.QueueSoftDeletion,
		short: "Configure the soft_deletion queue of a warehouse",
		get:   warehouseGet((*client.TaskQueues).SoftDeletionConfig),
		set:   warehouseSet((*client.TaskQueues).SetSoftDeletionConfig),
	},
	{
		use: "task-log-cleanup", queue: client.QueueTaskLogCleanup, project: true,
		short: "Configure the task_log_cleanup queue of a project",
		fields: []queueField{
			{"cleanup-period", stringField, "How often the cleanup runs, as an ISO 8601 duration (e.g. P1D)"},
			{"retention-period", stringField, "How long task logs are kept, as an ISO 8601 duration (e.g. P90D)"},
		},
		get: projectGet((*client.TaskQueues).TaskLogCleanupConfig),
		set: projectSet((*client.TaskQueues).SetTaskLogCleanupConfig),
	},
	{
		use: "expire-snapshots", queue: client.QueueExpireSnapshots, plus: true,
		short: "Configure the expire_snapshots queue of a warehouse",
		fields: []queueField{
			{"enable-expire-snapshots", boolField, "Expire snapshots of all tables in the warehouse"},
			{"max-snapshot-age-ms", int64Field, "Maximum age of a snapshot before it expires, in milliseconds"},
			{"max-ref-age-ms", int64Field, "Maximum age of a snapshot reference other than main, in milliseconds"},
			{"min-snapshots-to-expire", int64Field, "Minimum number of expirable snapshots before a run starts"},
			{"min-snapshots-to-keep", int64Field, "Minimum number of snapshots to keep per table and branch"},
			{"disable-min-age-check", boolField, "Allow ages below 24 hours"},
		},
		get: warehouseGet((*client.TaskQueues).ExpireSnapshotsConfig),
		set: warehouseSet((*client.TaskQueues).SetExpireSnapshotsConfig),
		schedule: warehouseSchedule(managementv1.NewScheduleExpireSnapshotsTaskRequest,
			(*client.TaskQueues).ScheduleExpireSnapshots),
	},
	{
		use: "optimize", queue: client.QueueOptimize, plus: true,
		short: "Configure the optimize queue of a warehouse",
		fields: []queueField{
			{"enable-optimize", boolField, "Optimize all tables in the warehouse"},
			{"enable-compaction", boolField, "Compact data files; unset follows enable-optimize"},
			{"enable-rewrite-manifests", boolField, "Rewrite manifests; unset follows enable-optimize"},
			{"target-file-size-bytes", int64Field, "Size of the data and delete files written"},
			{"target-manifest-size-bytes", int64Field, "Size of the manifests written"},
		},
		get:   warehouseGet((*client.TaskQueues).OptimizeConfig),
		set:   warehouseSet((*client.TaskQueues).SetOptimizeConfig),
		steps: true,
		schedule: warehouseSchedule(managementv1.NewScheduleOptimizeTaskRequest,
			(*client.TaskQueues).ScheduleOptimize),
	},
	{
		use: "remove-orphan-files", queue: client.QueueRemoveOrphanFiles, plus: true,
		short: "Configure the remove_orphan_files queue of a warehouse",
		fields: []queueField{
			{"enable-remove-orphan-files", boolField, "Schedule orphan-file removal for all tables in the warehouse"},
			{"default-older-than-ms", int64Field, "Remove only files older than this, in milliseconds"},
			{"disable-min-older-than-check", boolField, "Allow default-older-than-ms below 24 hours"},
			{"dry-run", boolField, "Find orphan files without deleting them"},
			{"max-run-time-seconds", int64Field, "Maximum duration of one task attempt, in seconds"},
			{"maximum-interval-seconds", int64Field, "Upper bound on the interval between runs, in seconds"},
			{"target-reclaim-bytes", int64Field, "Orphan bytes a run should find; sets the interval between runs"},
		},
		get: warehouseGet((*client.TaskQueues).RemoveOrphanFilesConfig),
		set: warehouseSet((*client.TaskQueues).SetRemoveOrphanFilesConfig),
		schedule: warehouseSchedule(managementv1.NewScheduleRemoveOrphanFilesTaskRequest,
			(*client.TaskQueues).ScheduleRemoveOrphanFiles),
	},
}

// warehouseGet adapts the read of a warehouse queue's configuration.
func warehouseGet[T any](get func(*client.TaskQueues, context.Context, string) (*T, error)) func(context.Context, queueTarget) (any, error) {
	return func(ctx context.Context, t queueTarget) (any, error) {
		return get(t.c.TaskQueues, ctx, t.warehouseID)
	}
}

// projectGet adapts the read of a project queue's configuration.
func projectGet[T any](get func(*client.TaskQueues, context.Context, ...client.CallOption) (*T, error)) func(context.Context, queueTarget) (any, error) {
	return func(ctx context.Context, t queueTarget) (any, error) {
		return get(t.c.TaskQueues, ctx, t.callOpts...)
	}
}

// warehouseSet adapts the set of a warehouse queue's configuration: it
// decodes the request from the body built from the flags.
func warehouseSet[T any](set func(*client.TaskQueues, context.Context, string, *T) error) func(context.Context, queueTarget, []byte) error {
	return func(ctx context.Context, t queueTarget, body []byte) error {
		var req T
		if err := json.Unmarshal(body, &req); err != nil {
			return err
		}
		return set(t.c.TaskQueues, ctx, t.warehouseID, &req)
	}
}

// projectSet adapts the set of a project queue's configuration, as
// warehouseSet does.
func projectSet[T any](set func(*client.TaskQueues, context.Context, *T, ...client.CallOption) error) func(context.Context, queueTarget, []byte) error {
	return func(ctx context.Context, t queueTarget, body []byte) error {
		var req T
		if err := json.Unmarshal(body, &req); err != nil {
			return err
		}
		return set(t.c.TaskQueues, ctx, &req, t.callOpts...)
	}
}

// scheduleRequest is what every Schedule…TaskRequest has.
type scheduleRequest interface {
	SetScheduledFor(v time.Time)
}

// warehouseSchedule adapts scheduling a task on a Lakekeeper+ queue: it
// builds the request with newRequest, sets --scheduled-for, and the steps
// when the request takes them (optimize).
func warehouseSchedule[R scheduleRequest](
	newRequest func(managementv1.WarehouseTaskEntityId) R,
	schedule func(*client.TaskQueues, context.Context, string, R) (*managementv1.ScheduleTaskResponse, error),
) func(context.Context, queueTarget, queueSchedule) (*managementv1.ScheduleTaskResponse, error) {
	return func(ctx context.Context, t queueTarget, s queueSchedule) (*managementv1.ScheduleTaskResponse, error) {
		req := newRequest(s.entity)
		if s.scheduledFor != nil {
			req.SetScheduledFor(*s.scheduledFor)
		}
		if withSteps, ok := any(req).(interface {
			SetPayload(v managementv1.ScheduleOptimizeTaskRequestPayload)
		}); ok && len(s.steps) > 0 {
			withSteps.SetPayload(managementv1.ScheduleOptimizeTaskRequestPayload{Steps: s.steps})
		}
		return schedule(t.c.TaskQueues, ctx, t.warehouseID, req)
	}
}

func newTaskQueueCmd(opts *clientOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "task-queue",
		Aliases: []string{"tq"},
		Short:   "Manage task queues",
		Long: "Manage task queues.\n\n" +
			"Each queue has get and set; the Lakekeeper+ queues also have schedule. " +
			"Queues marked (Lakekeeper+) need a Lakekeeper+ server running the queue.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}

	for _, q := range taskQueues {
		cmd.AddCommand(newQueueCmd(opts, q))
	}
	return cmd
}

func newQueueCmd(opts *clientOptions, q taskQueue) *cobra.Command {
	short := q.short
	if q.plus {
		short = lakekeeperPlus(short)
	}
	cmd := &cobra.Command{
		Use:     q.use,
		Aliases: []string{string(q.queue)},
		Short:   short,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}
	cmd.AddCommand(newQueueGetCmd(opts, q))
	cmd.AddCommand(newQueueSetCmd(opts, q))
	if q.schedule != nil {
		cmd.AddCommand(newQueueScheduleCmd(opts, q))
	}
	return cmd
}

// queueArgs returns the positional arguments of a queue's get and set:
// a warehouse id, or none for a project queue.
func queueArgs(q taskQueue) (string, cobra.PositionalArgs) {
	if q.project {
		return "", cobra.NoArgs
	}
	return " <warehouse-id>", cobra.ExactArgs(1)
}

func newQueueGetCmd(opts *clientOptions, q taskQueue) *cobra.Command {
	var (
		output  string
		project string
	)
	use, args := queueArgs(q)
	short := "Print the " + string(q.queue) + " queue configuration"
	if q.plus {
		short = lakekeeperPlus(short)
	}

	cmd := &cobra.Command{
		Use:   "get" + use,
		Short: short,
		Long: short + "\n\n" +
			"Text output prints every field; a field shown as unset inherits the server default.",
		Args: args,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			c, err := newClient(ctx, opts)
			if err != nil {
				return err
			}
			cfg, err := q.get(ctx, queueTarget{c, firstArg(args), projectFlagOptions(cmd, project)})
			if err != nil {
				return err
			}

			switch output {
			case "json":
				return printJSON(cmd.OutOrStdout(), cfg)
			case "text":
				return printQueueConfig(cmd.OutOrStdout(), q, cfg)
			default:
				return fmt.Errorf("unknown output format: %s", output)
			}
		},
	}

	cmd.Flags().StringVarP(&output, "output", "o", "text", "Output format. One of: json|text")
	if q.project {
		addProjectFlag(cmd, &project)
	}
	return cmd
}

func newQueueSetCmd(opts *clientOptions, q taskQueue) *cobra.Command {
	var project string
	use, args := queueArgs(q)
	short := "Set the " + string(q.queue) + " queue configuration"
	if q.plus {
		short = lakekeeperPlus(short)
	}

	cmd := &cobra.Command{
		Use:   "set" + use,
		Short: short,
		Long: short + "\n\n" +
			"Only the flags given are sent; every other field is left unset and inherits the server default. " +
			"The configuration replaces the stored one.",
		Args: args,
		RunE: func(cmd *cobra.Command, args []string) error {
			body, err := queueSetBody(cmd, q)
			if err != nil {
				return err
			}
			ctx := cmd.Context()
			c, err := newClient(ctx, opts)
			if err != nil {
				return err
			}
			if err := q.set(ctx, queueTarget{c, firstArg(args), projectFlagOptions(cmd, project)}, body); err != nil {
				return err
			}
			if q.project {
				fmt.Fprintf(cmd.OutOrStdout(), "Task queue %s configuration set\n", q.queue)
			} else {
				fmt.Fprintf(cmd.OutOrStdout(), "Task queue %s configuration set for warehouse %s\n", q.queue, args[0])
			}
			return nil
		},
	}

	cmd.Flags().Int64(heartbeatField, 0,
		"Seconds without a heartbeat after which a running task counts as lost")
	for _, f := range q.fields {
		f.kind.define(cmd, f.name, f.usage)
	}
	if q.project {
		addProjectFlag(cmd, &project)
	}
	return cmd
}

func newQueueScheduleCmd(opts *clientOptions, q taskQueue) *cobra.Command {
	var (
		output       string
		scheduledFor string
		steps        []string
	)
	short := lakekeeperPlus("Schedule a " + string(q.queue) + " task for a table")
	long := short + "\n\n" +
		"The task runs on the next worker poll, or at --scheduled-for (RFC 3339, within about a year). " +
		"The warehouse and table configuration still apply; the server rejects the task when they disable it."
	if q.steps {
		long += "\n\nWithout --step the server runs its default steps. Steps: " + optimizeStepNames() + "."
	}

	cmd := &cobra.Command{
		Use:   "schedule <warehouse-id> <table-id>",
		Short: short,
		Long:  long,
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			s := queueSchedule{
				entity: managementv1.WarehouseTaskEntityIdTableAsWarehouseTaskEntityId(
					managementv1.NewWarehouseTaskEntityIdTable(args[1], "table")),
			}
			if scheduledFor != "" {
				at, err := time.Parse(time.RFC3339, scheduledFor)
				if err != nil {
					return fmt.Errorf("invalid --scheduled-for %q: want RFC 3339, e.g. 2030-01-02T03:04:05Z", scheduledFor)
				}
				s.scheduledFor = &at
			}
			for _, name := range steps {
				step, err := managementv1.NewOptimizeStepFromValue(name)
				if err != nil {
					return fmt.Errorf("invalid --step %q: want one of %s", name, optimizeStepNames())
				}
				s.steps = append(s.steps, *step)
			}
			if output != "json" && output != "text" {
				return fmt.Errorf("unknown output format: %s", output)
			}

			ctx := cmd.Context()
			c, err := newClient(ctx, opts)
			if err != nil {
				return err
			}
			resp, err := q.schedule(ctx, queueTarget{c: c, warehouseID: args[0]}, s)
			if err != nil {
				return err
			}
			if output == "json" {
				return printJSON(cmd.OutOrStdout(), resp)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Task %s scheduled on queue %s for table %s\n", resp.TaskId, q.queue, args[1])
			return nil
		},
	}

	cmd.Flags().StringVarP(&output, "output", "o", "text", "Output format. One of: json|text")
	cmd.Flags().StringVar(&scheduledFor, "scheduled-for", "", "When the task runs, in RFC 3339; default: the next worker poll")
	if q.steps {
		cmd.Flags().StringArrayVar(&steps, "step", nil, "Optimize step to run; repeat for several. One of: "+optimizeStepNames())
	}
	return cmd
}

func optimizeStepNames() string {
	names := make([]string, 0, len(managementv1.AllowedOptimizeStepEnumValues))
	for _, s := range managementv1.AllowedOptimizeStepEnumValues {
		names = append(names, string(s))
	}
	return strings.Join(names, ", ")
}

// queueSetBody builds the JSON of a queue's Set… request from the flags the
// user gave, and only those.
func queueSetBody(cmd *cobra.Command, q taskQueue) ([]byte, error) {
	flags := cmd.Flags()
	queueConfig := map[string]any{}
	for _, f := range q.fields {
		if !flags.Changed(f.name) {
			continue
		}
		v, err := f.kind.read(cmd, f.name)
		if err != nil {
			return nil, err
		}
		queueConfig[f.name] = v
	}
	body := map[string]any{"queue-config": queueConfig}
	if flags.Changed(heartbeatField) {
		v, err := flags.GetInt64(heartbeatField)
		if err != nil {
			return nil, err
		}
		body[heartbeatField] = v
	}
	return json.Marshal(body)
}

// printQueueConfig prints the heartbeat setting, then every field of the
// queue in name order, as "<name>: <value>", with "unset" for a field the
// configuration leaves unset.
func printQueueConfig(w io.Writer, q taskQueue, cfg any) error {
	raw, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	var decoded struct {
		Heartbeat   any            `json:"max-seconds-since-last-heartbeat"`
		QueueConfig map[string]any `json:"queue-config"`
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	if err := dec.Decode(&decoded); err != nil {
		return err
	}

	fmt.Fprintf(w, "%s: %s\n", heartbeatField, queueValue(decoded.Heartbeat))
	names := make([]string, 0, len(q.fields))
	for _, f := range q.fields {
		names = append(names, f.name)
	}
	slices.Sort(names)
	for _, name := range names {
		fmt.Fprintf(w, "%s: %s\n", name, queueValue(decoded.QueueConfig[name]))
	}
	return nil
}

func queueValue(v any) string {
	if v == nil {
		return "unset"
	}
	return fmt.Sprint(v)
}

func firstArg(args []string) string {
	if len(args) == 0 {
		return ""
	}
	return args[0]
}

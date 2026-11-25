package commands

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"slices"

	"github.com/spf13/cobra"

	managementv1 "github.com/lakekeeper/go-lakekeeper/pkg/apis/management/v1"
	"github.com/lakekeeper/go-lakekeeper/pkg/client"
)

// heartbeatField is the field every queue's configuration has next to its
// queue-config.
const heartbeatField = "max-seconds-since-last-heartbeat"

type queueFieldKind int

const (
	boolField queueFieldKind = iota
	int64Field
	stringField
)

// queueField is one field of a queue-config. Its name is the JSON name, and
// also the name of its `set` flag.
type queueField struct {
	name  string
	kind  queueFieldKind
	usage string
}

// taskQueue describes one task queue for the `task-queue` commands.
type taskQueue struct {
	use   string // command name
	queue string // the server's name for the queue
	short string
	plus  bool // a Lakekeeper+ queue
	// project: the queue is configured per project (-p), not per warehouse.
	project bool
	fields  []queueField
	get     func(ctx context.Context, c *client.Client, warehouseID string, callOpts []client.CallOption) (any, error)
	// set decodes body, the JSON of the queue's Set… request, and sends it.
	set func(ctx context.Context, c *client.Client, warehouseID string, body []byte, callOpts []client.CallOption) error
}

var taskQueues = []taskQueue{
	{
		use: "tabular-purge", queue: "tabular_purge",
		short: "Configure the tabular_purge queue of a warehouse",
		get: func(ctx context.Context, c *client.Client, wh string, _ []client.CallOption) (any, error) {
			return c.TaskQueues.TabularPurgeConfig(ctx, wh)
		},
		set: func(ctx context.Context, c *client.Client, wh string, body []byte, _ []client.CallOption) error {
			var req managementv1.SetPurgeQueueConfig
			if err := json.Unmarshal(body, &req); err != nil {
				return err
			}
			return c.TaskQueues.SetTabularPurgeConfig(ctx, wh, &req)
		},
	},
	{
		use: "soft-deletion", queue: "soft_deletion",
		short: "Configure the soft_deletion queue of a warehouse",
		get: func(ctx context.Context, c *client.Client, wh string, _ []client.CallOption) (any, error) {
			return c.TaskQueues.SoftDeletionConfig(ctx, wh)
		},
		set: func(ctx context.Context, c *client.Client, wh string, body []byte, _ []client.CallOption) error {
			var req managementv1.SetSoftDeletionQueueConfig
			if err := json.Unmarshal(body, &req); err != nil {
				return err
			}
			return c.TaskQueues.SetSoftDeletionConfig(ctx, wh, &req)
		},
	},
	{
		use: "task-log-cleanup", queue: "task_log_cleanup", project: true,
		short: "Configure the task_log_cleanup queue of a project",
		fields: []queueField{
			{"cleanup-period", stringField, "How often the cleanup runs, as an ISO 8601 duration (e.g. P1D)"},
			{"retention-period", stringField, "How long task logs are kept, as an ISO 8601 duration (e.g. P90D)"},
		},
		get: func(ctx context.Context, c *client.Client, _ string, callOpts []client.CallOption) (any, error) {
			return c.TaskQueues.TaskLogCleanupConfig(ctx, callOpts...)
		},
		set: func(ctx context.Context, c *client.Client, _ string, body []byte, callOpts []client.CallOption) error {
			var req managementv1.SetTaskLogCleanupConfig
			if err := json.Unmarshal(body, &req); err != nil {
				return err
			}
			return c.TaskQueues.SetTaskLogCleanupConfig(ctx, &req, callOpts...)
		},
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
		get: func(ctx context.Context, c *client.Client, wh string, _ []client.CallOption) (any, error) {
			return c.TaskQueues.ExpireSnapshotsConfig(ctx, wh)
		},
		set: func(ctx context.Context, c *client.Client, wh string, body []byte, _ []client.CallOption) error {
			var req managementv1.SetExpireSnapshotsQueueConfig
			if err := json.Unmarshal(body, &req); err != nil {
				return err
			}
			return c.TaskQueues.SetExpireSnapshotsConfig(ctx, wh, &req)
		},
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
		get: func(ctx context.Context, c *client.Client, wh string, _ []client.CallOption) (any, error) {
			return c.TaskQueues.OptimizeConfig(ctx, wh)
		},
		set: func(ctx context.Context, c *client.Client, wh string, body []byte, _ []client.CallOption) error {
			var req managementv1.SetOptimizeQueueConfig
			if err := json.Unmarshal(body, &req); err != nil {
				return err
			}
			return c.TaskQueues.SetOptimizeConfig(ctx, wh, &req)
		},
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
		get: func(ctx context.Context, c *client.Client, wh string, _ []client.CallOption) (any, error) {
			return c.TaskQueues.RemoveOrphanFilesConfig(ctx, wh)
		},
		set: func(ctx context.Context, c *client.Client, wh string, body []byte, _ []client.CallOption) error {
			var req managementv1.SetRemoveOrphanFilesQueueConfig
			if err := json.Unmarshal(body, &req); err != nil {
				return err
			}
			return c.TaskQueues.SetRemoveOrphanFilesConfig(ctx, wh, &req)
		},
	},
}

func newTaskQueueCmd(opts *clientOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "task-queue",
		Aliases: []string{"tq"},
		Short:   "Manage task queues",
		Long: "Manage task queues.\n\n" +
			"Each queue has get and set. Queues marked (Lakekeeper+) need a Lakekeeper+ server running the queue.",
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
		Aliases: []string{q.queue},
		Short:   short,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}
	cmd.AddCommand(newQueueGetCmd(opts, q))
	cmd.AddCommand(newQueueSetCmd(opts, q))
	return cmd
}

// queueTarget returns the positional arguments of a queue's get and set:
// a warehouse id, or none for a project queue.
func queueTarget(q taskQueue) (string, cobra.PositionalArgs) {
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
	use, args := queueTarget(q)
	short := "Print the " + q.queue + " queue configuration"
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
			cfg, err := q.get(ctx, c, firstArg(args), projectFlagOptions(cmd, project))
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
	use, args := queueTarget(q)
	short := "Set the " + q.queue + " queue configuration"
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
			if err := q.set(ctx, c, firstArg(args), body, projectFlagOptions(cmd, project)); err != nil {
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
		switch f.kind {
		case boolField:
			cmd.Flags().Bool(f.name, false, f.usage)
		case int64Field:
			cmd.Flags().Int64(f.name, 0, f.usage)
		case stringField:
			cmd.Flags().String(f.name, "", f.usage)
		}
	}
	if q.project {
		addProjectFlag(cmd, &project)
	}
	return cmd
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
		var (
			v   any
			err error
		)
		switch f.kind {
		case boolField:
			v, err = flags.GetBool(f.name)
		case int64Field:
			v, err = flags.GetInt64(f.name)
		case stringField:
			v, err = flags.GetString(f.name)
		}
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

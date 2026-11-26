package commands

import (
	"reflect"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"

	managementv1 "github.com/lakekeeper/go-lakekeeper/pkg/apis/management/v1"
	"github.com/lakekeeper/go-lakekeeper/pkg/client"
)

// The `set` flags of each queue are kept by hand, for their usage text.
// They must match the queue-config fields of the generated Set… request:
// the same JSON names, each with the flag type of its field.
func TestTaskQueueFields_MatchTheGeneratedQueueConfig(t *testing.T) {
	t.Parallel()
	requests := map[client.Queue]any{
		client.QueueTabularPurge:      managementv1.SetPurgeQueueConfig{},
		client.QueueSoftDeletion:      managementv1.SetSoftDeletionQueueConfig{},
		client.QueueTaskLogCleanup:    managementv1.SetTaskLogCleanupConfig{},
		client.QueueExpireSnapshots:   managementv1.SetExpireSnapshotsQueueConfig{},
		client.QueueOptimize:          managementv1.SetOptimizeQueueConfig{},
		client.QueueRemoveOrphanFiles: managementv1.SetRemoveOrphanFilesQueueConfig{},
	}
	assert.Len(t, taskQueues, len(requests))

	for _, q := range taskQueues {
		request, ok := requests[q.queue]
		if !assert.True(t, ok, "no generated request for %s", q.queue) {
			continue
		}
		config, ok := reflect.TypeOf(request).FieldByName("QueueConfig")
		if !assert.True(t, ok, q.queue) {
			continue
		}
		assert.Equal(t, generatedFlagTypes(config.Type), handKeptFlagTypes(q), q.queue)
	}
}

// generatedFlagTypes maps each JSON name of a queue-config struct to the
// flag type its field takes; a free-form map has none.
func generatedFlagTypes(config reflect.Type) map[string]string {
	types := map[string]string{}
	if config.Kind() != reflect.Struct {
		return types
	}
	for i := range config.NumField() {
		field := config.Field(i)
		name, _, _ := strings.Cut(field.Tag.Get("json"), ",")
		if name == "" || name == "-" || name == "queue-name" {
			continue
		}
		// *bool, *int64, …, or NullableBool, NullableInt32, …
		ft := field.Type
		if ft.Kind() == reflect.Pointer {
			ft = ft.Elem()
		}
		typ := strings.ToLower(strings.TrimPrefix(ft.Name(), "Nullable"))
		switch typ {
		case "int32", "int64":
			types[name] = "int64"
		default:
			types[name] = typ
		}
	}
	return types
}

// handKeptFlagTypes maps each of a queue's fields to the type of the flag
// its kind defines.
func handKeptFlagTypes(q taskQueue) map[string]string {
	types := map[string]string{}
	cmd := &cobra.Command{}
	for _, f := range q.fields {
		f.kind.define(cmd, f.name, f.usage)
		types[f.name] = cmd.Flags().Lookup(f.name).Value.Type()
	}
	return types
}

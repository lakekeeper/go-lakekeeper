//go:build e2e_cli

// Setup and read-back that lkctl has no command for: tables (the Iceberg
// catalog) and the task list of a warehouse. Both go through the SDK, like
// the bootstrap in TestMain.

package clie2e

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/apache/iceberg-go"
	"github.com/apache/iceberg-go/table"
	"github.com/google/uuid"
	"golang.org/x/oauth2/clientcredentials"

	managementv1 "github.com/lakekeeper/go-lakekeeper/pkg/apis/management/v1"
	"github.com/lakekeeper/go-lakekeeper/pkg/client"
	"github.com/lakekeeper/go-lakekeeper/pkg/core"
)

// sdkClient returns an SDK client authenticated as the admin of the stack.
func sdkClient(t *testing.T) *client.Client {
	t.Helper()
	cfg := clientcredentials.Config{
		ClientID:     os.Getenv("LAKEKEEPER_CLIENT_ID"),
		ClientSecret: os.Getenv("LAKEKEEPER_CLIENT_SECRET"),
		TokenURL:     os.Getenv("LAKEKEEPER_TOKEN_URL"),
		Scopes:       strings.Fields(os.Getenv("LAKEKEEPER_SCOPE")),
	}
	c, err := client.NewWithAuthSource(t.Context(), os.Getenv("LAKEKEEPER_BASE_URL"),
		&core.OAuthClientCredentialsAuthSource{TokenSource: cfg.TokenSource(t.Context())})
	if err != nil {
		t.Fatalf("create SDK client: %v", err)
	}
	return c
}

// MustProvisionTable creates a table in a new namespace of the warehouse
// (in the default project) and returns the table's id. Cleanup drops both;
// register it after the warehouse, so it runs before the warehouse delete.
func MustProvisionTable(t *testing.T, warehouseName string) string {
	t.Helper()
	cat, err := sdkClient(t).CatalogV1(t.Context(), uuid.Nil.String(), warehouseName)
	if err != nil {
		t.Fatalf("catalog for warehouse %s: %v", warehouseName, err)
	}

	ns := table.Identifier{strings.ReplaceAll(randomName("e2e_ns"), "-", "_")}
	if err := cat.CreateNamespace(t.Context(), ns, nil); err != nil {
		t.Fatalf("create namespace: %v", err)
	}
	ident := append(append(table.Identifier{}, ns...), "events")
	tbl, err := cat.CreateTable(t.Context(), ident, iceberg.NewSchema(0,
		iceberg.NestedField{ID: 1, Name: "id", Type: iceberg.PrimitiveTypes.Int64, Required: true}))
	if err != nil {
		t.Fatalf("create table: %v", err)
	}

	t.Cleanup(func() {
		// t.Context is already canceled when cleanups run.
		ctx := context.Background()
		if err := cat.DropTable(ctx, ident); err != nil {
			t.Errorf("cleanup: drop table: %v", err)
		}
		if err := cat.DropNamespace(ctx, ns); err != nil {
			t.Errorf("cleanup: drop namespace: %v", err)
		}
	})
	return tbl.Metadata().TableUUID().String()
}

// warehouseTasks returns the tasks of a warehouse's queue.
func warehouseTasks(t *testing.T, warehouseID, queue string) []managementv1.WarehouseTaskInfo {
	t.Helper()
	c := sdkClient(t)
	out, resp, err := c.TasksAPI.ListTasks(t.Context(), warehouseID).
		ListTasksRequest(managementv1.ListTasksRequest{QueueName: []string{queue}}).Execute()
	if err != nil {
		t.Fatalf("list tasks: %v", client.WrapAPIError("list tasks", err, resp))
	}
	return out.Tasks
}

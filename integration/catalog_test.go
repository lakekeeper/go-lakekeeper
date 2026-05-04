//go:build integration

package integration

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestCatalog_Basic exercises the Iceberg REST catalog delegation in
// pkg/client.CatalogV1: it must succeed for warehouses on both the default
// project and on a freshly-created project.
func TestCatalog_Basic(t *testing.T) {
	c := sharedClient

	project := MustCreateProject(t, c)

	_, defaultWh := MustCreateWarehouse(t, c, defaultProjectID)
	_, err := c.CatalogV1(t.Context(), defaultProjectID, defaultWh)
	require.NoError(t, err)

	_, projectWh := MustCreateWarehouse(t, c, project)
	_, err = c.CatalogV1(t.Context(), project, projectWh)
	require.NoError(t, err)
}

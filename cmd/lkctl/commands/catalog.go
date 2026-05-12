package commands

import (
	"errors"

	"github.com/spf13/cobra"
)

// newCatalogCmd returns a placeholder for the future Iceberg catalog
// integration. The real surface delegates to apache/iceberg-go via
// client.CatalogV1; until the CLI ergonomics are designed, the command exists
// only to reserve the verb.
func newCatalogCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "catalog",
		Short: "Interact with the Iceberg catalog (not implemented)",
		RunE: func(*cobra.Command, []string) error {
			return errors.New("catalog command is not implemented")
		},
	}
}

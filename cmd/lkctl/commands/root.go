// Package commands hosts the lkctl CLI.
//
// The CLI was rewritten in conjunction with the move to OpenAPI-generated
// types and is being reintroduced incrementally as commands are migrated to
// the new managementv1 client. For now, the binary builds and exposes a
// version-only root command so the release pipeline keeps producing
// artifacts.
package commands

import (
	"github.com/spf13/cobra"

	"github.com/lakekeeper/go-lakekeeper/pkg/version"
)

// NewCommand returns the lkctl root command.
func NewCommand() *cobra.Command {
	root := &cobra.Command{
		Use:           "lkctl",
		Short:         "Command-line client for the Lakekeeper Iceberg catalog.",
		Version:       version.GetVersion().Version,
		SilenceErrors: true,
		SilenceUsage:  true,
	}
	return root
}

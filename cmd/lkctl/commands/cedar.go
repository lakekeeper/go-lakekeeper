package commands

import (
	"fmt"
	"io"
	"maps"
	"slices"
	"strings"

	"github.com/spf13/cobra"
)

// lakekeeperPlus marks a command's short help as needing Lakekeeper+.
// Lakekeeper+ commands are always registered; this marking is how help
// tells them apart, without contacting a server.
func lakekeeperPlus(short string) string {
	return short + " (Lakekeeper+)"
}

func newCedarCmd(opts *clientOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cedar",
		Short: lakekeeperPlus("Manage Cedar policies"),
		Long: lakekeeperPlus("Manage Cedar policies") + "\n\n" +
			"Needs Lakekeeper+ with the Cedar authorization backend.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}

	cmd.AddCommand(newCedarSchemaCmd(opts))
	cmd.AddCommand(newCedarScopeCmd(opts, projectCedarScope))
	cmd.AddCommand(newCedarScopeCmd(opts, warehouseCedarScope))
	cmd.AddCommand(newCedarBreakGlassStatusCmd(opts))
	cmd.AddCommand(newCedarPolicySourcesCmd(opts))
	cmd.AddCommand(newCedarEntitySourcesCmd(opts))
	cmd.AddCommand(newCedarServerPoliciesCmd(opts))
	cmd.AddCommand(newCedarResolveEntitiesCmd(opts))

	return cmd
}

func newCedarSchemaCmd(opts *clientOptions) *cobra.Command {
	var output string

	cmd := &cobra.Command{
		Use:   "schema",
		Short: lakekeeperPlus("Print the active Cedar schema"),
		Long: lakekeeperPlus("Print the active Cedar schema") + "\n\n" +
			"Text output prints each schema fragment under a comment naming its file.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			c, err := newClient(ctx, opts)
			if err != nil {
				return err
			}
			schema, err := c.Cedar.Schema(ctx)
			if err != nil {
				return err
			}

			switch output {
			case "json":
				return printJSON(cmd.OutOrStdout(), schema)
			case "text":
				return printCedarSchema(cmd.OutOrStdout(), schema.GetSchemaFragments())
			default:
				return fmt.Errorf("unknown output format: %s", output)
			}
		},
	}

	cmd.Flags().StringVarP(&output, "output", "o", "text", "Output format. One of: json|text")
	return cmd
}

// printCedarSchema prints the fragments in path order, each under a "// <path>"
// line (a Cedar comment), separated by blank lines.
func printCedarSchema(w io.Writer, fragments map[string]string) error {
	for i, p := range slices.Sorted(maps.Keys(fragments)) {
		if i > 0 {
			fmt.Fprintln(w)
		}
		fmt.Fprintf(w, "// %s\n%s\n", p, strings.TrimRight(fragments[p], "\n"))
	}
	return nil
}

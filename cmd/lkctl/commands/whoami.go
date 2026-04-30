package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

// newWhoamiCmd returns `lkctl whoami`, which prints the user identified by
// the access token currently in use.
func newWhoamiCmd(opts *clientOptions) *cobra.Command {
	var output string

	cmd := &cobra.Command{
		Use:   "whoami",
		Short: "Print the current user",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()

			c, err := newClient(ctx, opts)
			if err != nil {
				return err
			}
			user, _, err := c.UserAPI.Whoami(ctx).Execute()
			if err != nil {
				return fmt.Errorf("whoami: %w", err)
			}

			switch output {
			case "json":
				return printJSON(cmd.OutOrStdout(), user)
			case "text", "wide":
				return printUsers(cmd.OutOrStdout(), output, nil, user)
			default:
				return fmt.Errorf("unknown output format: %s", output)
			}
		},
	}

	cmd.Flags().StringVarP(&output, "output", "o", "text", "Output format. One of: json|text|wide")
	return cmd
}

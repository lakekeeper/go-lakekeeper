package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	managementv1 "github.com/lakekeeper/go-lakekeeper/pkg/apis/management/v1"
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
				return wrapAPIError("whoami", err)
			}

			switch output {
			case "json":
				return printJSON(cmd.OutOrStdout(), user)
			case "text", "wide":
				return printUsers(cmd.OutOrStdout(), output, nil, whoamiUser(user))
			default:
				return fmt.Errorf("unknown output format: %s", output)
			}
		},
	}

	cmd.Flags().StringVarP(&output, "output", "o", "text", "Output format. One of: json|text|wide")
	return cmd
}

// whoamiUser projects a whoami response onto the User fields printUsers shows.
func whoamiUser(w *managementv1.WhoamiResponse) *managementv1.User {
	u := managementv1.NewUser(w.CreatedAt, w.Id, w.LastUpdatedWith, w.Name, w.UserType)
	u.Email.Set(w.Email)
	u.UpdatedAt.Set(w.UpdatedAt)
	return u
}

package commands

import (
	"fmt"
	"io"
	"text/tabwriter"
	"time"

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

			if output == "json" {
				return printJSON(cmd.OutOrStdout(), user)
			}
			return printWhoami(cmd.OutOrStdout(), output, user)
		},
	}

	cmd.Flags().StringVarP(&output, "output", "o", "text", "Output format. One of: json|text|wide")
	return cmd
}

// printWhoami writes a tabular representation of the current user to w.
// Supported formats mirror printUsers, plus the instance-admin flag that only
// the whoami response carries:
//   - "text": ID, NAME, EMAIL, USER TYPE, INSTANCE ADMIN
//   - "wide": adds CREATED AT, UPDATED AT, LAST UPDATED WITH
func printWhoami(w io.Writer, output string, u *managementv1.WhoamiResponse) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	switch output {
	case "text":
		fmt.Fprintln(tw, "ID\tNAME\tEMAIL\tUSER TYPE\tINSTANCE ADMIN")
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%t\n",
			u.Id, u.Name, formatStringPtr(u.Email), u.UserType, u.IsInstanceAdmin)
	case "wide":
		fmt.Fprintln(tw, "ID\tNAME\tEMAIL\tUSER TYPE\tINSTANCE ADMIN\tCREATED AT\tUPDATED AT\tLAST UPDATED WITH")
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%t\t%s\t%s\t%s\n",
			u.Id, u.Name, formatStringPtr(u.Email), u.UserType, u.IsInstanceAdmin,
			u.CreatedAt.Format(time.RFC3339), formatTimePtr(u.UpdatedAt), u.LastUpdatedWith)
	default:
		return fmt.Errorf("unknown output format: %s", output)
	}
	return tw.Flush()
}

package commands

import (
	"fmt"
	"io"
	"text/tabwriter"
	"time"

	managementv1 "github.com/lakekeeper/go-lakekeeper/pkg/apis/management/v1"
)

// printUsers writes a tabular representation of users to w. Supported formats:
//   - "text": ID, NAME, EMAIL, USER TYPE
//   - "wide": adds CREATED AT, UPDATED AT, LAST UPDATED WITH
//
// If nextPageToken is non-nil, it is printed after the table.
func printUsers(w io.Writer, output string, nextPageToken *string, users ...*managementv1.User) error {
	switch output {
	case "text":
		tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
		fmt.Fprintln(tw, "ID\tNAME\tEMAIL\tUSER TYPE")
		for _, u := range users {
			fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", u.Id, u.Name, formatNullableString(u.Email), u.UserType)
		}
		if err := tw.Flush(); err != nil {
			return err
		}
	case "wide":
		tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
		fmt.Fprintln(tw, "ID\tNAME\tEMAIL\tUSER TYPE\tCREATED AT\tUPDATED AT\tLAST UPDATED WITH")
		for _, u := range users {
			fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
				u.Id, u.Name, formatNullableString(u.Email), u.UserType,
				u.CreatedAt.Format(time.RFC3339), formatNullableTime(u.UpdatedAt), u.LastUpdatedWith)
		}
		if err := tw.Flush(); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unknown output format: %s", output)
	}
	if nextPageToken != nil {
		fmt.Fprintf(w, "\nNext page token: %s\n", *nextPageToken)
	}
	return nil
}

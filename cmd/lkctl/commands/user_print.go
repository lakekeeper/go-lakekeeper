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
// If nextPageToken is non-empty, it is printed after the table.
func printUsers(w io.Writer, output, nextPageToken string, users ...managementv1.User) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	switch output {
	case "text":
		fmt.Fprintln(tw, "ID\tNAME\tEMAIL\tUSER TYPE")
		for _, u := range users {
			fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", u.GetId(), u.GetName(), u.GetEmail(), u.GetUserType())
		}
	case "wide":
		fmt.Fprintln(tw, "ID\tNAME\tEMAIL\tUSER TYPE\tCREATED AT\tUPDATED AT\tLAST UPDATED WITH")
		for _, u := range users {
			updatedAt, _ := u.GetUpdatedAtOk()
			fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
				u.GetId(), u.GetName(), u.GetEmail(), u.GetUserType(),
				u.GetCreatedAt().Format(time.RFC3339), formatTimePtr(updatedAt), u.GetLastUpdatedWith())
		}
	default:
		return fmt.Errorf("unknown output format: %s", output)
	}
	if err := tw.Flush(); err != nil {
		return err
	}
	if nextPageToken != "" {
		fmt.Fprintf(w, "\nNext page token: %s\n", nextPageToken)
	}
	return nil
}

package commands

import (
	"encoding/json"
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

// printSearchUsers writes the users of a search response as a table: ID,
// NAME, EMAIL, USER TYPE.
func printSearchUsers(w io.Writer, users ...managementv1.SearchUser) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "ID\tNAME\tEMAIL\tUSER TYPE")
	for _, u := range users {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", u.GetId(), u.GetName(), u.GetEmail(), u.GetUserType())
	}
	return tw.Flush()
}

// printRoleMemberships writes role memberships as a table: ID, NAME, IDENT.
// If nextPageToken is non-empty, it is printed after the table.
func printRoleMemberships(w io.Writer, nextPageToken string, roles ...managementv1.RoleMembership) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "ID\tNAME\tIDENT")
	for _, r := range roles {
		fmt.Fprintf(tw, "%s\t%s\t%s\n", r.GetId(), r.GetName(), r.GetIdent())
	}
	if err := tw.Flush(); err != nil {
		return err
	}
	if nextPageToken != "" {
		fmt.Fprintf(w, "\nNext page token: %s\n", nextPageToken)
	}
	return nil
}

// userActionNames returns the action name of each allowed user action. The
// generated LakekeeperUserAction is a oneOf with one struct per action, each
// serialized as {"action": NAME}; reading the name from that form keeps new
// actions printable without a code change.
func userActionNames(actions []managementv1.LakekeeperUserAction) ([]string, error) {
	names := make([]string, 0, len(actions))
	for _, a := range actions {
		raw, err := json.Marshal(a)
		if err != nil {
			return nil, fmt.Errorf("decode user action: %w", err)
		}
		var v struct {
			Action string `json:"action"`
		}
		if err := json.Unmarshal(raw, &v); err != nil {
			return nil, fmt.Errorf("decode user action: %w", err)
		}
		names = append(names, v.Action)
	}
	return names, nil
}

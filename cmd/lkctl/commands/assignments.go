package commands

import (
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/lakekeeper/go-lakekeeper/pkg/permissions"
)

// printAssignments writes a tabular listing of permission assignments to w.
// Empty input writes "No assignments". The function is generic over the
// resource-specific assignment type (ServerAssignment, ProjectAssignment, …);
// each value is decoded via permissions.DescribeAssignment.
func printAssignments[T any](w io.Writer, assignments ...T) error {
	if len(assignments) == 0 {
		_, err := fmt.Fprintln(w, "No assignments")
		return err
	}

	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "PRINCIPAL TYPE\tPRINCIPAL ID\tASSIGNMENT")
	dropped := 0
	for _, a := range assignments {
		row, ok := permissions.DescribeAssignment(a)
		if !ok {
			dropped++
			continue
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\n", displayPrincipalType(row.PrincipalType), row.PrincipalID, row.Relation)
	}
	if err := tw.Flush(); err != nil {
		return err
	}
	if dropped > 0 {
		fmt.Fprintf(w, "(warning: %d assignment(s) could not be decoded)\n", dropped)
	}
	return nil
}

// displayPrincipalType title-cases the wire-format principal type for table
// display ("user" → "User", "role" → "Role"). Anything else is passed through
// unchanged, since it shouldn't reach the printer.
func displayPrincipalType(s string) string {
	switch s {
	case "user":
		return "User"
	case "role":
		return "Role"
	default:
		return s
	}
}

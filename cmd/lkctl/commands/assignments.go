package commands

import (
	"encoding/json"
	"fmt"
	"io"
	"text/tabwriter"
)

// assignmentRow is the flattened, principal-agnostic view of a single
// permission assignment used by the `assignments` subcommands. Every
// resource-specific assignment in the generated client serializes to
// `{type, user|role}` on the wire, which is what describeAssignment
// extracts.
type assignmentRow struct {
	PrincipalType string
	PrincipalID   string
	Relation      string
}

// describeAssignment converts any generated *Assignment union value into an
// assignmentRow by JSON-roundtripping through the wire shape they all share:
// `{type, user?, role?}`.
//
// Returns false if the value can't be marshalled or carries no principal
// payload (e.g., the union is empty / unset).
func describeAssignment(a any) (assignmentRow, bool) {
	b, err := json.Marshal(a)
	if err != nil {
		return assignmentRow{}, false
	}
	var raw struct {
		Type string `json:"type"`
		User string `json:"user,omitempty"`
		Role string `json:"role,omitempty"`
	}
	if err := json.Unmarshal(b, &raw); err != nil {
		return assignmentRow{}, false
	}
	if raw.Type == "" || (raw.User == "" && raw.Role == "") {
		return assignmentRow{}, false
	}
	row := assignmentRow{Relation: raw.Type}
	switch {
	case raw.User != "":
		row.PrincipalType = "User"
		row.PrincipalID = raw.User
	case raw.Role != "":
		row.PrincipalType = "Role"
		row.PrincipalID = raw.Role
	}
	return row, true
}

// printAssignments writes a tabular listing of permission assignments to w.
// Empty input writes "No assignments". The function is generic over the
// resource-specific assignment type (ServerAssignment, ProjectAssignment, …);
// each value is decoded via describeAssignment.
func printAssignments[T any](w io.Writer, assignments ...T) error {
	if len(assignments) == 0 {
		_, err := fmt.Fprintln(w, "No assignments")
		return err
	}

	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "PRINCIPAL TYPE\tPRINCIPAL ID\tASSIGNMENT")
	dropped := 0
	for _, a := range assignments {
		row, ok := describeAssignment(a)
		if !ok {
			dropped++
			continue
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\n", row.PrincipalType, row.PrincipalID, row.Relation)
	}
	if err := tw.Flush(); err != nil {
		return err
	}
	if dropped > 0 {
		fmt.Fprintf(w, "(warning: %d assignment(s) could not be decoded)\n", dropped)
	}
	return nil
}

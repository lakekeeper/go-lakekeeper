// Package permissions provides helpers for constructing and inspecting the
// generated *Assignment union types in pkg/apis/management/v1.
//
// Every resource-specific assignment in the generated client (Server,
// Project, Warehouse, Role) serializes to the same wire shape:
// `{"type": <relation>, "user"|"role": <id>}`. The helpers here consume and
// produce that wire shape generically, so callers can build and decode
// assignments without hard-coding which discriminator branch each relation
// belongs to.
package permissions

import (
	"encoding/json"
	"errors"
	"fmt"
)

// PrincipalKind discriminates between user and role principals when building
// permission-update payloads. The zero value is intentionally invalid so a
// forgotten kind surfaces as an error from BuildAssignment rather than as a
// silent user-principal payload.
type PrincipalKind int

const (
	_ PrincipalKind = iota
	PrincipalUser
	PrincipalRole
)

// AssignmentRow is the flattened, principal-agnostic projection of a single
// permission assignment.
type AssignmentRow struct {
	// PrincipalType is the wire-format principal kind ("user" or "role"),
	// lowercase. Display layers (e.g. lkctl's table printer) capitalize as
	// needed; do not capitalize here so equality checks against the wire
	// value stay direct in tests using assert.ElementsMatch.
	PrincipalType string
	PrincipalID   string
	Relation      string
}

// BuildAssignment constructs a generated *Assignment union value for a given
// relation, principal kind, and principal ID. It works generically across
// resource types because the generated UnmarshalJSON for each *Assignment
// dispatches on the `type` discriminator and decodes into the matching
// variant — the same wire shape that DescribeAssignment reads back out.
func BuildAssignment[T any](relation string, kind PrincipalKind, id string) (T, error) {
	var zero T
	if relation == "" {
		return zero, errors.New("relation must not be empty")
	}
	if id == "" {
		return zero, errors.New("principal id must not be empty")
	}

	payload := map[string]string{"type": relation}
	switch kind {
	case PrincipalUser:
		payload["user"] = id
	case PrincipalRole:
		payload["role"] = id
	default:
		return zero, fmt.Errorf("unknown principal kind %d", kind)
	}

	b, err := json.Marshal(payload)
	if err != nil {
		return zero, fmt.Errorf("marshal assignment payload: %w", err)
	}

	var out T
	if err := json.Unmarshal(b, &out); err != nil {
		return zero, fmt.Errorf("invalid relation %q for this resource: %w", relation, err)
	}

	// The generated *Assignment.UnmarshalJSON falls through with no error when
	// the `type` discriminator matches no known variant — leaving `out` as a
	// zero-valued union. Verify by re-decoding via DescribeAssignment, which
	// only returns ok when the result carries a real `{type, user|role}`
	// payload.
	if _, ok := DescribeAssignment(out); !ok {
		return zero, fmt.Errorf("unknown relation %q for this resource", relation)
	}
	return out, nil
}

// PrincipalSet groups user and role ids for a multi-relation grant or
// revoke. It is the input shape for BuildAssignmentSet, which expands the
// (relation × principal) cartesian product into a slice of generated
// *Assignment values.
type PrincipalSet struct {
	Users []string
	Roles []string
}

// BuildAssignmentSet expands `relations × (set.Users + set.Roles)` into a
// slice of generated *Assignment values for assignment to a request's
// Writes (grant) or Deletes (revoke) field. Returns the first error if any
// single assignment fails to build (e.g., an unknown relation for the
// resource type).
//
// The four-line nested loop this replaces appears once per resource per
// verb (warehouse/project/role/server × grant/revoke), so consolidating it
// here removes ~120 LoC from lkctl alone.
func BuildAssignmentSet[T any](relations []string, set PrincipalSet) ([]T, error) {
	if len(set.Users) == 0 && len(set.Roles) == 0 {
		return nil, errors.New("no principals: set.Users and set.Roles are both empty")
	}
	out := make([]T, 0, len(relations)*(len(set.Users)+len(set.Roles)))
	for _, rel := range relations {
		for _, u := range set.Users {
			a, err := BuildAssignment[T](rel, PrincipalUser, u)
			if err != nil {
				return nil, err
			}
			out = append(out, a)
		}
		for _, r := range set.Roles {
			a, err := BuildAssignment[T](rel, PrincipalRole, r)
			if err != nil {
				return nil, err
			}
			out = append(out, a)
		}
	}
	return out, nil
}

// DescribeAssignment converts any generated *Assignment union value into an
// AssignmentRow by JSON-roundtripping through the wire shape they all share:
// `{type, user?, role?}`.
//
// Returns false if the value can't be marshalled or carries no principal
// payload (e.g., the union is empty / unset).
func DescribeAssignment(a any) (AssignmentRow, bool) {
	b, err := json.Marshal(a)
	if err != nil {
		return AssignmentRow{}, false
	}
	var raw struct {
		Type string `json:"type"`
		User string `json:"user,omitempty"`
		Role string `json:"role,omitempty"`
	}
	if err := json.Unmarshal(b, &raw); err != nil {
		return AssignmentRow{}, false
	}
	if raw.Type == "" || (raw.User == "" && raw.Role == "") {
		return AssignmentRow{}, false
	}
	row := AssignmentRow{Relation: raw.Type}
	switch {
	case raw.User != "":
		row.PrincipalType = "user"
		row.PrincipalID = raw.User
	case raw.Role != "":
		row.PrincipalType = "role"
		row.PrincipalID = raw.Role
	}
	return row, true
}

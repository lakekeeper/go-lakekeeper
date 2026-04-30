package commands

import (
	"encoding/json"
	"errors"
	"fmt"
)

// principalKind discriminates between user and role principals when building
// permission-update payloads for the grant subcommands.
type principalKind int

const (
	principalUser principalKind = iota
	principalRole
)

// buildAssignment constructs a generated *Assignment union value for a given
// relation, principal kind, and principal ID. It works generically across
// resource types (server, project, role, warehouse) because the generated
// UnmarshalJSON for each *Assignment dispatches on the `type` discriminator
// and decodes into the matching variant — the same wire shape that
// describeAssignment reads back out.
func buildAssignment[T any](relation string, kind principalKind, id string) (T, error) {
	var zero T
	if relation == "" {
		return zero, errors.New("relation must not be empty")
	}
	if id == "" {
		return zero, errors.New("principal id must not be empty")
	}

	payload := map[string]string{"type": relation}
	switch kind {
	case principalUser:
		payload["user"] = id
	case principalRole:
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
	// zero-valued union. Verify by re-decoding via describeAssignment, which
	// only returns ok when the result carries a real `{type, user|role}`
	// payload.
	if _, ok := describeAssignment(out); !ok {
		return zero, fmt.Errorf("unknown relation %q for this resource", relation)
	}
	return out, nil
}

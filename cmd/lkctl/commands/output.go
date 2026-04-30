package commands

import (
	"encoding/json"
	"fmt"
	"io"
)

// printJSON writes a JSON-encoded representation of v to w, indented with two
// spaces and terminated by a newline.
func printJSON(w io.Writer, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal json: %w", err)
	}
	_, err = fmt.Fprintln(w, string(b))
	return err
}

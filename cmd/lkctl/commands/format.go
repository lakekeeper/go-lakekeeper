package commands

import "time"

// formatStringPtr returns the string value pointed to, or an empty string for
// nil. Used for the optional response fields the generator emits as *string
// after the preprocessor strips OAS-3.1 nullable type-arrays.
func formatStringPtr(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

// formatTimePtr returns the RFC3339-formatted time pointed to, or an empty
// string for nil.
func formatTimePtr(p *time.Time) string {
	if p == nil {
		return ""
	}
	return p.Format(time.RFC3339)
}

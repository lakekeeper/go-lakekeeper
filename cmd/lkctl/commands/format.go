package commands

import "time"

// formatStringPtr returns the string value pointed to, or an empty string for
// nil. Used with the Get() of the generated NullableString wrappers, which
// return nil for absent or null response fields.
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

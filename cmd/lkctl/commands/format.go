package commands

import (
	"time"

	managementv1 "github.com/lakekeeper/go-lakekeeper/pkg/apis/management/v1"
)

// formatNullableString returns the string value held by a NullableString, or
// an empty string if the value is nil/unset.
func formatNullableString(v managementv1.NullableString) string {
	if !v.IsSet() || v.Get() == nil {
		return ""
	}
	return *v.Get()
}

// formatNullableTime returns the RFC3339-formatted time held by a NullableTime,
// or an empty string if the value is nil/unset.
func formatNullableTime(v managementv1.NullableTime) string {
	if !v.IsSet() || v.Get() == nil {
		return ""
	}
	return v.Get().Format(time.RFC3339)
}

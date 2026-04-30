package commands

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	managementv1 "github.com/lakekeeper/go-lakekeeper/pkg/apis/management/v1"
)

func TestFormatNullableString(t *testing.T) {
	t.Parallel()

	var unset managementv1.NullableString
	assert.Empty(t, formatNullableString(unset))

	set := *managementv1.NewNullableString(managementv1.PtrString("hello"))
	assert.Equal(t, "hello", formatNullableString(set))

	null := *managementv1.NewNullableString(nil)
	assert.Empty(t, formatNullableString(null))
}

func TestFormatNullableTime(t *testing.T) {
	t.Parallel()

	var unset managementv1.NullableTime
	assert.Empty(t, formatNullableTime(unset))

	ts := time.Date(2026, 4, 30, 12, 34, 56, 0, time.UTC)
	set := *managementv1.NewNullableTime(&ts)
	assert.Equal(t, ts.Format(time.RFC3339), formatNullableTime(set))

	null := *managementv1.NewNullableTime(nil)
	assert.Empty(t, formatNullableTime(null))
}

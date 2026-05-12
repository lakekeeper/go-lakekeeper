package commands

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestFormatStringPtr(t *testing.T) {
	t.Parallel()

	assert.Empty(t, formatStringPtr(nil))

	val := "hello"
	assert.Equal(t, "hello", formatStringPtr(&val))

	empty := ""
	assert.Empty(t, formatStringPtr(&empty))
}

func TestFormatTimePtr(t *testing.T) {
	t.Parallel()

	assert.Empty(t, formatTimePtr(nil))

	ts := time.Date(2026, 4, 30, 12, 34, 56, 0, time.UTC)
	assert.Equal(t, ts.Format(time.RFC3339), formatTimePtr(&ts))
}

package commands

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/lakekeeper/go-lakekeeper/pkg/version"
)

func TestPrintClientVersionShort(t *testing.T) {
	t.Parallel()

	v := version.Version{Version: "v1.2.3"}
	got := printClientVersion(&v, true)
	assert.Equal(t, "lkctl: v1.2.3\n", got)
}

func TestPrintServerVersion(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "lakekeeper: 0.9.1\n", printServerVersion("0.9.1"))
}

func TestPrintClientVersionLong(t *testing.T) {
	t.Parallel()

	v := version.Version{
		Version:      "v1.2.3",
		BuildDate:    "2026-04-30",
		GitCommit:    "deadbeef",
		GitTreeState: "clean",
		GitTag:       "v1.2.3",
		GoVersion:    "go1.24",
		Compiler:     "gc",
		Platform:     "darwin/arm64",
	}
	got := printClientVersion(&v, false)
	assert.Contains(t, got, "lkctl: ")
	assert.Contains(t, got, "BuildDate: 2026-04-30")
	assert.Contains(t, got, "GitCommit: deadbeef")
	assert.Contains(t, got, "GitTreeState: clean")
	assert.Contains(t, got, "GitTag: v1.2.3")
	assert.Contains(t, got, "GoVersion: go1.24")
	assert.Contains(t, got, "Platform: darwin/arm64")
}

func TestPrintClientVersionLongOmitsEmptyGitTag(t *testing.T) {
	t.Parallel()

	v := version.Version{
		Version:      "v1.2.3+abcdef.dirty",
		BuildDate:    "2026-04-30",
		GitCommit:    "abcdef",
		GitTreeState: "dirty",
		// GitTag intentionally empty — dev build, never tagged.
		GoVersion: "go1.24",
		Compiler:  "gc",
		Platform:  "darwin/arm64",
	}
	got := printClientVersion(&v, false)
	assert.NotContains(t, got, "GitTag")
}

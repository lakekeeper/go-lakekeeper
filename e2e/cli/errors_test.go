//go:build e2e_cli

// Negative paths: bad credentials, missing required flags, unknown resource.
// Each asserts non-zero exit and stderr (or combined output) contains a
// recognisable signal so the operator gets a useful failure.

package clie2e

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBadCredentialsExitNonZero(t *testing.T) {
	t.Parallel()

	args := []string{
		"--base-url", os.Getenv("LAKEKEEPER_BASE_URL"),
		"--auth-mode", "token",
		"--access-token", "definitely-not-a-real-token",
		"whoami",
	}
	stdout, stderr, code := runRaw(t, nil, args...)
	assert.NotZero(t, code, "expected non-zero exit on bad credentials")
	combined := string(stdout) + string(stderr)
	assert.NotEmpty(t, combined, "expected diagnostic output on auth failure")
}

func TestMissingRequiredFlag(t *testing.T) {
	t.Parallel()

	// `warehouse create NAME` requires --file; cobra exits with code 1 when
	// a required flag is missing (exit code 2 would be the convention but
	// cobra itself uses 1 in this codebase). Runs on every backend because
	// cobra exits before contacting the server.
	args := append(authFlagsOAuth2(), "warehouse", "create", "doesnotmatter")
	stdout, stderr, code := runRaw(t, nil, args...)
	assert.NotZero(t, code, "expected non-zero exit when --file is missing")
	combined := string(stdout) + string(stderr)
	assert.Contains(t, combined, "file", "expected diagnostic mentioning the missing --file flag")
}

func TestNotFoundResourceExitNonZero(t *testing.T) {
	requireBackend(t, BackendCompose)
	t.Parallel()

	// All-zeros UUID is the *default* project — present and accessible —
	// so we use a non-existent random one to provoke a not-found path.
	missing := "11111111-2222-3333-4444-555555555555"
	stdout, stderr, code := runRaw(t, nil,
		append(authFlagsOAuth2(), "project", "get", missing, "--output", "json")...,
	)
	assert.NotZero(t, code, "expected non-zero exit for missing project")
	combined := string(stdout) + string(stderr)
	assert.NotEmpty(t, combined)
}

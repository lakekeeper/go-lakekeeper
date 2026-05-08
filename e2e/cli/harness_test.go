//go:build e2e_cli

// Package clie2e drives the lkctl binary against a live Lakekeeper stack.
// Selected by build tag e2e_cli (i.e. invoked from e2e/compose/run.sh or
// e2e/kind/run.sh, never as part of unit tests).
//
// SDK-level coverage of the underlying client lives in ./integration/.
// These tests exercise the *CLI surface only*: every assertion is on lkctl
// stdout/stderr/exit, never on a Go API result. Setup that requires the SDK
// (initial bootstrap) happens in TestMain, but per-test state is read back
// through `lkctl get`/`list`.
package clie2e

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"golang.org/x/oauth2/clientcredentials"

	managementv1 "github.com/lakekeeper/go-lakekeeper/pkg/apis/management/v1"
	"github.com/lakekeeper/go-lakekeeper/pkg/client"
	"github.com/lakekeeper/go-lakekeeper/pkg/core"
)

var (
	backendOnce   sync.Once
	activeBackend Backend
	backendErr    error
)

// TestMain locates the repo .env (compose only), boots the configured backend,
// and bootstraps the Lakekeeper server via the SDK so every test starts in a
// known good state. Per-test side-effects are validated through the CLI
// surface only.
func TestMain(m *testing.M) {
	// .env lives at repo root for the compose backend; harmless to skip
	// when running under kind (env vars are passed in by run.sh).
	_ = godotenv.Load(repoRootRelative(".env"))

	if err := initBackend(); err != nil {
		fmt.Fprintf(os.Stderr, "init e2e backend: %v\n", err)
		os.Exit(1)
	}

	if activeBackend.Name() == BackendCompose {
		if err := bootstrapForCompose(); err != nil {
			fmt.Fprintf(os.Stderr, "bootstrap: %v\n", err)
			os.Exit(1)
		}
	}

	code := m.Run()

	if activeBackend != nil {
		activeBackend.Close()
	}
	os.Exit(code)
}

// initBackend selects the backend implementation per LKCTL_E2E_BACKEND.
// Defaults to compose so a bare `go test -tags e2e_cli ./e2e/cli/...` run
// against an already-up stack works.
func initBackend() error {
	backendOnce.Do(func() {
		mode := os.Getenv("LKCTL_E2E_BACKEND")
		if mode == "" {
			mode = BackendCompose
		}
		switch mode {
		case BackendCompose:
			activeBackend, backendErr = newComposeBackend()
		case BackendKind:
			activeBackend, backendErr = newKindBackend()
		default:
			backendErr = fmt.Errorf("unknown LKCTL_E2E_BACKEND %q", mode)
		}
	})
	return backendErr
}

// bootstrapForCompose mints an OAuth admin token via client-credentials and
// runs the idempotent server bootstrap once before any test executes. Mirrors
// what the integration suite does — the live stack starts un-bootstrapped.
func bootstrapForCompose() error {
	ctx := context.Background()
	cfg := clientcredentials.Config{
		ClientID:     os.Getenv("LAKEKEEPER_CLIENT_ID"),
		ClientSecret: os.Getenv("LAKEKEEPER_CLIENT_SECRET"),
		TokenURL:     os.Getenv("LAKEKEEPER_TOKEN_URL"),
		Scopes:       strings.Fields(os.Getenv("LAKEKEEPER_SCOPE")),
	}
	ts := cfg.TokenSource(ctx)
	if _, err := client.NewWithAuthSource(ctx,
		os.Getenv("LAKEKEEPER_BASE_URL"),
		&core.OAuthTokenSource{TokenSource: ts},
		client.WithInitialBootstrap(true, true, core.Ptr(managementv1.USERTYPE_APPLICATION)),
	); err != nil {
		return fmt.Errorf("create client: %w", err)
	}
	return nil
}

// freshKeycloakToken mints a fresh client-credentials access token. Tests
// that exercise --auth-mode token / --auth-mode k8s need the raw bearer
// string; we deliberately don't reuse sharedClient's token because its
// lifetime is opaque.
func freshKeycloakToken(t *testing.T) string {
	t.Helper()
	cfg := clientcredentials.Config{
		ClientID:     os.Getenv("LAKEKEEPER_CLIENT_ID"),
		ClientSecret: os.Getenv("LAKEKEEPER_CLIENT_SECRET"),
		TokenURL:     os.Getenv("LAKEKEEPER_TOKEN_URL"),
		Scopes:       strings.Fields(os.Getenv("LAKEKEEPER_SCOPE")),
	}
	tok, err := cfg.TokenSource(t.Context()).Token()
	if err != nil {
		t.Fatalf("mint keycloak token: %v", err)
	}
	return tok.AccessToken
}

// authFlagsOAuth2 returns the --auth-mode oauth2 flag set populated from the
// environment (.env). Use this as the base for any compose-mode lkctl call
// that needs admin credentials.
func authFlagsOAuth2() []string {
	return []string{
		"--base-url", os.Getenv("LAKEKEEPER_BASE_URL"),
		"--auth-mode", "oauth2",
		"--token-url", os.Getenv("LAKEKEEPER_TOKEN_URL"),
		"--client-id", os.Getenv("LAKEKEEPER_CLIENT_ID"),
		"--client-secret", os.Getenv("LAKEKEEPER_CLIENT_SECRET"),
		"--scopes", os.Getenv("LAKEKEEPER_SCOPE"),
	}
}

// authFlagsAccessToken returns --auth-mode token flags using a freshly minted
// admin bearer.
func authFlagsAccessToken(t *testing.T) []string {
	t.Helper()
	return []string{
		"--base-url", os.Getenv("LAKEKEEPER_BASE_URL"),
		"--auth-mode", "token",
		"--access-token", freshKeycloakToken(t),
	}
}

// authFlagsK8s returns --auth-mode k8s flags. lkctl reads the projected SA
// token at the default path (/var/run/secrets/kubernetes.io/serviceaccount/token)
// when no --k8s-token-path is supplied, so the runner pod's mount is enough.
// Compose backend never calls this — it has no SA token to present.
func authFlagsK8s() []string {
	return []string{
		"--base-url", os.Getenv("LAKEKEEPER_BASE_URL"),
		"--auth-mode", "k8s",
	}
}

// backendAdminFlags returns admin auth flags for the active backend:
// oauth2 client-credentials on compose (Keycloak is deployed there) and
// Kubernetes SA on kind (no Keycloak, projected SA token at the default
// path). Tests that just need an admin-authenticated lkctl invocation go
// through runOK/runFail and don't have to know which backend they're on.
func backendAdminFlags() []string {
	if activeBackend.Name() == BackendKind {
		return authFlagsK8s()
	}
	return authFlagsOAuth2()
}

// runOK executes lkctl with admin creds for the active backend (oauth2 on
// compose, k8s SA on kind) and fails the test on any non-zero exit. Returns
// combined stdout.
func runOK(t *testing.T, args ...string) []byte {
	t.Helper()
	full := append(backendAdminFlags(), args...)
	stdout, stderr, code, err := activeBackend.Exec(t.Context(), nil, full...)
	if err != nil {
		t.Fatalf("lkctl %v: spawn error: %v\nstderr: %s", redactArgs(args), err, stderr)
	}
	if code != 0 {
		t.Fatalf("lkctl %v: exit %d\nstdout: %s\nstderr: %s",
			redactArgs(args), code, stdout, stderr)
	}
	return stdout
}

// runFail executes lkctl with admin creds for the active backend (oauth2 on
// compose, k8s SA on kind) and fails the test if the exit code is zero.
// Returns (stdout, stderr, exitCode).
func runFail(t *testing.T, args ...string) ([]byte, []byte, int) {
	t.Helper()
	full := append(backendAdminFlags(), args...)
	stdout, stderr, code, err := activeBackend.Exec(t.Context(), nil, full...)
	if err != nil {
		t.Fatalf("lkctl %v: spawn error: %v", redactArgs(args), err)
	}
	if code == 0 {
		t.Fatalf("lkctl %v: expected non-zero exit, got 0\nstdout: %s\nstderr: %s",
			redactArgs(args), stdout, stderr)
	}
	return stdout, stderr, code
}

// runRaw executes lkctl with the given full arg list (no auth flags injected).
// Used by tests that supply their own auth flags (token, k8s, bad-creds).
func runRaw(t *testing.T, stdin []byte, args ...string) ([]byte, []byte, int) {
	t.Helper()
	stdout, stderr, code, err := activeBackend.Exec(t.Context(), stdin, args...)
	if err != nil {
		t.Fatalf("lkctl %v: spawn error: %v", redactArgs(args), err)
	}
	return stdout, stderr, code
}

// redactArgs scrubs values following sensitive flags so bearer tokens and
// client secrets don't leak into CI test logs on failure.
func redactArgs(args []string) []string {
	out := append([]string(nil), args...)
	for i := 0; i < len(out)-1; i++ {
		switch out[i] {
		case "--access-token", "--client-secret":
			out[i+1] = "<redacted>"
		}
	}
	return out
}

// decodeJSON parses lkctl JSON output into v, failing the test on error.
func decodeJSON(t *testing.T, data []byte, v any) {
	t.Helper()
	if err := json.Unmarshal(data, v); err != nil {
		t.Fatalf("decode lkctl json output: %v\nraw: %s", err, data)
	}
}

// randomName produces a short collision-resistant suffix for test resources.
func randomName(prefix string) string {
	return fmt.Sprintf("%s-%s", prefix, uuid.NewString()[:8])
}

// repoRootRelative resolves a path relative to the repo root from the
// caller's source file. Avoids depending on os.Getwd, which is brittle
// when go-test changes working directory.
func repoRootRelative(rel string) string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return rel
	}
	// e2e/cli/harness.go -> repo root
	return filepath.Join(filepath.Dir(filepath.Dir(filepath.Dir(file))), rel)
}

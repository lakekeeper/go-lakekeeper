//go:build e2e_cli

package clie2e

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
)

const (
	BackendCompose = "compose"
	BackendKind    = "kind"
)

// Backend abstracts how lkctl is invoked. Compose runs the binary on the host
// against the docker-compose stack; kind runs it inside a long-lived pod via
// `kubectl exec`, which gives the binary access to a real projected SA token.
//
// The CLI surface is identical between backends; tests gate kind-specific
// expectations with t.Skip when the active backend can't satisfy them.
type Backend interface {
	Name() string

	// Exec runs lkctl with the given args, optional stdin, and returns
	// (stdout, stderr, exitCode, err). err is non-nil only on spawn /
	// transport failures, never on non-zero exit — tests that expect a
	// non-zero exit must inspect the returned exitCode directly.
	Exec(ctx context.Context, stdin []byte, args ...string) (stdout, stderr []byte, exitCode int, err error)

	// Close releases any backend-owned resources (e.g. a temp build dir).
	Close()
}

// composeBackend invokes a host-built lkctl binary directly.
type composeBackend struct {
	binPath  string
	buildDir string
}

// newComposeBackend resolves the lkctl binary. Honours LKCTL_BIN if set
// (canonical when invoked from e2e/compose/run.sh, which builds once before
// driving the suite); otherwise builds cmd/lkctl into a temp dir.
func newComposeBackend() (Backend, error) {
	if bin := os.Getenv("LKCTL_BIN"); bin != "" {
		return &composeBackend{binPath: bin}, nil
	}

	dir, err := os.MkdirTemp("", "lkctl-e2e-")
	if err != nil {
		return nil, fmt.Errorf("temp dir: %w", err)
	}
	bin := filepath.Join(dir, "lkctl")
	out, err := exec.Command("go", "build", "-o", bin, repoRootRelative("cmd")).CombinedOutput()
	if err != nil {
		_ = os.RemoveAll(dir)
		return nil, fmt.Errorf("build lkctl: %w\n%s", err, out)
	}
	return &composeBackend{binPath: bin, buildDir: dir}, nil
}

func (b *composeBackend) Name() string { return BackendCompose }

func (b *composeBackend) Exec(ctx context.Context, stdin []byte, args ...string) ([]byte, []byte, int, error) {
	cmd := exec.CommandContext(ctx, b.binPath, args...)
	if stdin != nil {
		cmd.Stdin = bytes.NewReader(stdin)
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	code := 0
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			code = exitErr.ExitCode()
			err = nil
		}
	}
	return stdout.Bytes(), stderr.Bytes(), code, err
}

func (b *composeBackend) Close() {
	if b.buildDir != "" {
		_ = os.RemoveAll(b.buildDir)
	}
}

// kindBackend invokes lkctl inside a long-lived pod (lkctl-runner) that has
// the projected SA token mounted at the standard path. The pod is created by
// e2e/kind/run.sh before tests run and is torn down on teardown.
//
// We `kubectl exec` for every call rather than `kubectl run --rm` per call
// to keep latency reasonable across the suite.
type kindBackend struct {
	pod       string
	namespace string
	kubectl   string

	// closeOnce protects against repeat Close() calls.
	closeOnce sync.Once
}

func newKindBackend() (Backend, error) {
	pod := os.Getenv("LKCTL_E2E_POD")
	if pod == "" {
		pod = "lkctl-runner"
	}
	ns := os.Getenv("LKCTL_E2E_NAMESPACE")
	if ns == "" {
		ns = "default"
	}
	kc := os.Getenv("KUBECTL")
	if kc == "" {
		kc = "kubectl"
	}
	if _, err := exec.LookPath(kc); err != nil {
		return nil, fmt.Errorf("kubectl not found on PATH: %w", err)
	}
	return &kindBackend{pod: pod, namespace: ns, kubectl: kc}, nil
}

func (b *kindBackend) Name() string { return BackendKind }

func (b *kindBackend) Exec(ctx context.Context, stdin []byte, args ...string) ([]byte, []byte, int, error) {
	full := []string{"exec", "-n", b.namespace}
	if stdin != nil {
		full = append(full, "-i")
	}
	full = append(full, b.pod, "--", "lkctl")
	full = append(full, args...)
	cmd := exec.CommandContext(ctx, b.kubectl, full...)
	if stdin != nil {
		cmd.Stdin = bytes.NewReader(stdin)
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	code := 0
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			code = exitErr.ExitCode()
			err = nil
		}
	}
	return stdout.Bytes(), stderr.Bytes(), code, err
}

func (b *kindBackend) Close() {
	b.closeOnce.Do(func() {})
}

// requireBackend skips the test if the active backend is not one of the
// permitted modes. Use sparingly — most tests should run on every backend.
func requireBackend(tb testing.TB, modes ...string) {
	tb.Helper()
	for _, m := range modes {
		if activeBackend.Name() == m {
			return
		}
	}
	tb.Skipf("backend %s not in %v", activeBackend.Name(), modes)
}

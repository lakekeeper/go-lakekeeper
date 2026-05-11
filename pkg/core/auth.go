package core

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"golang.org/x/oauth2"
)

// DefaultK8sServiceAccountTokenPath is the standard projected-volume mount
// for a pod's service-account token. Used both as the SDK fallback when
// K8sServiceAccountAuthSource.ServiceAccountTokenPath is nil and as the
// lkctl --k8s-token-path default.
const DefaultK8sServiceAccountTokenPath = "/var/run/secrets/kubernetes.io/serviceaccount/token"

type (
	// AuthSource is used to obtain access tokens.
	AuthSource interface {
		// Init is called once before making any requests.
		// If the token source needs access to client to initialize itself, it should do so here.
		Init(context.Context) error

		// Header returns an authentication header. When no error is returned, the
		// key and value should never be empty.
		Header(context.Context) (key, value string, err error)

		// GetToken creates a token
		// mainly use to create the Catalog REST API
		GetToken(context.Context) (string, error)
	}

	// OAuthClientCredentialsAuthSource wraps a `golang.org/x/oauth2.TokenSource`
	// produced by the OAuth 2.0 client-credentials flow (typically via
	// `(*clientcredentials.Config).TokenSource(ctx)`). The wrapper delegates
	// caching and refresh-before-expiry to the underlying TokenSource — for
	// the standard `clientcredentials` source that is `oauth2.ReuseTokenSource`,
	// which is goroutine-safe and refreshes ~10 seconds before stated expiry
	// without any custom timer.
	//
	// The struct technically accepts any `oauth2.TokenSource`, so it also
	// works for other OAuth flows (auth-code, device, refresh-token-only). The
	// type name is deliberately specific to the client-credentials flow because
	// that is the only flow the SDK documents and tests; if you need another
	// flow, construct your own `oauth2.TokenSource` and pass it in here, or
	// implement `AuthSource` directly.
	OAuthClientCredentialsAuthSource struct {
		TokenSource oauth2.TokenSource
	}

	// AccessTokenAuthSource is an AuthSource that uses a static access token.
	// The token is added to the Authorization header using the Bearer scheme.
	// There is no expiry handling — if the token expires, requests start
	// returning 401. Use `OAuthClientCredentialsAuthSource` for long-running
	// processes.
	AccessTokenAuthSource struct {
		Token string
	}

	// K8sServiceAccountAuthSource is an AuthSource that loads the projected
	// Kubernetes service-account token from a file path at request time.
	//
	// The token is **re-read on every request** (file reads against tmpfs are
	// ~1µs and atomic by virtue of the kubelet's rename-on-rotate write
	// pattern), so a long-running pod will pick up the kubelet's hourly
	// rotations without restart. `Init` validates that the path is readable at
	// construction so misconfigurations surface eagerly.
	K8sServiceAccountAuthSource struct {
		// ServiceAccountTokenPath is the path to the service account token
		// file. Defaults to DefaultK8sServiceAccountTokenPath when nil.
		ServiceAccountTokenPath *string
	}
)

// check the implementations
var (
	_ AuthSource = (*OAuthClientCredentialsAuthSource)(nil)
	_ AuthSource = (*AccessTokenAuthSource)(nil)
	_ AuthSource = (*K8sServiceAccountAuthSource)(nil)
)

// Init validates that the configured TokenSource can actually obtain a token
// against the configured token endpoint, so a misconfigured TokenURL or set
// of credentials surfaces here at construction time rather than on the first
// API request.
func (as *OAuthClientCredentialsAuthSource) Init(_ context.Context) error {
	if as.TokenSource == nil {
		return errors.New("OAuthClientCredentialsAuthSource: TokenSource is nil")
	}
	if _, err := as.TokenSource.Token(); err != nil {
		return fmt.Errorf("OAuthClientCredentialsAuthSource: initial token fetch failed (check token URL and credentials): %w", err)
	}
	return nil
}

func (as *OAuthClientCredentialsAuthSource) Header(_ context.Context) (string, string, error) {
	t, err := as.TokenSource.Token()
	if err != nil {
		return "", "", err
	}

	return "Authorization", fmt.Sprintf("%s %s", t.TokenType, t.AccessToken), nil
}

func (as *OAuthClientCredentialsAuthSource) GetToken(_ context.Context) (string, error) {
	t, err := as.TokenSource.Token()
	if err != nil {
		return "", fmt.Errorf("failed to get token: %w", err)
	}
	return t.AccessToken, nil
}

// Init rejects an empty token at construction so the misconfiguration surfaces
// here rather than as an opaque 401 from the server later.
func (as *AccessTokenAuthSource) Init(context.Context) error {
	if as.Token == "" {
		return errors.New("AccessTokenAuthSource: Token is empty")
	}
	return nil
}

func (as *AccessTokenAuthSource) Header(context.Context) (string, string, error) {
	return "Authorization", "Bearer " + as.Token, nil
}

func (as *AccessTokenAuthSource) GetToken(context.Context) (string, error) {
	return as.Token, nil
}

// Init normalises the token path and validates that the file is readable
// right now. Subsequent Header / GetToken calls re-read the file each time
// so kubelet rotations are picked up without restart.
func (as *K8sServiceAccountAuthSource) Init(_ context.Context) error {
	if as.ServiceAccountTokenPath == nil {
		as.ServiceAccountTokenPath = Ptr(DefaultK8sServiceAccountTokenPath)
	}
	if _, err := as.readToken(); err != nil {
		return err
	}
	return nil
}

// readToken reads the configured service-account token file and returns its
// trimmed contents. Trims trailing whitespace / newline because those are a
// common kubelet artefact and would otherwise corrupt the Authorization
// header. Returns a clear error on missing-file, empty-file, or read-error.
func (as *K8sServiceAccountAuthSource) readToken() (string, error) {
	path := DefaultK8sServiceAccountTokenPath
	if as.ServiceAccountTokenPath != nil {
		path = *as.ServiceAccountTokenPath
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read service account token from %q: %w", path, err)
	}
	tok := strings.TrimRight(string(b), " \r\n\t")
	if tok == "" {
		return "", fmt.Errorf("service account token at %q is empty", path)
	}
	return tok, nil
}

func (as *K8sServiceAccountAuthSource) Header(context.Context) (header, value string, err error) {
	tok, err := as.readToken()
	if err != nil {
		return "", "", err
	}
	return "Authorization", "Bearer " + tok, nil
}

func (as *K8sServiceAccountAuthSource) GetToken(context.Context) (string, error) {
	return as.readToken()
}

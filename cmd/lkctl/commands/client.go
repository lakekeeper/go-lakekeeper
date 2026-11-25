package commands

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"

	"golang.org/x/oauth2/clientcredentials"

	managementv1 "github.com/lakekeeper/go-lakekeeper/pkg/apis/management/v1"
	"github.com/lakekeeper/go-lakekeeper/pkg/client"
	"github.com/lakekeeper/go-lakekeeper/pkg/core"
)

// Auth modes accepted by --auth-mode / LAKEKEEPER_AUTH_MODE.
const (
	authModeOAuth2 = "oauth2"
	authModeToken  = "token"
	authModeK8s    = "k8s"
)

// clientOptions captures the global flags that control how lkctl talks to the
// Lakekeeper server. Populated by the persistent flags on the root command.
type clientOptions struct {
	baseURL      string
	authMode     string
	tokenURL     string
	clientID     string
	clientSecret string
	scope        []string
	accessToken  string
	k8sTokenPath string
	bootstrap    bool
	debug        bool

	// stderr receives the supported-version warning; the root command sets
	// it to its own stderr.
	stderr io.Writer
	// versionChecked makes the supported-version check run once per run;
	// NewCommand sets it, and without it there is no check.
	versionChecked *sync.Once
}

// checkSupportedVersions warns on stderr, once per run, when the server is
// outside the versions this lkctl supports. It runs on the capabilities of
// a server info the command fetched anyway, so it adds no request, and it
// never fails the command.
func (o *clientOptions) checkSupportedVersions(caps client.Capabilities) {
	if o.versionChecked == nil {
		return
	}
	o.versionChecked.Do(func() {
		if caps.InSupportedVersions() || o.stderr == nil {
			return
		}
		fmt.Fprintf(o.stderr, "Warning: server is %s; this %s supports %s\n",
			caps.String(), cliName, caps.SupportedVersions())
	})
}

// validate returns an error if any of the required connection fields are
// missing. Required fields depend on the selected auth mode.
func (o *clientOptions) validate() error {
	if o.baseURL == "" {
		return errors.New("base url is required")
	}
	switch o.authMode {
	case authModeOAuth2:
		switch {
		case o.tokenURL == "":
			return errors.New("token url is required")
		case o.clientID == "":
			return errors.New("client id is required")
		case o.clientSecret == "":
			return errors.New("client secret is required")
		case len(o.scope) == 0:
			return errors.New("scope is required")
		}
	case authModeToken:
		if o.accessToken == "" {
			return errors.New("access token is required")
		}
	case authModeK8s:
		// k8sTokenPath always carries a value here: the cobra flag
		// defaults to core.DefaultK8sServiceAccountTokenPath. The
		// AuthSource's Init reads the file when the client is constructed.
	default:
		return fmt.Errorf("unknown auth mode %q", o.authMode)
	}
	return nil
}

// newClient validates opts, builds an AuthSource for the selected auth mode,
// and constructs an authenticated *client.Client. If the bootstrap flag is
// set, the server is bootstrapped during construction.
func newClient(ctx context.Context, opts *clientOptions) (*client.Client, error) {
	if err := opts.validate(); err != nil {
		return nil, err
	}

	authSource, err := buildAuthSource(ctx, opts)
	if err != nil {
		return nil, err
	}

	clientOpts := []client.Option{client.WithCapabilitiesObserver(opts.checkSupportedVersions)}
	if opts.bootstrap {
		clientOpts = append(clientOpts, client.WithInitialBootstrap(true, true, core.Ptr(managementv1.USERTYPE_APPLICATION)))
	}

	return client.NewWithAuthSource(ctx, opts.baseURL, authSource, clientOpts...)
}

func buildAuthSource(ctx context.Context, opts *clientOptions) (core.AuthSource, error) {
	switch opts.authMode {
	case authModeOAuth2:
		oauthCfg := clientcredentials.Config{
			ClientID:     opts.clientID,
			ClientSecret: opts.clientSecret,
			TokenURL:     opts.tokenURL,
			Scopes:       opts.scope,
		}
		tokenSource := oauthCfg.TokenSource(ctx)
		if _, err := tokenSource.Token(); err != nil {
			return nil, fmt.Errorf("oauth2 token: %w", err)
		}
		return &core.OAuthClientCredentialsAuthSource{TokenSource: tokenSource}, nil
	case authModeToken:
		return &core.AccessTokenAuthSource{Token: opts.accessToken}, nil
	case authModeK8s:
		return &core.K8sServiceAccountAuthSource{ServiceAccountTokenPath: &opts.k8sTokenPath}, nil
	default:
		return nil, fmt.Errorf("unknown auth mode %q", opts.authMode)
	}
}

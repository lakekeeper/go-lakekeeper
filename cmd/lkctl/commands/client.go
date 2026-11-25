package commands

import (
	"context"
	"errors"
	"fmt"
	"io"

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
	// client is the client newClient built for this run, for the
	// supported-version check.
	client *client.Client
}

// checkSupportedVersions warns on stderr when the server is outside the
// versions this lkctl supports. NewCommand runs it once per run, after the
// command. It reads only the capabilities of a server info the command
// fetched anyway, so it adds no request, and it never fails the command.
func (o *clientOptions) checkSupportedVersions() {
	if o.client == nil || o.stderr == nil {
		return
	}
	caps, ok := o.client.CachedCapabilities()
	if !ok || caps.InSupportedVersions() {
		return
	}
	fmt.Fprintf(o.stderr, "Warning: server is %s; this %s supports %s\n",
		caps, cliName, caps.SupportedVersions())
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
// set, it bootstraps the server.
func newClient(ctx context.Context, opts *clientOptions) (*client.Client, error) {
	if err := opts.validate(); err != nil {
		return nil, err
	}

	authSource, err := buildAuthSource(ctx, opts)
	if err != nil {
		return nil, err
	}

	c, err := client.NewWithAuthSource(ctx, opts.baseURL, authSource)
	if err != nil {
		return nil, err
	}
	opts.client = c
	if opts.bootstrap {
		if err := bootstrap(ctx, c); err != nil {
			return nil, err
		}
	}
	return c, nil
}

// bootstrap bootstraps the server as an operator application, unless it is
// bootstrapped already: what client.WithInitialBootstrap does, but after
// construction, so the supported-version check sees the server info even
// when bootstrapping fails.
func bootstrap(ctx context.Context, c *client.Client) error {
	info, err := c.Server.Info(ctx)
	if err != nil {
		return fmt.Errorf("bootstrap: %w", err)
	}
	if info.GetBootstrapped() {
		return nil
	}
	req := managementv1.NewBootstrapRequest(true)
	req.SetIsOperator(true)
	req.SetUserType(managementv1.USERTYPE_APPLICATION)
	return c.Server.Bootstrap(ctx, req)
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

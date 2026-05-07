package commands

import (
	"context"
	"errors"
	"fmt"

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
		// defaults to core.DefaultK8sServiceAccountTokenPath. File
		// existence is checked lazily by the AuthSource on first use.
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

	var clientOpts []client.Option
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
		return &core.OAuthTokenSource{TokenSource: tokenSource}, nil
	case authModeToken:
		return &core.AccessTokenAuthSource{Token: opts.accessToken}, nil
	case authModeK8s:
		return &core.K8sServiceAccountAuthSource{ServiceAccountTokenPath: &opts.k8sTokenPath}, nil
	default:
		return nil, fmt.Errorf("unknown auth mode %q", opts.authMode)
	}
}

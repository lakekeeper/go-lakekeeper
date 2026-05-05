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

// clientOptions captures the global flags that control how lkctl talks to the
// Lakekeeper server. Populated by the persistent flags on the root command.
type clientOptions struct {
	baseURL      string
	tokenURL     string
	clientID     string
	clientSecret string
	scope        []string
	bootstrap    bool
	debug        bool
}

// validate returns an error if any of the required connection fields are
// missing. Optional flags (bootstrap, debug) are not checked.
func (o *clientOptions) validate() error {
	switch {
	case o.baseURL == "":
		return errors.New("base url is required")
	case o.tokenURL == "":
		return errors.New("token url is required")
	case o.clientID == "":
		return errors.New("client id is required")
	case o.clientSecret == "":
		return errors.New("client secret is required")
	case len(o.scope) == 0:
		return errors.New("scope is required")
	}
	return nil
}

// newClient validates opts, exchanges OAuth2 client credentials for a token
// source, and constructs an authenticated *client.Client. If the bootstrap
// flag is set, the server is bootstrapped during construction.
func newClient(ctx context.Context, opts *clientOptions) (*client.Client, error) {
	if err := opts.validate(); err != nil {
		return nil, err
	}

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

	authSource := &core.OAuthTokenSource{TokenSource: tokenSource}

	var clientOpts []client.Option
	if opts.bootstrap {
		clientOpts = append(clientOpts, client.WithInitialBootstrap(true, true, core.Ptr(managementv1.USERTYPE_APPLICATION)))
	}

	return client.NewWithAuthSource(ctx, opts.baseURL, authSource, clientOpts...)
}

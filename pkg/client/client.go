package client

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/apache/iceberg-go/catalog/rest"
	"github.com/hashicorp/go-cleanhttp"
	"github.com/hashicorp/go-retryablehttp"

	managementv1 "github.com/lakekeeper/go-lakekeeper/pkg/apis/management/v1"
	"github.com/lakekeeper/go-lakekeeper/pkg/core"
	"github.com/lakekeeper/go-lakekeeper/pkg/version"
)

var defaultUserAgent = "go-lakekeeper/" + version.GetVersion().Version

// Client is a thin facade over the generated managementv1.APIClient.
//
// The generated APIClient is embedded, so each generated service handle
// (ProjectAPI, WarehouseAPI, ServerAPI, …) is reachable as a public field
// directly on Client. The facade adds: auth wiring (via core.AuthSource and a
// custom http.RoundTripper), optional retry behaviour, optional bootstrap on
// construction, and a CatalogV1 helper that delegates to apache/iceberg-go.
type Client struct {
	*managementv1.APIClient

	baseURL    *url.URL
	authSource core.AuthSource
}

// New constructs a Client authenticated with a static OAuth bearer token.
func New(ctx context.Context, baseURL, token string, options ...Option) (*Client, error) {
	return NewWithAuthSource(ctx, baseURL, &core.AccessTokenAuthSource{Token: token}, options...)
}

// NewWithAuthSource constructs a Client that obtains its credentials from the
// given AuthSource. The AuthSource's Init is invoked once during construction.
func NewWithAuthSource(ctx context.Context, baseURL string, as core.AuthSource, options ...Option) (*Client, error) {
	if as == nil {
		return nil, errors.New("auth source must not be nil")
	}

	parsed, err := url.Parse(strings.TrimRight(baseURL, "/"))
	if err != nil {
		return nil, fmt.Errorf("parse base url: %w", err)
	}
	if parsed.Host == "" {
		return nil, errors.New("base URL must include a host")
	}

	if err := as.Init(ctx); err != nil {
		return nil, fmt.Errorf("init auth source: %w", err)
	}

	settings := defaultSettings()
	for _, opt := range options {
		if opt == nil {
			continue
		}
		if err := opt(&settings); err != nil {
			return nil, err
		}
	}

	cfg := managementv1.NewConfiguration()
	cfg.UserAgent = defaultUserAgent
	if settings.userAgent != "" {
		cfg.UserAgent = settings.userAgent
	}
	cfg.Servers = managementv1.ServerConfigurations{{URL: parsed.String()}}
	cfg.HTTPClient = newHTTPClient(as, settings)

	c := &Client{
		APIClient:  managementv1.NewAPIClient(cfg),
		baseURL:    parsed,
		authSource: as,
	}

	if settings.bootstrap {
		if err := c.runBootstrap(ctx, settings); err != nil {
			return nil, fmt.Errorf("bootstrap: %w", err)
		}
	}

	return c, nil
}

// BaseURL returns a copy of the configured base URL.
func (c *Client) BaseURL() *url.URL {
	u := *c.baseURL
	return &u
}

// AuthSource returns the AuthSource used for authentication. Useful for
// callers who need to obtain a raw token (e.g., to drive the Iceberg catalog
// directly without going through CatalogV1).
func (c *Client) AuthSource() core.AuthSource {
	return c.authSource
}

// CatalogV1 returns an Iceberg REST catalog client scoped to a specific
// project + warehouse, using apache/iceberg-go for the protocol layer.
func (c *Client) CatalogV1(ctx context.Context, projectID, warehouse string, opts ...rest.Option) (*rest.Catalog, error) {
	if c.authSource != nil {
		token, err := c.authSource.GetToken(ctx)
		if err != nil {
			return nil, fmt.Errorf("get token: %w", err)
		}
		opts = append(opts, rest.WithOAuthToken(token))
	}
	opts = append(opts, rest.WithWarehouseLocation(fmt.Sprintf("%s/%s", projectID, warehouse)))

	catalogURL := *c.baseURL
	catalogURL.Path = strings.TrimRight(c.baseURL.Path, "/") + "/catalog"

	return rest.NewCatalog(ctx, "rest", catalogURL.String(), opts...)
}

func (c *Client) runBootstrap(ctx context.Context, s settings) error {
	info, _, err := c.ServerAPI.GetServerInfo(ctx).Execute()
	if err != nil {
		return fmt.Errorf("server info: %w", err)
	}
	if info != nil && info.Bootstrapped {
		return nil
	}

	req := managementv1.NewBootstrapRequest(true)
	isOp := s.bootstrapAsOperator
	req.IsOperator = &isOp
	if s.bootstrapUserType != nil {
		req.SetUserType(*s.bootstrapUserType)
	}

	if _, err := c.ServerAPI.Bootstrap(ctx).BootstrapRequest(*req).Execute(); err != nil {
		return err
	}
	return nil
}

func newHTTPClient(as core.AuthSource, s settings) *http.Client {
	transport := &core.AuthRoundTripper{
		Base:       cleanhttp.DefaultPooledTransport(),
		AuthSource: as,
	}
	base := &http.Client{Transport: transport}
	if s.disableRetries {
		return base
	}

	retry := retryablehttp.NewClient()
	retry.HTTPClient = base
	retry.Logger = nil
	if s.retryMax > 0 {
		retry.RetryMax = s.retryMax
	}
	if s.retryWaitMin > 0 {
		retry.RetryWaitMin = s.retryWaitMin
	}
	if s.retryWaitMax > 0 {
		retry.RetryWaitMax = s.retryWaitMax
	}
	if s.checkRetry != nil {
		retry.CheckRetry = s.checkRetry
	}
	if s.backoff != nil {
		retry.Backoff = s.backoff
	}
	if s.errorHandler != nil {
		retry.ErrorHandler = s.errorHandler
	}

	return retry.StandardClient()
}

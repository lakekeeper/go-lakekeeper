package client

import (
	"errors"
	"net/http"
	"time"

	"github.com/hashicorp/go-retryablehttp"

	managementv1 "github.com/lakekeeper/go-lakekeeper/pkg/apis/management/v1"
)

// Option customises a Client at construction time. Options apply in order,
// so when the same option is given more than once, the last one wins. All
// options take effect before WithInitialBootstrap sends its requests.
type Option func(*settings) error

// settings holds the values an Option may set. Kept unexported so that the
// surface stays append-only — new options don't widen the public API.
type settings struct {
	userAgent string

	// httpClient, if set, is a copy of the client given to WithHTTPClient,
	// with a nil Transport resolved to http.DefaultTransport.
	httpClient *http.Client

	// Retry behaviour, applied to the http.Client used by the generated
	// APIClient. Zero values mean "use retryablehttp defaults".
	disableRetries bool
	retryMax       int
	retryWaitMin   time.Duration
	retryWaitMax   time.Duration
	checkRetry     retryablehttp.CheckRetry
	backoff        retryablehttp.Backoff
	errorHandler   retryablehttp.ErrorHandler

	// Bootstrap behaviour, applied once at construction time.
	bootstrap           bool
	bootstrapAsOperator bool
	bootstrapUserType   *managementv1.UserType
}

func defaultSettings() settings {
	return settings{}
}

// WithUserAgent overrides the default User-Agent header.
func WithUserAgent(userAgent string) Option {
	return func(s *settings) error {
		s.userAgent = userAgent
		return nil
	}
}

// WithHTTPClient supplies the HTTP client whose settings the SDK builds on.
//
// The client's Transport becomes the base transport under the SDK's auth
// layer (and, unless WithoutRetries is given, the retry layer), so every
// request is still authenticated and retried. A nil Transport falls back to
// http.DefaultTransport, as net/http does. The other fields (Timeout,
// CheckRedirect, Jar) carry over. Use it for a custom TLS configuration
// (private CA, InsecureSkipVerify), a proxy, or timeouts.
//
// The supplied client is copied, never mutated. It also applies to the
// requests sent by WithInitialBootstrap, and CatalogV1 uses its transport
// (see CatalogV1). A nil client fails construction.
func WithHTTPClient(httpClient *http.Client) Option {
	return func(s *settings) error {
		if httpClient == nil {
			return errors.New("http client must not be nil")
		}
		copied := *httpClient
		if copied.Transport == nil {
			copied.Transport = http.DefaultTransport
		}
		s.httpClient = &copied
		return nil
	}
}

// WithoutRetries disables the retry layer entirely.
func WithoutRetries() Option {
	return func(s *settings) error {
		s.disableRetries = true
		return nil
	}
}

// WithRetryMax overrides the maximum number of retries.
func WithRetryMax(n int) Option {
	return func(s *settings) error {
		s.retryMax = n
		return nil
	}
}

// WithRetryWait overrides the minimum and maximum wait between retries.
func WithRetryWait(minWait, maxWait time.Duration) Option {
	return func(s *settings) error {
		s.retryWaitMin = minWait
		s.retryWaitMax = maxWait
		return nil
	}
}

// WithCheckRetry overrides retryablehttp's CheckRetry callback.
func WithCheckRetry(check retryablehttp.CheckRetry) Option {
	return func(s *settings) error {
		s.checkRetry = check
		return nil
	}
}

// WithBackoff overrides retryablehttp's Backoff callback.
func WithBackoff(backoff retryablehttp.Backoff) Option {
	return func(s *settings) error {
		s.backoff = backoff
		return nil
	}
}

// WithErrorHandler overrides retryablehttp's ErrorHandler.
func WithErrorHandler(handler retryablehttp.ErrorHandler) Option {
	return func(s *settings) error {
		s.errorHandler = handler
		return nil
	}
}

// WithInitialBootstrap arranges for the server to be bootstrapped at client
// construction time if it has not been bootstrapped yet.
//
// acceptTermsOfUse must be true for the bootstrap to proceed; if false, the
// option is a no-op.
//
// isOperator controls whether the bootstrapping user receives the operator
// role.
//
// userType is optional. If nil, the server falls back to the type encoded in
// the auth token.
func WithInitialBootstrap(acceptTermsOfUse, isOperator bool, userType *managementv1.UserType) Option {
	return func(s *settings) error {
		if !acceptTermsOfUse {
			return nil
		}
		s.bootstrap = true
		s.bootstrapAsOperator = isOperator
		s.bootstrapUserType = userType
		return nil
	}
}

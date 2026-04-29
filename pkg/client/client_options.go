package client

import (
	"time"

	"github.com/hashicorp/go-retryablehttp"

	managementv1 "github.com/lakekeeper/go-lakekeeper/pkg/apis/management/v1"
)

// Option customises a Client at construction time.
type Option func(*settings) error

// settings holds the values an Option may set. Kept unexported so that the
// surface stays append-only — new options don't widen the public API.
type settings struct {
	userAgent string

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

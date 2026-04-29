package core

import (
	"fmt"
	"net/http"
)

// AuthRoundTripper injects the auth header from an AuthSource into every
// outgoing request. Pre-existing values for the same header are not
// overwritten, so callers can override per-request if needed.
type AuthRoundTripper struct {
	Base       http.RoundTripper
	AuthSource AuthSource
}

func (t *AuthRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	base := t.Base
	if base == nil {
		base = http.DefaultTransport
	}

	if t.AuthSource == nil {
		return base.RoundTrip(req)
	}

	key, value, err := t.AuthSource.Header(req.Context())
	if err != nil {
		return nil, fmt.Errorf("auth source: %w", err)
	}

	if req.Header.Get(key) == "" {
		clone := req.Clone(req.Context())
		clone.Header.Set(key, value)
		req = clone
	}

	return base.RoundTrip(req)
}

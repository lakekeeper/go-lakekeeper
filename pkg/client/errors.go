package client

import (
	"errors"
	"fmt"
	"strings"

	managementv1 "github.com/lakekeeper/go-lakekeeper/pkg/apis/management/v1"
)

const apiErrorBodyLimit = 1024

// wrapAPIError annotates err with prefix and, when err carries a server
// response body via *managementv1.GenericOpenAPIError, appends the body
// (truncated at apiErrorBodyLimit) so callers see the server's actual
// complaint instead of a bare status line or "undefined response type".
// Falls back to fmt.Errorf("%s: %w", prefix, err) on any other error type
// (and returns nil for nil err).
func wrapAPIError(prefix string, err error) error {
	if err == nil {
		return nil
	}
	var apiErr *managementv1.GenericOpenAPIError
	if errors.As(err, &apiErr) {
		if body := apiErr.Body(); len(body) > 0 {
			s := strings.TrimSpace(string(body))
			if len(s) > apiErrorBodyLimit {
				s = s[:apiErrorBodyLimit] + "...(truncated)"
			}
			return fmt.Errorf("%s: %s: %s", prefix, apiErr.Error(), s)
		}
	}
	return fmt.Errorf("%s: %w", prefix, err)
}

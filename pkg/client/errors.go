package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	managementv1 "github.com/lakekeeper/go-lakekeeper/pkg/apis/management/v1"
)

// apiErrorBodyLimit caps how much of the response body APIError.Error
// includes. Generous enough for Lakekeeper's JSON error payloads, small
// enough to keep an accidental HTML page from flooding a terminal.
const apiErrorBodyLimit = 1024

// ErrEmptyResponse is returned by a façade method that returns an entity when
// the server answers with a success status but no body. A nil error from such
// a method therefore always comes with a non-nil result.
var ErrEmptyResponse = errors.New("empty response body")

// APIError is the error every façade returns when the server answers with
// an error status. Callers tell failures apart by StatusCode:
//
//	var apiErr *client.APIError
//	if errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound {
//		// ...
//	}
//
// It unwraps to the generated *managementv1.GenericOpenAPIError, so
// errors.As reaches that type too. Build one with WrapAPIError.
type APIError struct {
	// StatusCode is the HTTP status code, or 0 when unknown (WrapAPIError
	// was given no response).
	StatusCode int
	// Body is the full raw response body; empty when the server sent none.
	// Error includes it, truncated.
	Body []byte
	// Model is the decoded Lakekeeper error model (code, message, type),
	// or nil when Body does not parse as one.
	Model *managementv1.ErrorModel

	op     string
	status string
	err    error
}

// Error reports "<op>: <status>: <body>". The body is truncated, and
// omitted when empty.
func (e *APIError) Error() string {
	msg := e.op + ": " + e.status
	if s := strings.TrimSpace(string(e.Body)); s != "" {
		if len(s) > apiErrorBodyLimit {
			s = s[:apiErrorBodyLimit] + "...(truncated)"
		}
		msg += ": " + s
	}
	return msg
}

// Unwrap returns the underlying generated error.
func (e *APIError) Unwrap() error { return e.err }

// WrapAPIError is the single constructor for façade errors. It is exported
// so callers of the generated client (the escape hatch embedded in Client)
// get the same error the façades return:
//
//	out, resp, err := c.TasksAPI.ListTasks(ctx, warehouseID).Execute()
//	if err != nil {
//		return client.WrapAPIError("list tasks", err, resp)
//	}
//
// op names the operation and prefixes the message. resp may be nil; the
// status is then unknown (StatusCode 0). WrapAPIError returns nil for a nil
// err, an *APIError when err carries a generated
// *managementv1.GenericOpenAPIError for an error status, and otherwise err
// wrapped with %w (transport failure, context cancellation, or a 2xx body
// that failed to decode), so the chain stays intact.
func WrapAPIError(op string, err error, resp *http.Response) error {
	if err == nil {
		return nil
	}
	var genErr *managementv1.GenericOpenAPIError
	if errors.As(err, &genErr) {
		if resp != nil && resp.StatusCode < http.StatusMultipleChoices {
			// The server answered with success; it is the response body
			// that failed to decode. That is not an error status, so it is
			// not an APIError, and the decode message is the cause to show.
			return fmt.Errorf("%s: decode %s response: %w", op, resp.Status, err)
		}
		apiErr := &APIError{op: op, err: err, Body: genErr.Body(), status: genErr.Error()}
		var decoded managementv1.IcebergErrorResponse
		if json.Unmarshal(apiErr.Body, &decoded) == nil {
			apiErr.Model = &decoded.Error
		}
		if resp != nil {
			apiErr.StatusCode = resp.StatusCode
			apiErr.status = resp.Status
			// The generated message starts with the status line, unless
			// decoding the error body failed (a 4xx with an HTML page, for
			// example). Keep that decode error instead of dropping it.
			if msg := genErr.Error(); !strings.HasPrefix(msg, resp.Status) {
				apiErr.status += " (" + msg + ")"
			}
		}
		return apiErr
	}
	return fmt.Errorf("%s: %w", op, err)
}

// entityResult finishes a façade method that returns an entity: err goes
// through WrapAPIError, and a nil result without an error (the generated
// client's decoding of an empty 2xx body) becomes ErrEmptyResponse.
func entityResult[T any](op string, out *T, resp *http.Response, err error) (*T, error) {
	if err != nil {
		return out, WrapAPIError(op, err, resp)
	}
	if out == nil {
		return nil, fmt.Errorf("%s: %w", op, ErrEmptyResponse)
	}
	return out, nil
}

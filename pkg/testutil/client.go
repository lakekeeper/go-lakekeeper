// Package testutil is the httptest harness for façade and command tests: a
// test server, a *client.Client pointed at it, and a recorder for the
// requests the server receives.
//
// Façade tests must live in package client_test (pkg/client importing this
// package from an internal test would be an import cycle).
package testutil

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"

	"github.com/lakekeeper/go-lakekeeper/pkg/client"
)

// NewServer starts an httptest server serving the returned mux and returns
// the server's base URL. The server is closed when the test ends. Use it
// directly when the code under test builds its own client, as lkctl commands
// do (pass the URL as --base-url).
func NewServer(t *testing.T) (*http.ServeMux, string) {
	t.Helper()
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return mux, server.URL
}

// ServerMux starts a test server like NewServer and returns a *client.Client
// talking to it, with a static token and retries disabled. Register handlers
// on the mux (Respond, or mux.HandleFunc for custom behaviour) before calling
// the façade under test.
func ServerMux(t *testing.T) (*http.ServeMux, *client.Client) {
	t.Helper()
	mux, baseURL := NewServer(t)
	c, err := client.New(t.Context(), baseURL, "test-token", client.WithoutRetries())
	if err != nil {
		t.Fatal(err)
	}
	return mux, c
}

// CapturedRequest is a copy of a request the test server received, safe to
// read after the call under test has returned.
type CapturedRequest struct {
	Method string
	Path   string // URL path, without the query
	Query  url.Values
	Header http.Header
	Body   []byte
}

// RequestLog records the requests a handler registered with Respond receives.
// It is safe for concurrent use.
type RequestLog struct {
	mu   sync.Mutex
	reqs []CapturedRequest
}

// Len returns how many requests were received.
func (l *RequestLog) Len() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.reqs)
}

// Last returns the most recent request. It fails the test immediately if no
// request was received.
func (l *RequestLog) Last(t *testing.T) CapturedRequest {
	t.Helper()
	l.mu.Lock()
	defer l.mu.Unlock()
	if len(l.reqs) == 0 {
		t.Fatal("test server received no request")
	}
	return l.reqs[len(l.reqs)-1]
}

func (l *RequestLog) add(r *http.Request) error {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return err
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.reqs = append(l.reqs, CapturedRequest{
		Method: r.Method,
		Path:   r.URL.Path,
		Query:  r.URL.Query(),
		Header: r.Header.Clone(),
		Body:   body,
	})
	return nil
}

// Respond registers pattern (a ServeMux pattern such as
// "GET /management/v1/warehouse/{id}") on mux. Every matching request is
// recorded in the returned log and answered with status and body:
//
//   - nil: no body (for 204 responses or bodiless errors);
//   - string or []byte: written as is, with Content-Type application/json
//     (use it for literal JSON payloads and Lakekeeper error bodies);
//   - anything else: encoded as JSON.
func Respond(t *testing.T, mux *http.ServeMux, pattern string, status int, body any) *RequestLog {
	t.Helper()
	log := &RequestLog{}
	mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
		if err := log.add(r); err != nil {
			t.Errorf("read request body: %v", err)
		}
		if body == nil {
			w.WriteHeader(status)
			return
		}
		var payload []byte
		switch b := body.(type) {
		case string:
			payload = []byte(b)
		case []byte:
			payload = b
		default:
			var err error
			if payload, err = json.Marshal(b); err != nil {
				t.Errorf("encode response: %v", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		if _, err := w.Write(payload); err != nil {
			t.Errorf("write response: %v", err)
		}
	})
	return log
}

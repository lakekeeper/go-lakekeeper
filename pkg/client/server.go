package client

import (
	"context"
	"errors"
	"net/http"

	managementv1 "github.com/lakekeeper/go-lakekeeper/pkg/apis/management/v1"
)

// Server is a one-call façade over ServerAPIService.
type Server struct {
	api *managementv1.ServerAPIService
	// caps is the Client's capability cache, which every server info
	// fetch fills.
	caps *capabilityCache
}

// Info returns the server's bootstrap state and version metadata. It also
// fills the Client's capabilities (see Client.Capabilities), so a later
// Capabilities call costs no request.
func (s *Server) Info(ctx context.Context) (*managementv1.ServerInfo, error) {
	info, caps, err := s.fetchInfo(ctx)
	if caps != nil && s.caps != nil {
		s.caps.record(caps)
	}
	return info, err
}

// fetchInfo gets the server info and the capabilities read from it. The
// capabilities are read from the raw body, so they are there even when the
// generated model rejects the info, as it does for an older server that
// lacks a field the spec requires; caps is nil only when the body carries no
// version.
func (s *Server) fetchInfo(ctx context.Context) (*managementv1.ServerInfo, *Capabilities, error) {
	out, resp, err := s.api.GetServerInfo(ctx).Execute()
	info, wrapped := entityResult("get server info", out, resp, err)
	if wrapped == nil {
		return info, capabilitiesFromServerInfo(info), nil
	}
	var genErr *managementv1.GenericOpenAPIError
	if resp != nil && resp.StatusCode < http.StatusMultipleChoices && errors.As(err, &genErr) {
		return info, capabilitiesFromBody(genErr.Body()), wrapped
	}
	return info, nil, wrapped
}

// Bootstrap performs initial server setup with the calling principal.
// An already bootstrapped server rejects it with a
// 400 *APIError; check Info's bootstrapped flag first, or use
// WithInitialBootstrap, which does that at construction time.
func (s *Server) Bootstrap(ctx context.Context, req *managementv1.BootstrapRequest) error {
	const op = "bootstrap"
	if req == nil {
		return errors.New(op + ": request must not be nil")
	}
	resp, err := s.api.Bootstrap(ctx).BootstrapRequest(*req).Execute()
	return WrapAPIError(op, err, resp)
}

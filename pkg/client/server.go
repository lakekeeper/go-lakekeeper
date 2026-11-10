package client

import (
	"context"
	"errors"

	managementv1 "github.com/lakekeeper/go-lakekeeper/pkg/apis/management/v1"
)

// Server is a one-call façade over ServerAPIService.
type Server struct {
	api *managementv1.ServerAPIService
}

// Info returns the server's bootstrap state and version metadata.
func (s *Server) Info(ctx context.Context) (*managementv1.ServerInfo, error) {
	out, resp, err := s.api.GetServerInfo(ctx).Execute()
	return entityResult("get server info", out, resp, err)
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

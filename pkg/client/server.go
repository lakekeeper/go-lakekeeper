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
	out, _, err := s.api.GetServerInfo(ctx).Execute()
	return out, err
}

// Bootstrap performs initial server setup. Idempotent against an
// already-bootstrapped server (the server returns OK with no effect).
// Construction-time bootstrap is also available via WithInitialBootstrap.
func (s *Server) Bootstrap(ctx context.Context, req *managementv1.BootstrapRequest) error {
	if req == nil {
		return errors.New("bootstrap: request must not be nil")
	}
	_, err := s.api.Bootstrap(ctx).BootstrapRequest(*req).Execute()
	return err
}

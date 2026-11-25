package client

import (
	"context"

	managementv1 "github.com/lakekeeper/go-lakekeeper/pkg/apis/management/v1"
)

// cedarRequirement is what every Cedar operation needs: Lakekeeper+ running
// the Cedar authorization backend.
var cedarRequirement = requirement{
	feature:        "Cedar",
	lakekeeperPlus: true,
	authzBackend:   AuthzBackendCedar,
}

// Cedar is a one-call façade over PermissionsCedarAPIService: Cedar policy
// management on Lakekeeper+ with the Cedar authorization backend. On other
// servers its methods fail with an *UnsupportedError (see ErrUnsupported).
type Cedar struct {
	api  *managementv1.PermissionsCedarAPIService
	caps *capabilityCache
}

// Schema returns the active Cedar schema, as fragments keyed by file path.
func (c *Cedar) Schema(ctx context.Context) (*managementv1.GetCedarSchemaResponse, error) {
	const op = "get cedar schema"
	out, resp, err := c.api.GetCedarSchema(ctx).Execute()
	out, err = entityResult(op, out, resp, err)
	return out, c.caps.diagnose(ctx, cedarRequirement, op, err)
}

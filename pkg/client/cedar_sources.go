package client

import (
	"context"
	"errors"

	managementv1 "github.com/lakekeeper/go-lakekeeper/pkg/apis/management/v1"
)

// PolicySources lists the sources the server's Cedar policies come from
// (files, Kubernetes ConfigMaps), each with its policy count. The
// predefined-policies tier is not a configured source, so it is not listed;
// ListServerPolicies without a source filter includes its policies.
func (c *Cedar) PolicySources(ctx context.Context) (*managementv1.ListCedarPolicySourcesResponse, error) {
	const op = "list cedar policy sources"
	out, resp, err := c.api.ListCedarPolicySources(ctx).Execute()
	out, err = entityResult(op, out, resp, err)
	return out, c.caps.diagnose(ctx, cedarRequirement, op, err)
}

// EntitySources lists the sources the server's additional Cedar entities
// come from (files, Kubernetes ConfigMaps), each with its entity count.
func (c *Cedar) EntitySources(ctx context.Context) (*managementv1.ListCedarEntitySourcesResponse, error) {
	const op = "list cedar entity sources"
	out, resp, err := c.api.ListCedarEntitySources(ctx).Execute()
	out, err = entityResult(op, out, resp, err)
	return out, c.caps.diagnose(ctx, cedarRequirement, op, err)
}

// ListServerPolicies returns one page of the server's active Cedar policies
// that match query, as Cedar's JSON policy format. A nil query sends an
// empty one: every policy, the predefined-policies tier included. Page with
// the response's GetNextPageToken and the query's PageToken. Policies stored
// for projects and warehouses are not included; list them with
// ListProjectPolicies and ListWarehousePolicies.
func (c *Cedar) ListServerPolicies(ctx context.Context, query *managementv1.ListCedarPoliciesQuery) (*managementv1.GetCedarPoliciesResponse, error) {
	const op = "list server cedar policies"
	if query == nil {
		query = &managementv1.ListCedarPoliciesQuery{}
	}
	out, resp, err := c.api.ListCedarPolicies(ctx).ListCedarPoliciesQuery(*query).Execute()
	out, err = entityResult(op, out, resp, err)
	return out, c.caps.diagnose(ctx, cedarRequirement, op, err)
}

// ResolveEntities returns the Cedar entities the server derives for an
// authorization decision on req's resource, for req's principal or, without
// one, for the caller.
//
// The spec gives this operation no x-project-id header, so it takes no
// WithProject. A project resource without a project id therefore leaves the
// choice to the server; name the project in the request to be sure.
func (c *Cedar) ResolveEntities(ctx context.Context, req *managementv1.ResolveCedarEntitiesRequest) (*managementv1.ResolveCedarEntitiesResponse, error) {
	const op = "resolve cedar entities"
	if req == nil {
		return nil, errors.New(op + ": request must not be nil")
	}
	out, resp, err := c.api.ResolveCedarEntities(ctx).ResolveCedarEntitiesRequest(*req).Execute()
	out, err = entityResult(op, out, resp, err)
	return out, c.caps.diagnose(ctx, cedarRequirement, op, err)
}

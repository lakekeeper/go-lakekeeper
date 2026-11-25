package client

import (
	"context"
	"errors"

	managementv1 "github.com/lakekeeper/go-lakekeeper/pkg/apis/management/v1"
)

// PredefinedPoliciesOptions are the query parameters of listing and setting
// predefined Cedar policies, at project and at warehouse scope. A nil
// *PredefinedPoliciesOptions or a zero field sends nothing for that
// parameter.
type PredefinedPoliciesOptions struct {
	// IncludePolicy asks for each policy's rule, as Cedar's JSON policy
	// format, in the listing.
	IncludePolicy bool
}

// policyIncludable is any generated request builder that takes the
// include-policy query parameter.
type policyIncludable[R any] interface {
	IncludePolicy(includePolicy bool) R
}

func withPredefinedPoliciesOptions[R policyIncludable[R]](req R, opts *PredefinedPoliciesOptions) R {
	if opts != nil && opts.IncludePolicy {
		return req.IncludePolicy(true)
	}
	return req
}

func requireChanges(op string, changes []managementv1.PredefinedPolicyChange) error {
	if len(changes) == 0 {
		return errors.New(op + ": at least one change is required")
	}
	return nil
}

// ListProjectPredefinedPolicies lists the predefined Cedar policies with
// their enablement for a project. Name the Project with WithProject, or omit
// it for the server's default project. Accepts WithBreakGlass: each
// can-toggle then says whether emergency access could switch that policy.
func (c *Cedar) ListProjectPredefinedPolicies(ctx context.Context, opts *PredefinedPoliciesOptions, callOpts ...CallOption) (*managementv1.PredefinedPoliciesResponse, error) {
	const op = "list project predefined cedar policies"
	req, err := withProjectAndBreakGlass(op, c.api.ListProjectPredefinedPolicies(ctx), callOpts)
	if err != nil {
		return nil, err
	}
	out, resp, err := withPredefinedPoliciesOptions(req, opts).Execute()
	out, err = entityResult(op, out, resp, err)
	return out, c.caps.diagnose(ctx, cedarRequirement, op, err)
}

// SetProjectPredefinedPolicies enables or disables predefined Cedar policies
// for a project and returns the project's listing as of the write. changes
// must not be empty. Accepts WithProject and WithBreakGlass.
func (c *Cedar) SetProjectPredefinedPolicies(ctx context.Context, changes []managementv1.PredefinedPolicyChange, opts *PredefinedPoliciesOptions, callOpts ...CallOption) (*managementv1.PredefinedPoliciesResponse, error) {
	const op = "set project predefined cedar policies"
	if err := requireChanges(op, changes); err != nil {
		return nil, err
	}
	req, err := withProjectAndBreakGlass(op, c.api.SetProjectPredefinedPolicies(ctx), callOpts)
	if err != nil {
		return nil, err
	}
	req = req.SetPredefinedPoliciesRequest(*managementv1.NewSetPredefinedPoliciesRequest(changes))
	out, resp, err := withPredefinedPoliciesOptions(req, opts).Execute()
	out, err = entityResult(op, out, resp, err)
	return out, c.caps.diagnose(ctx, cedarRequirement, op, err)
}

// ResetProjectPredefinedPolicies drops every decision a project has recorded
// about predefined Cedar policies, putting it back on the provided defaults.
// Accepts WithProject.
func (c *Cedar) ResetProjectPredefinedPolicies(ctx context.Context, callOpts ...CallOption) error {
	const op = "reset project predefined cedar policies"
	req, err := withProject(op, c.api.ResetProjectPredefinedPolicies(ctx), callOpts)
	if err != nil {
		return err
	}
	resp, err := req.Execute()
	return c.caps.diagnose(ctx, cedarRequirement, op, WrapAPIError(op, err, resp))
}

// ListWarehousePredefinedPolicies lists the predefined Cedar policies a
// warehouse decides, with their enablement for it. Accepts WithBreakGlass.
func (c *Cedar) ListWarehousePredefinedPolicies(ctx context.Context, warehouseID string, opts *PredefinedPoliciesOptions, callOpts ...CallOption) (*managementv1.PredefinedPoliciesResponse, error) {
	const op = "list warehouse predefined cedar policies"
	req, err := withBreakGlass(op, c.api.ListWarehousePredefinedPolicies(ctx, warehouseID), callOpts)
	if err != nil {
		return nil, err
	}
	out, resp, err := withPredefinedPoliciesOptions(req, opts).Execute()
	out, err = entityResult(op, out, resp, err)
	return out, c.caps.diagnose(ctx, cedarRequirement, op, err)
}

// SetWarehousePredefinedPolicies enables or disables predefined Cedar
// policies for a warehouse and returns the warehouse's listing as of the
// write. changes must not be empty. Accepts WithBreakGlass.
func (c *Cedar) SetWarehousePredefinedPolicies(ctx context.Context, warehouseID string, changes []managementv1.PredefinedPolicyChange, opts *PredefinedPoliciesOptions, callOpts ...CallOption) (*managementv1.PredefinedPoliciesResponse, error) {
	const op = "set warehouse predefined cedar policies"
	if err := requireChanges(op, changes); err != nil {
		return nil, err
	}
	req, err := withBreakGlass(op, c.api.SetWarehousePredefinedPolicies(ctx, warehouseID), callOpts)
	if err != nil {
		return nil, err
	}
	req = req.SetPredefinedPoliciesRequest(*managementv1.NewSetPredefinedPoliciesRequest(changes))
	out, resp, err := withPredefinedPoliciesOptions(req, opts).Execute()
	out, err = entityResult(op, out, resp, err)
	return out, c.caps.diagnose(ctx, cedarRequirement, op, err)
}

// ResetWarehousePredefinedPolicies drops every decision a warehouse has
// recorded about predefined Cedar policies, so it follows its project again.
func (c *Cedar) ResetWarehousePredefinedPolicies(ctx context.Context, warehouseID string) error {
	const op = "reset warehouse predefined cedar policies"
	resp, err := c.api.ResetWarehousePredefinedPolicies(ctx, warehouseID).Execute()
	return c.caps.diagnose(ctx, cedarRequirement, op, WrapAPIError(op, err, resp))
}

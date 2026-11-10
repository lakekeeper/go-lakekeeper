package client

import (
	"context"
	"errors"

	managementv1 "github.com/lakekeeper/go-lakekeeper/pkg/apis/management/v1"
)

// Projects is a one-call façade over ProjectAPIService. Get and Rename act
// on the project named by WithProject (sent as x-project-id) and, without
// it, on the server's default project. Delete requires WithProject. Create
// and List are not project-scoped. For endpoints without a façade, reach for
// c.ProjectAPI.
type Projects struct {
	api *managementv1.ProjectAPIService
}

// Create creates a new project and returns the resulting metadata.
func (p *Projects) Create(ctx context.Context, req *managementv1.CreateProjectRequest) (*managementv1.CreateProjectResponse, error) {
	const op = "create project"
	if req == nil {
		return nil, errors.New(op + ": request must not be nil")
	}
	out, resp, err := p.api.CreateProject(ctx).CreateProjectRequest(*req).Execute()
	return entityResult(op, out, resp, err)
}

// Get fetches the project named by WithProject, or the server's default
// project when none is named.
func (p *Projects) Get(ctx context.Context, callOpts ...CallOption) (*managementv1.GetProjectResponse, error) {
	const op = "get project"
	req, err := withProject(op, p.api.GetProject(ctx), callOpts)
	if err != nil {
		return nil, err
	}
	out, resp, err := req.Execute()
	return entityResult(op, out, resp, err)
}

// Delete removes the project named by WithProject. Unlike the other
// project-scoped methods it has no default-project fallback: without
// WithProject it returns ErrProjectRequired and sends nothing, so it can't
// delete the server's default project by omission. To delete the default
// project, name it explicitly.
func (p *Projects) Delete(ctx context.Context, callOpts ...CallOption) error {
	const op = "delete project"
	req, err := withRequiredProject(op, p.api.DeleteProject(ctx), callOpts)
	if err != nil {
		return err
	}
	resp, err := req.Execute()
	return WrapAPIError(op, err, resp)
}

// List returns all projects visible to the caller.
func (p *Projects) List(ctx context.Context) (*managementv1.ListProjectsResponse, error) {
	out, resp, err := p.api.ListProjects(ctx).Execute()
	return entityResult("list projects", out, resp, err)
}

// Rename changes the display name of the project named by WithProject, or of
// the server's default project when none is named.
func (p *Projects) Rename(ctx context.Context, req *managementv1.RenameProjectRequest, callOpts ...CallOption) error {
	const op = "rename project"
	if req == nil {
		return errors.New(op + ": request must not be nil")
	}
	call, err := withProject(op, p.api.RenameProject(ctx), callOpts)
	if err != nil {
		return err
	}
	resp, err := call.RenameProjectRequest(*req).Execute()
	return WrapAPIError(op, err, resp)
}

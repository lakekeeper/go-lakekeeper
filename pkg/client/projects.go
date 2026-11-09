package client

import (
	"context"
	"errors"

	managementv1 "github.com/lakekeeper/go-lakekeeper/pkg/apis/management/v1"
)

// Projects is a one-call façade over ProjectAPIService. Get, Rename and
// Delete act on the project named by WithProject (sent as x-project-id) and,
// without it, on the server's default project. Create and List are not
// project-scoped. For endpoints without a façade, reach for c.ProjectAPI.
type Projects struct {
	api *managementv1.ProjectAPIService
}

// Create creates a new project and returns the resulting metadata.
func (p *Projects) Create(ctx context.Context, req *managementv1.CreateProjectRequest) (*managementv1.CreateProjectResponse, error) {
	if req == nil {
		return nil, errors.New("create project: request must not be nil")
	}
	out, resp, err := p.api.CreateProject(ctx).CreateProjectRequest(*req).Execute()
	return out, WrapAPIError("create project", err, resp)
}

// Get fetches the project named by WithProject, or the server's default
// project when none is named.
func (p *Projects) Get(ctx context.Context, callOpts ...CallOption) (*managementv1.GetProjectResponse, error) {
	out, resp, err := withProject(p.api.GetProject(ctx), callOpts).Execute()
	return out, WrapAPIError("get project", err, resp)
}

// Delete removes the project named by WithProject. Without it, the server
// deletes its default project, so name the project unless that is intended.
func (p *Projects) Delete(ctx context.Context, callOpts ...CallOption) error {
	resp, err := withProject(p.api.DeleteProject(ctx), callOpts).Execute()
	return WrapAPIError("delete project", err, resp)
}

// List returns all projects visible to the caller.
func (p *Projects) List(ctx context.Context) (*managementv1.ListProjectsResponse, error) {
	out, resp, err := p.api.ListProjects(ctx).Execute()
	return out, WrapAPIError("list projects", err, resp)
}

// Rename changes the display name of the project named by WithProject, or of
// the server's default project when none is named.
func (p *Projects) Rename(ctx context.Context, req *managementv1.RenameProjectRequest, callOpts ...CallOption) error {
	if req == nil {
		return errors.New("rename project: request must not be nil")
	}
	resp, err := withProject(p.api.RenameProject(ctx), callOpts).RenameProjectRequest(*req).Execute()
	return WrapAPIError("rename project", err, resp)
}

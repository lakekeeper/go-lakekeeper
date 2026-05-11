package client

import (
	"context"
	"errors"

	managementv1 "github.com/lakekeeper/go-lakekeeper/pkg/apis/management/v1"
)

// Projects is a one-call façade over ProjectAPIService. Most methods take
// the project id via header (X-Project-Id); pass it as the projectID
// argument here. For per-call request options, reach for c.ProjectAPI.
type Projects struct {
	api *managementv1.ProjectAPIService
}

// Create creates a new project and returns the resulting metadata.
func (p *Projects) Create(ctx context.Context, req *managementv1.CreateProjectRequest) (*managementv1.CreateProjectResponse, error) {
	if req == nil {
		return nil, errors.New("create project: request must not be nil")
	}
	out, _, err := p.api.CreateProject(ctx).CreateProjectRequest(*req).Execute()
	return out, err
}

// Get fetches a project by id.
func (p *Projects) Get(ctx context.Context, projectID string) (*managementv1.GetProjectResponse, error) {
	out, _, err := p.api.GetProject(ctx).XProjectId(projectID).Execute()
	return out, err
}

// Delete removes a project by id.
func (p *Projects) Delete(ctx context.Context, projectID string) error {
	_, err := p.api.DeleteProject(ctx).XProjectId(projectID).Execute()
	return err
}

// List returns all projects visible to the caller.
func (p *Projects) List(ctx context.Context) (*managementv1.ListProjectsResponse, error) {
	out, _, err := p.api.ListProjects(ctx).Execute()
	return out, err
}

// Rename updates a project's display name.
func (p *Projects) Rename(ctx context.Context, projectID string, req *managementv1.RenameProjectRequest) error {
	if req == nil {
		return errors.New("rename project: request must not be nil")
	}
	_, err := p.api.RenameProject(ctx).XProjectId(projectID).RenameProjectRequest(*req).Execute()
	return err
}

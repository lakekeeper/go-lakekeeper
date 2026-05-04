//go:build integration

package integration

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	managementv1 "github.com/lakekeeper/go-lakekeeper/pkg/apis/management/v1"
)

func TestProject_Create(t *testing.T) {
	c := sharedClient

	req := managementv1.NewCreateProjectRequest(randomName("test-project"))
	resp, r, err := c.ProjectAPI.CreateProject(t.Context()).CreateProjectRequest(*req).Execute()
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, r.StatusCode)
	assert.NotEmpty(t, resp.ProjectId)

	t.Cleanup(func() {
		r, err := c.ProjectAPI.DeleteProject(context.Background()).XProjectId(resp.ProjectId).Execute()
		if err != nil {
			t.Errorf("delete project: %v", err)
			return
		}
		assert.Equal(t, http.StatusNoContent, r.StatusCode)
	})
}

func TestProject_Rename(t *testing.T) {
	c := sharedClient

	req := managementv1.NewCreateProjectRequest(randomName("test-project"))
	created, r, err := c.ProjectAPI.CreateProject(t.Context()).CreateProjectRequest(*req).Execute()
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, r.StatusCode)
	assert.NotEmpty(t, created.ProjectId)

	t.Cleanup(func() {
		r, err := c.ProjectAPI.DeleteProject(context.Background()).XProjectId(created.ProjectId).Execute()
		if err != nil {
			t.Errorf("delete project: %v", err)
			return
		}
		assert.Equal(t, http.StatusNoContent, r.StatusCode)
	})

	renamed := randomName("test-project-renamed")
	rename := managementv1.NewRenameProjectRequest(renamed)
	r, err = c.ProjectAPI.RenameProject(t.Context()).XProjectId(created.ProjectId).RenameProjectRequest(*rename).Execute()
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, r.StatusCode)

	got, r, err := c.ProjectAPI.GetProject(t.Context()).XProjectId(created.ProjectId).Execute()
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, r.StatusCode)
	assert.Equal(t, renamed, got.ProjectName)
}

func TestProject_Delete(t *testing.T) {
	c := sharedClient

	req := managementv1.NewCreateProjectRequest(randomName("test-project"))
	created, r, err := c.ProjectAPI.CreateProject(t.Context()).CreateProjectRequest(*req).Execute()
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, r.StatusCode)
	assert.NotEmpty(t, created.ProjectId)

	r, err = c.ProjectAPI.DeleteProject(t.Context()).XProjectId(created.ProjectId).Execute()
	require.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, r.StatusCode)

	got, r, err := c.ProjectAPI.GetProject(t.Context()).XProjectId(created.ProjectId).Execute()
	// Lakekeeper currently returns 403 for reads of a deleted/non-existent
	// project; accept 404 too so a future server-side fix doesn't break us.
	require.Error(t, err)
	require.NotNil(t, r)
	assert.Contains(t, []int{http.StatusForbidden, http.StatusNotFound}, r.StatusCode)
	assert.Nil(t, got)
}

func TestProject_List(t *testing.T) {
	c := sharedClient

	resp, r, err := c.ProjectAPI.ListProjects(t.Context()).Execute()
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, r.StatusCode)

	// Other tests run concurrently and may have a project of their own visible
	// at this moment; only assert the default project is present, not that
	// it's the only one.
	assert.Contains(t, resp.Projects, managementv1.GetProjectResponse{
		ProjectId:   defaultProjectID,
		ProjectName: "Default Project",
	})
}

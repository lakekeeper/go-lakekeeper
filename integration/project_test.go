//go:build integration

package integration

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	managementv1 "github.com/lakekeeper/go-lakekeeper/pkg/apis/management/v1"
	"github.com/lakekeeper/go-lakekeeper/pkg/client"
)

func TestProject_Create(t *testing.T) {
	c := sharedClient

	name := randomName("test-project")
	created, err := c.Projects.Create(t.Context(), managementv1.NewCreateProjectRequest(name))
	require.NoError(t, err)
	require.NotEmpty(t, created.GetProjectId())

	t.Cleanup(func() {
		if err := c.Projects.Delete(context.Background(), client.WithProject(created.GetProjectId())); err != nil {
			t.Errorf("delete project: %v", err)
		}
	})

	got, err := c.Projects.Get(t.Context(), client.WithProject(created.GetProjectId()))
	require.NoError(t, err)
	assert.Equal(t, created.GetProjectId(), got.GetProjectId())
	assert.Equal(t, name, got.GetProjectName())
}

func TestProject_Rename(t *testing.T) {
	c := sharedClient
	project := MustCreateProject(t, c)

	renamed := randomName("test-project-renamed")
	err := c.Projects.Rename(t.Context(), managementv1.NewRenameProjectRequest(renamed), client.WithProject(project))
	require.NoError(t, err)

	got, err := c.Projects.Get(t.Context(), client.WithProject(project))
	require.NoError(t, err)
	assert.Equal(t, renamed, got.GetProjectName())
}

func TestProject_Delete(t *testing.T) {
	c := sharedClient

	created, err := c.Projects.Create(t.Context(), managementv1.NewCreateProjectRequest(randomName("test-project")))
	require.NoError(t, err)

	err = c.Projects.Delete(t.Context(), client.WithProject(created.GetProjectId()))
	require.NoError(t, err)

	got, err := c.Projects.Get(t.Context(), client.WithProject(created.GetProjectId()))
	// Lakekeeper currently returns 403 for reads of a deleted/non-existent
	// project; accept 404 too so a future server-side fix doesn't break us.
	var apiErr *client.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Contains(t, []int{http.StatusForbidden, http.StatusNotFound}, apiErr.StatusCode)
	assert.Nil(t, got)
}

func TestProject_Get_WithoutProjectReturnsDefaultProject(t *testing.T) {
	c := sharedClient

	got, err := c.Projects.Get(t.Context())
	require.NoError(t, err)
	assert.Equal(t, defaultProjectID, got.GetProjectId())
}

func TestProject_List(t *testing.T) {
	c := sharedClient

	resp, err := c.Projects.List(t.Context())
	require.NoError(t, err)

	// Other tests run concurrently and may have a project of their own visible
	// at this moment; only assert the default project is present, not that
	// it's the only one.
	assert.Contains(t, resp.GetProjects(), managementv1.GetProjectResponse{
		ProjectId:   defaultProjectID,
		ProjectName: "Default Project",
	})
}

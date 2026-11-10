package client_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	managementv1 "github.com/lakekeeper/go-lakekeeper/pkg/apis/management/v1"
	"github.com/lakekeeper/go-lakekeeper/pkg/client"
	"github.com/lakekeeper/go-lakekeeper/pkg/testutil"
)

func TestProjects_NilRequestsRejected(t *testing.T) {
	t.Parallel()
	_, c := testutil.ServerMux(t)

	_, err := c.Projects.Create(t.Context(), nil)
	require.ErrorContains(t, err, "request must not be nil")

	err = c.Projects.Rename(t.Context(), nil, client.WithProject(projectA))
	require.ErrorContains(t, err, "request must not be nil")
}

func TestProjects_Get_WithProjectSendsHeaderAndReturnsProject(t *testing.T) {
	t.Parallel()
	mux, c := testutil.ServerMux(t)
	reqs := testutil.Respond(t, mux, "GET /management/v1/project", http.StatusOK,
		`{"project-id":"11111111-1111-1111-1111-111111111111","project-name":"Alpha"}`)

	got, err := c.Projects.Get(t.Context(), client.WithProject(projectA))

	require.NoError(t, err)
	assert.Equal(t, projectA, got.GetProjectId())
	assert.Equal(t, "Alpha", got.GetProjectName())
	assert.Equal(t, projectA, reqs.Last(t).Header.Get("x-project-id"))
}

func TestProjects_Get_WithoutProjectSendsNoProject(t *testing.T) {
	t.Parallel()
	mux, c := testutil.ServerMux(t)
	reqs := testutil.Respond(t, mux, "GET /management/v1/project", http.StatusOK,
		`{"project-id":"00000000-0000-0000-0000-000000000000","project-name":"Default Project"}`)

	got, err := c.Projects.Get(t.Context())

	require.NoError(t, err)
	assert.Equal(t, "Default Project", got.GetProjectName())
	assert.NotContains(t, reqs.Last(t).Header, "X-Project-Id", "without WithProject the server's default project applies")
}

func TestProjects_Delete_WithProjectSendsHeader(t *testing.T) {
	t.Parallel()
	mux, c := testutil.ServerMux(t)
	reqs := testutil.Respond(t, mux, "DELETE /management/v1/project", http.StatusNoContent, nil)

	err := c.Projects.Delete(t.Context(), client.WithProject(projectA))

	require.NoError(t, err)
	assert.Equal(t, projectA, reqs.Last(t).Header.Get("x-project-id"))
}

// Deleting is the one call where the server's default project is never a
// safe fallback: without WithProject it would delete that project. Delete
// therefore requires the option, unlike every other project-scoped method.
func TestProjects_Delete_WithoutProjectIsRejected(t *testing.T) {
	t.Parallel()
	mux, c := testutil.ServerMux(t)
	reqs := testutil.Respond(t, mux, "DELETE /management/v1/project", http.StatusNoContent, nil)

	err := c.Projects.Delete(t.Context())

	require.ErrorIs(t, err, client.ErrProjectRequired)
	assert.Zero(t, reqs.Len(), "no request must be sent")
}

func TestProjects_Create_SendsBodyAndReturnsID(t *testing.T) {
	t.Parallel()
	mux, c := testutil.ServerMux(t)
	reqs := testutil.Respond(t, mux, "POST /management/v1/project", http.StatusCreated,
		`{"project-id":"11111111-1111-1111-1111-111111111111"}`)

	got, err := c.Projects.Create(t.Context(), managementv1.NewCreateProjectRequest("Alpha"))

	require.NoError(t, err)
	assert.Equal(t, projectA, got.GetProjectId())
	sent := reqs.Last(t)
	assert.JSONEq(t, `{"project-name":"Alpha"}`, string(sent.Body))
	assert.NotContains(t, sent.Header, "X-Project-Id")
}

func TestProjects_List_ReturnsGeneratedResponse(t *testing.T) {
	t.Parallel()
	mux, c := testutil.ServerMux(t)
	reqs := testutil.Respond(t, mux, "GET /management/v1/project-list", http.StatusOK, `{"projects":[
	  {"project-id":"00000000-0000-0000-0000-000000000000","project-name":"Default Project"},
	  {"project-id":"11111111-1111-1111-1111-111111111111","project-name":"Alpha"}]}`)

	got, err := c.Projects.List(t.Context())

	require.NoError(t, err)
	require.Len(t, got.GetProjects(), 2)
	assert.Equal(t, projectA, got.GetProjects()[1].GetProjectId())
	assert.Equal(t, "Alpha", got.GetProjects()[1].GetProjectName())
	sent := reqs.Last(t)
	assert.Empty(t, sent.Query)
	assert.NotContains(t, sent.Header, "X-Project-Id")
}

func TestProjects_Get_ServerErrorIsAPIError(t *testing.T) {
	t.Parallel()
	mux, c := testutil.ServerMux(t)
	testutil.Respond(t, mux, "GET /management/v1/project", http.StatusForbidden,
		`{"error":{"code":403,"message":"Forbidden action get_metadata on project","type":"NotAuthorized","stack":[]}}`)

	_, err := c.Projects.Get(t.Context(), client.WithProject(projectA))

	require.ErrorContains(t, err, "get project: 403 Forbidden")
	var apiErr *client.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, http.StatusForbidden, apiErr.StatusCode)
	require.NotNil(t, apiErr.Model)
	assert.Equal(t, "NotAuthorized", apiErr.Model.GetType())
	var genErr *managementv1.GenericOpenAPIError
	require.ErrorAs(t, err, &genErr, "errors.As must reach the generated error")
}

func TestProjects_Rename_SendsBodyAndHeader(t *testing.T) {
	t.Parallel()
	mux, c := testutil.ServerMux(t)
	reqs := testutil.Respond(t, mux, "POST /management/v1/project/rename", http.StatusOK, nil)

	err := c.Projects.Rename(t.Context(), managementv1.NewRenameProjectRequest("Beta"), client.WithProject(projectA))

	require.NoError(t, err)
	got := reqs.Last(t)
	assert.Equal(t, projectA, got.Header.Get("x-project-id"))
	assert.JSONEq(t, `{"new-name":"Beta"}`, string(got.Body))
}

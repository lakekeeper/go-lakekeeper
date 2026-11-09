package client_test

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	managementv1 "github.com/lakekeeper/go-lakekeeper/pkg/apis/management/v1"
	"github.com/lakekeeper/go-lakekeeper/pkg/client"
	"github.com/lakekeeper/go-lakekeeper/pkg/testutil"
)

const (
	roleID   = "0198618c-5be8-7a82-a0b9-1076c9dd12f0"
	roleBody = `{"id":"0198618c-5be8-7a82-a0b9-1076c9dd12f0","name":"data-eng","project-id":"11111111-1111-1111-1111-111111111111",
	  "provider-id":"lakekeeper","source-id":"0198618c-5be8-7a82-a0b9-1076c9dd12f0","created-at":"2026-01-02T03:04:05Z",
	  "ident":"lakekeeper~0198618c-5be8-7a82-a0b9-1076c9dd12f0"}`
)

func TestRoles_Get_WithProjectSendsHeaderAndReturnsRole(t *testing.T) {
	t.Parallel()
	mux, c := testutil.ServerMux(t)
	reqs := testutil.Respond(t, mux, "GET /management/v1/role/{id}", http.StatusOK, roleBody)

	got, err := c.Roles.Get(t.Context(), roleID, client.WithProject(projectA))

	require.NoError(t, err)
	assert.Equal(t, roleID, got.GetId())
	assert.Equal(t, "data-eng", got.GetName())
	sent := reqs.Last(t)
	assert.Equal(t, "/management/v1/role/"+roleID, sent.Path)
	assert.Equal(t, projectA, sent.Header.Get("x-project-id"))
}

func TestRoles_Create_SendsBodyAndHeaderAndReturnsRole(t *testing.T) {
	t.Parallel()
	mux, c := testutil.ServerMux(t)
	reqs := testutil.Respond(t, mux, "POST /management/v1/role", http.StatusCreated, roleBody)
	req := managementv1.NewCreateRoleRequest("data-eng")
	req.SetDescription("data engineering team")

	got, err := c.Roles.Create(t.Context(), req, client.WithProject(projectA))

	require.NoError(t, err)
	assert.Equal(t, roleID, got.GetId())
	sent := reqs.Last(t)
	assert.Equal(t, projectA, sent.Header.Get("x-project-id"))
	assert.JSONEq(t, `{"name":"data-eng","description":"data engineering team"}`, string(sent.Body),
		"the deprecated project-id body field must not be sent")
}

func TestRoles_Update_SendsBodyAndHeaderAndReturnsRole(t *testing.T) {
	t.Parallel()
	mux, c := testutil.ServerMux(t)
	reqs := testutil.Respond(t, mux, "POST /management/v1/role/{id}", http.StatusOK, roleBody)

	got, err := c.Roles.Update(t.Context(), roleID, managementv1.NewUpdateRoleRequest("data-eng"), client.WithProject(projectA))

	require.NoError(t, err)
	assert.Equal(t, "data-eng", got.GetName())
	sent := reqs.Last(t)
	assert.Equal(t, "/management/v1/role/"+roleID, sent.Path)
	assert.Equal(t, projectA, sent.Header.Get("x-project-id"))
	assert.JSONEq(t, `{"name":"data-eng"}`, string(sent.Body))
}

func TestRoles_List_SendsPagingAndHeaderAndReturnsPage(t *testing.T) {
	t.Parallel()
	mux, c := testutil.ServerMux(t)
	reqs := testutil.Respond(t, mux, "GET /management/v1/role", http.StatusOK,
		`{"roles":[`+roleBody+`],"next-page-token":"page-3"}`)

	got, err := c.Roles.List(t.Context(), &client.ListRolesOptions{PageSize: 1, PageToken: "page-2"}, client.WithProject(projectA))

	require.NoError(t, err)
	require.Len(t, got.GetRoles(), 1)
	assert.Equal(t, roleID, got.GetRoles()[0].GetId())
	assert.Equal(t, "page-3", got.GetNextPageToken())
	sent := reqs.Last(t)
	assert.Equal(t, projectA, sent.Header.Get("x-project-id"))
	assert.Equal(t, url.Values{"pageSize": {"1"}, "pageToken": {"page-2"}}, sent.Query,
		"only the paging parameters may be sent, never the deprecated projectId")
}

func TestRoles_List_NilOptionsSendNoQuery(t *testing.T) {
	t.Parallel()
	mux, c := testutil.ServerMux(t)
	reqs := testutil.Respond(t, mux, "GET /management/v1/role", http.StatusOK, `{"roles":[]}`)

	_, err := c.Roles.List(t.Context(), nil)

	require.NoError(t, err)
	sent := reqs.Last(t)
	assert.Empty(t, sent.Query)
	assert.NotContains(t, sent.Header, "X-Project-Id")
}

func TestRoles_Delete_WithoutProjectSendsNoProject(t *testing.T) {
	t.Parallel()
	mux, c := testutil.ServerMux(t)
	reqs := testutil.Respond(t, mux, "DELETE /management/v1/role/{id}", http.StatusNoContent, nil)

	err := c.Roles.Delete(t.Context(), roleID)

	require.NoError(t, err)
	sent := reqs.Last(t)
	assert.Equal(t, "/management/v1/role/"+roleID, sent.Path)
	assert.NotContains(t, sent.Header, "X-Project-Id", "without WithProject the server's default project applies")
	assert.Empty(t, sent.Query, "neither the deprecated projectId nor force may be sent")
}

func TestRoles_Search_SendsBodyAndHeaderAndReturnsMatches(t *testing.T) {
	t.Parallel()
	mux, c := testutil.ServerMux(t)
	reqs := testutil.Respond(t, mux, "POST /management/v1/search/role", http.StatusOK, `{"roles":[`+roleBody+`]}`)

	got, err := c.Roles.Search(t.Context(), managementv1.NewSearchRoleRequest("data"), client.WithProject(projectA))

	require.NoError(t, err)
	require.Len(t, got.GetRoles(), 1)
	assert.Equal(t, "data-eng", got.GetRoles()[0].GetName())
	sent := reqs.Last(t)
	assert.Equal(t, projectA, sent.Header.Get("x-project-id"))
	assert.JSONEq(t, `{"search":"data"}`, string(sent.Body), "the deprecated project-id body field must not be sent")
}

func TestRoles_NilRequestsRejected(t *testing.T) {
	t.Parallel()
	_, c := testutil.ServerMux(t)

	_, err := c.Roles.Create(t.Context(), nil)
	require.ErrorContains(t, err, "create role: request must not be nil")

	_, err = c.Roles.Update(t.Context(), roleID, nil)
	require.ErrorContains(t, err, "update role: request must not be nil")

	_, err = c.Roles.Search(t.Context(), nil)
	require.ErrorContains(t, err, "search roles: request must not be nil")
}

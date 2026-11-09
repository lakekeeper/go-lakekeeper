//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	managementv1 "github.com/lakekeeper/go-lakekeeper/pkg/apis/management/v1"
	"github.com/lakekeeper/go-lakekeeper/pkg/client"
)

func TestRole_Lifecycle(t *testing.T) {
	t.Parallel()
	c := sharedClient
	project := MustCreateProject(t, c)

	name := randomName("test-role")
	createReq := managementv1.NewCreateRoleRequest(name)
	createReq.SetDescription("created by TestRole_Lifecycle")
	created, err := c.Roles.Create(t.Context(), createReq, client.WithProject(project))
	require.NoError(t, err)
	assert.Equal(t, project, created.GetProjectId())

	got, err := c.Roles.Get(t.Context(), created.GetId(), client.WithProject(project))
	require.NoError(t, err)
	assert.Equal(t, name, got.GetName())
	assert.Equal(t, "created by TestRole_Lifecycle", got.GetDescription())

	renamed := randomName("test-role-renamed")
	updated, err := c.Roles.Update(t.Context(), created.GetId(), managementv1.NewUpdateRoleRequest(renamed), client.WithProject(project))
	require.NoError(t, err)
	assert.Equal(t, renamed, updated.GetName())

	page, err := c.Roles.List(t.Context(), &client.ListRolesOptions{PageSize: 100}, client.WithProject(project))
	require.NoError(t, err)
	require.Len(t, page.GetRoles(), 1, "a fresh project holds only the created role")
	assert.Equal(t, created.GetId(), page.GetRoles()[0].GetId())

	require.NoError(t, c.Roles.Delete(t.Context(), created.GetId(), client.WithProject(project)))

	_, err = c.Roles.Get(t.Context(), created.GetId(), client.WithProject(project))
	var apiErr *client.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, http.StatusNotFound, apiErr.StatusCode)
}

func TestRole_List_ProjectScoping(t *testing.T) {
	t.Parallel()
	c := sharedClient
	project := MustCreateProject(t, c)
	role := MustCreateRole(t, c, project)

	inProject, err := c.Roles.List(t.Context(), nil, client.WithProject(project))
	require.NoError(t, err)
	assert.Equal(t, []string{role.GetId()}, roleIDs(inProject.GetRoles()))

	_, err = c.Roles.Get(t.Context(), role.GetId())
	var apiErr *client.APIError
	require.ErrorAs(t, err, &apiErr, "without WithProject the role must be looked up in the default project")
	assert.Contains(t, []int{http.StatusForbidden, http.StatusNotFound}, apiErr.StatusCode)
}

func TestRole_List_PagesWithToken(t *testing.T) {
	t.Parallel()
	c := sharedClient
	project := MustCreateProject(t, c)
	want := []string{MustCreateRole(t, c, project).GetId(), MustCreateRole(t, c, project).GetId()}

	first, err := c.Roles.List(t.Context(), &client.ListRolesOptions{PageSize: 1}, client.WithProject(project))
	require.NoError(t, err)
	require.Len(t, first.GetRoles(), 1)
	require.NotEmpty(t, first.GetNextPageToken())

	second, err := c.Roles.List(t.Context(),
		&client.ListRolesOptions{PageSize: 1, PageToken: first.GetNextPageToken()}, client.WithProject(project))
	require.NoError(t, err)

	assert.ElementsMatch(t, want, roleIDs(append(first.GetRoles(), second.GetRoles()...)))
}

func TestRole_Search_FindsByName(t *testing.T) {
	t.Parallel()
	c := sharedClient
	project := MustCreateProject(t, c)
	role := MustCreateRole(t, c, project)

	found, err := c.Roles.Search(t.Context(), managementv1.NewSearchRoleRequest(role.GetName()), client.WithProject(project))
	require.NoError(t, err)
	assert.Contains(t, roleIDs(found.GetRoles()), role.GetId())
}

// `role list --name` (findRolesByName) relies on role names being unique per
// project, ignoring case.
func TestRole_Create_NameIsUniqueInProjectIgnoringCase(t *testing.T) {
	t.Parallel()
	c := sharedClient
	project := MustCreateProject(t, c)
	role := MustCreateRole(t, c, project)

	_, err := c.Roles.Create(t.Context(),
		managementv1.NewCreateRoleRequest(strings.ToUpper(role.GetName())), client.WithProject(project))

	var apiErr *client.APIError
	require.ErrorAs(t, err, &apiErr, "a second role with the same name must be rejected")
	assert.Equal(t, http.StatusConflict, apiErr.StatusCode)
}

// Lakekeeper's role search returns the 10 roles nearest by trigram distance.
// Names that differ only in punctuation share every trigram, so they tie at
// distance 0 and the search can leave out the one asked for. `role list
// --name` must still find each of them.
func TestCLI_RoleListName_FindsRolesBeyondTheSearchCap(t *testing.T) {
	t.Parallel()
	c := sharedClient
	project := MustCreateProject(t, c)

	word := strings.ReplaceAll(uuid.NewString()[:8], "-", "")
	byName := map[string]string{}
	for _, sep := range []string{"-", "_", ".", " ", "+", ":", "@", "=", ",", ";", "~", "!"} {
		name := "tie" + sep + word
		role, err := c.Roles.Create(t.Context(), managementv1.NewCreateRoleRequest(name), client.WithProject(project))
		require.NoError(t, err, "create role %q", name)
		byName[name] = role.GetId()
		t.Cleanup(func() {
			if err := c.Roles.Delete(context.Background(), role.GetId(), client.WithProject(project)); err != nil {
				t.Errorf("delete role: %v", err)
			}
		})
	}

	searchMisses := 0
	for name := range byName {
		found, err := c.Roles.Search(t.Context(), managementv1.NewSearchRoleRequest(name), client.WithProject(project))
		require.NoError(t, err)
		if !containsRoleNamed(found.GetRoles(), name) {
			searchMisses++
		}
	}
	t.Logf("role search left out the exact name for %d of %d tied roles", searchMisses, len(byName))

	token := freshKeycloakToken(t)
	for name, id := range byName {
		out := runLkctl(t,
			"role", "list", "--name", name, "-p", project, "--output", "json",
			"--base-url", os.Getenv("LAKEKEEPER_BASE_URL"),
			"--auth-mode", "token",
			"--access-token", token,
		)
		var roles []managementv1.Role
		require.NoError(t, json.Unmarshal(out, &roles), "decode lkctl output: %s", out)
		assert.Equal(t, []string{id}, roleIDs(roles), "role list --name %q", name)
	}
}

func roleIDs(roles []managementv1.Role) []string {
	ids := make([]string, 0, len(roles))
	for i := range roles {
		ids = append(ids, roles[i].GetId())
	}
	return ids
}

func containsRoleNamed(roles []managementv1.Role, name string) bool {
	for i := range roles {
		if roles[i].GetName() == name {
			return true
		}
	}
	return false
}

//go:build integration

// Package integration runs end-to-end tests against a real Lakekeeper +
// Keycloak + MinIO + OpenFGA stack provisioned by docker-compose. Invoked via
// `make test-integration` which provisions `.env`, brings the stack up, waits
// for health, and runs `go test -tags integration ./integration/...`.
package integration

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"golang.org/x/oauth2/clientcredentials"

	managementv1 "github.com/lakekeeper/go-lakekeeper/pkg/apis/management/v1"
	"github.com/lakekeeper/go-lakekeeper/pkg/client"
	"github.com/lakekeeper/go-lakekeeper/pkg/core"
	"github.com/lakekeeper/go-lakekeeper/pkg/permissions"
	"github.com/lakekeeper/go-lakekeeper/pkg/storage/credential"
	"github.com/lakekeeper/go-lakekeeper/pkg/storage/profile"
)

// adminID is the fixed Keycloak subject for the bootstrap admin user that the
// docker-compose realm provisions; tests assert this principal owns
// server/project/role/warehouse permissions on initial state. Matches the
// `sub` of the seeded admin user in tests/keycloak-realm.json — change one
// and the other must follow.
const adminID = "oidc~6deeb417-cdf9-4320-8a30-ddecea77a4bd"

// randomName produces a short collision-resistant suffix for test resources.
// Centralized so we can swap the random source (currently UUID-derived) once,
// not per call site.
func randomName(prefix string) string {
	return fmt.Sprintf("%s-%s", prefix, uuid.NewString()[:8])
}

// freshKeycloakToken mints a fresh Keycloak access token via the same
// client-credentials flow TestMain uses. Used by the per-mode auth tests
// (auth_test.go, cli_test.go) which need the raw bearer string to feed into
// AccessTokenAuthSource or to write to a fake service-account-token file.
func freshKeycloakToken(t *testing.T) string {
	t.Helper()
	cfg := clientcredentials.Config{
		ClientID:     os.Getenv("LAKEKEEPER_CLIENT_ID"),
		ClientSecret: os.Getenv("LAKEKEEPER_CLIENT_SECRET"),
		TokenURL:     os.Getenv("LAKEKEEPER_TOKEN_URL"),
		Scopes:       []string{os.Getenv("LAKEKEEPER_SCOPE")},
	}
	tok, err := cfg.TokenSource(t.Context()).Token()
	if err != nil {
		t.Fatalf("mint keycloak token: %v", err)
	}
	return tok.AccessToken
}

// defaultProjectID is the all-zeros UUID that the server returns as the
// default project after bootstrap.
var defaultProjectID = uuid.Nil.String()

// sharedClient is built once in TestMain and reused by every test. The OAuth
// token-exchange and the (idempotent) initial-bootstrap call are expensive
// enough that doing them per-test was visibly slowing the suite.
var sharedClient *client.Client

func TestMain(m *testing.M) {
	if err := godotenv.Load("../.env"); err != nil {
		fmt.Fprintf(os.Stderr, "load .env: %v\n", err)
		os.Exit(1)
	}

	oauthCfg := clientcredentials.Config{
		ClientID:     os.Getenv("LAKEKEEPER_CLIENT_ID"),
		ClientSecret: os.Getenv("LAKEKEEPER_CLIENT_SECRET"),
		TokenURL:     os.Getenv("LAKEKEEPER_TOKEN_URL"),
		Scopes:       []string{os.Getenv("LAKEKEEPER_SCOPE")},
	}
	tokenSource := oauthCfg.TokenSource(context.Background())
	if _, err := tokenSource.Token(); err != nil {
		fmt.Fprintf(os.Stderr, "oauth2 token: %v\n", err)
		os.Exit(1)
	}

	authSource := &core.OAuthTokenSource{TokenSource: tokenSource}

	c, err := client.NewWithAuthSource(
		context.Background(),
		os.Getenv("LAKEKEEPER_BASE_URL"),
		authSource,
		client.WithInitialBootstrap(true, true, core.Ptr(managementv1.USERTYPE_APPLICATION)),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "create client: %v\n", err)
		os.Exit(1)
	}
	sharedClient = c

	code := m.Run()
	if lkctlBuildDir != "" {
		_ = os.RemoveAll(lkctlBuildDir)
	}
	os.Exit(code)
}

// MustProvisionUser creates a fresh randomly-named user and registers a
// t.Cleanup to delete it.
func MustProvisionUser(t *testing.T, c *client.Client) *managementv1.User {
	t.Helper()

	id := fmt.Sprintf("oidc~%s", uuid.New())
	name := randomName("test-user")
	email := name + "@example.com"

	req := managementv1.NewCreateUserRequest()
	req.SetId(id)
	req.SetName(name)
	req.SetEmail(email)
	req.SetUserType(managementv1.USERTYPE_HUMAN)
	req.SetUpdateIfExists(false)

	user, _, err := c.UserAPI.CreateUser(t.Context()).CreateUserRequest(*req).Execute()
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	t.Cleanup(func() {
		if _, err := c.UserAPI.DeleteUser(context.Background(), user.Id).Execute(); err != nil {
			t.Errorf("delete user: %v", err)
		}
	})
	return user
}

// MustCreateRole creates a fresh randomly-named role under the given project
// and registers a t.Cleanup to delete it.
func MustCreateRole(t *testing.T, c *client.Client, projectID string) *managementv1.Role {
	t.Helper()

	req := managementv1.NewCreateRoleRequest(randomName("test-role"))
	role, _, err := c.RoleAPI.CreateRole(t.Context()).XProjectId(projectID).CreateRoleRequest(*req).Execute()
	if err != nil {
		t.Fatalf("create role: %v", err)
	}

	t.Cleanup(func() {
		if _, err := c.RoleAPI.DeleteRole(context.Background(), role.Id).XProjectId(projectID).Execute(); err != nil {
			t.Errorf("delete role: %v", err)
		}
	})
	return role
}

// MustCreateProject creates a fresh randomly-named project and registers a
// t.Cleanup to delete it. Returns the new project ID.
func MustCreateProject(t *testing.T, c *client.Client) string {
	t.Helper()

	req := managementv1.NewCreateProjectRequest(randomName("test-project"))
	created, _, err := c.ProjectAPI.CreateProject(t.Context()).CreateProjectRequest(*req).Execute()
	if err != nil {
		t.Fatalf("create project: %v", err)
	}

	t.Cleanup(func() {
		if _, err := c.ProjectAPI.DeleteProject(context.Background()).XProjectId(created.ProjectId).Execute(); err != nil {
			t.Errorf("delete project: %v", err)
		}
	})
	return created.ProjectId
}

// MustCreateWarehouse creates a warehouse against the in-stack MinIO under the
// given project, with credentials matching the docker-compose root user, and
// registers a t.Cleanup to delete it. Returns (warehouseID, warehouseName).
//
// Credentials are fixture values from docker-compose.yml, not real secrets.
func MustCreateWarehouse(t *testing.T, c *client.Client, projectID string) (string, string) {
	t.Helper()

	name := randomName("test-wh")

	sp := managementv1.StorageProfileS3AsStorageProfile(profile.NewS3Profile(
		"testacc", "eu-local-1",
		profile.WithS3Endpoint("http://minio:9000/"),
		profile.WithS3PathStyleAccess(),
	))
	sc := credential.NewS3AccessKey("minio-root-user", "minio-root-password")

	req := managementv1.NewCreateWarehouseRequest(sp, name)
	req.SetProjectId(projectID)
	req.SetStorageCredential(sc)

	wh, _, err := c.WarehouseAPI.CreateWarehouse(t.Context()).CreateWarehouseRequest(*req).Execute()
	if err != nil {
		t.Fatalf("create warehouse: %v", err)
	}

	t.Cleanup(func() {
		if _, err := c.WarehouseAPI.DeleteWarehouse(context.Background(), wh.WarehouseId).Execute(); err != nil {
			t.Errorf("delete warehouse: %v", err)
		}
	})
	return wh.WarehouseId, name
}

// *_GetAccess tests use assert.Subset(t, want, resp.AllowedActions), which
// checks resp.AllowedActions ⊆ want. This is deliberate:
//   - server-side additions (a new action appearing in resp) FAIL the test —
//     surfaced as news that should be reviewed and reflected in `want`.
//   - server-side removals (an action disappearing from resp) PASS — server
//     deprecations don't break the suite. If strict equality is needed for a
//     specific case, use assert.ElementsMatch.

// userAssignment constructs a generated *Assignment union value for a given
// relation tied to a user principal. Called from the test goroutine, so
// t.Fatalf is safe here (and desirable: a build failure should halt the
// test, not be silently overlooked).
func userAssignment[T any](t *testing.T, relation, userID string) T {
	t.Helper()
	a, err := permissions.BuildAssignment[T](relation, permissions.PrincipalUser, userID)
	if err != nil {
		t.Fatalf("build user assignment: %v", err)
	}
	return a
}

// roleAssignment is the role-principal counterpart to userAssignment.
func roleAssignment[T any](t *testing.T, relation, roleID string) T {
	t.Helper()
	a, err := permissions.BuildAssignment[T](relation, permissions.PrincipalRole, roleID)
	if err != nil {
		t.Fatalf("build role assignment: %v", err)
	}
	return a
}

// describeAssignments converts a slice of any *Assignment type into a slice
// of permissions.AssignmentRow values. Use this in `assert.ElementsMatch` to
// compare expected and actual assignments without depending on slice order
// or on direct struct equality of the oneOf unions.
//
// Wire-format invariant: every generated `*Assignment` type is expected to
// marshal to `{"type": <relation>, "user"|"role": <id>}`. If the OpenAPI
// spec or generator template ever drops/renames those keys (e.g. emits
// `principal_id` instead of `user`/`role`), `permissions.DescribeAssignment`
// returns `(_, false)` and every permission test fails with "could not
// decode assignment" — fix the contract in pkg/apis/management/v1, not
// here.
func describeAssignments[T any](t *testing.T, in []T) []permissions.AssignmentRow {
	t.Helper()
	out := make([]permissions.AssignmentRow, 0, len(in))
	for _, a := range in {
		row, ok := permissions.DescribeAssignment(a)
		if !ok {
			t.Fatalf("could not decode assignment %+v", a)
		}
		out = append(out, row)
	}
	return out
}

// errorBody returns the raw response body that the generated client wrapped
// into the given error, falling back to err.Error() when the error isn't a
// *GenericOpenAPIError. The generated client puts the HTTP status line into
// err.Error() and stashes the unparsed response body in Body(); we extract
// it here for substring matching on Lakekeeper's structured error payloads
// (e.g. "TupleAlreadyExistsError").
//
// Falling back to err.Error() means a future error wrapping (middleware,
// retry layer, …) doesn't degrade the assertion to ` "" does not contain
// "TupleAlreadyExistsError" `; the actual error message reaches the test
// log instead.
func errorBody(err error) string {
	if err == nil {
		return ""
	}
	var apiErr *managementv1.GenericOpenAPIError
	if errors.As(err, &apiErr) {
		return string(apiErr.Body())
	}
	return err.Error()
}

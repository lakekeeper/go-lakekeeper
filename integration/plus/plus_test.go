//go:build integration_plus

// Package plus runs SDK tests against a Lakekeeper+ server on the compose
// stack (profiles plus and plus-openfga). Invoked via `make test-plus`, which
// runs e2e/plus/run.sh: it brings up one variant at a time and sets
// LAKEKEEPER_PLUS_AUTHZ_BACKEND to the variant's authorization backend.
//
// The Lakekeeper suite in ../ stays on the Lakekeeper stack; it does not run
// here.
package plus

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2/clientcredentials"

	managementv1 "github.com/lakekeeper/go-lakekeeper/pkg/apis/management/v1"
	"github.com/lakekeeper/go-lakekeeper/pkg/client"
	"github.com/lakekeeper/go-lakekeeper/pkg/core"
)

// sharedClient is built and bootstrapped once in TestMain.
var sharedClient *client.Client

// authzBackend is the authorization backend of the running variant:
// "cedar" or "openfga".
var authzBackend client.AuthzBackend

func TestMain(m *testing.M) {
	if err := godotenv.Load("../../.env"); err != nil {
		fmt.Fprintf(os.Stderr, "load .env: %v\n", err)
		os.Exit(1)
	}
	authzBackend = client.AuthzBackend(os.Getenv("LAKEKEEPER_PLUS_AUTHZ_BACKEND"))
	if authzBackend == "" {
		fmt.Fprintln(os.Stderr, "LAKEKEEPER_PLUS_AUTHZ_BACKEND is not set; run through make test-plus")
		os.Exit(1)
	}

	cfg := clientcredentials.Config{
		ClientID:     os.Getenv("LAKEKEEPER_CLIENT_ID"),
		ClientSecret: os.Getenv("LAKEKEEPER_CLIENT_SECRET"),
		TokenURL:     os.Getenv("LAKEKEEPER_TOKEN_URL"),
		Scopes:       strings.Fields(os.Getenv("LAKEKEEPER_SCOPE")),
	}
	c, err := client.NewWithAuthSource(
		context.Background(),
		os.Getenv("LAKEKEEPER_BASE_URL"),
		&core.OAuthClientCredentialsAuthSource{TokenSource: cfg.TokenSource(context.Background())},
		client.WithInitialBootstrap(true, true, core.Ptr(managementv1.USERTYPE_APPLICATION)),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "create client: %v\n", err)
		os.Exit(1)
	}
	sharedClient = c

	os.Exit(m.Run())
}

func TestServerInfo_ReportsLakekeeperPlus(t *testing.T) {
	info, err := sharedClient.Server.Info(t.Context())
	require.NoError(t, err)

	assert.True(t, info.GetBootstrapped())
	assert.Equal(t, string(authzBackend), info.GetAuthzBackend())
	assert.NotEmpty(t, info.GetLakekeeperEnterpriseVersion(), "lakekeeper-enterprise-version")
}

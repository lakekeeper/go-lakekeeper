package commands

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClientReturnsValidationError(t *testing.T) {
	t.Parallel()

	_, err := newClient(context.Background(), &clientOptions{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "base url is required")
}

func TestClientOptionsValidate(t *testing.T) {
	t.Parallel()

	oauth2Full := clientOptions{
		baseURL:      "http://localhost:8181",
		authMode:     "oauth2",
		tokenURL:     "http://localhost:8080/realms/iceberg/protocol/openid-connect/token",
		clientID:     "id",
		clientSecret: "secret",
		scope:        []string{"lakekeeper"},
	}
	require.NoError(t, oauth2Full.validate())

	tokenFull := clientOptions{
		baseURL:     "http://localhost:8181",
		authMode:    "token",
		accessToken: "abc",
	}
	require.NoError(t, tokenFull.validate())

	k8sBare := clientOptions{
		baseURL:  "http://localhost:8181",
		authMode: "k8s",
	}
	require.NoError(t, k8sBare.validate())

	k8sWithPath := clientOptions{
		baseURL:      "http://localhost:8181",
		authMode:     "k8s",
		k8sTokenPath: "/var/run/secrets/kubernetes.io/serviceaccount/token",
	}
	require.NoError(t, k8sWithPath.validate())

	mutateOAuth2 := func(mutate func(*clientOptions)) clientOptions {
		out := oauth2Full
		mutate(&out)
		return out
	}

	cases := []struct {
		name    string
		opts    clientOptions
		wantMsg string
	}{
		{
			"base url missing",
			clientOptions{authMode: "oauth2"},
			"base url",
		},
		{
			"oauth2: token url missing",
			mutateOAuth2(func(o *clientOptions) { o.tokenURL = "" }),
			"token url",
		},
		{
			"oauth2: client id missing",
			mutateOAuth2(func(o *clientOptions) { o.clientID = "" }),
			"client id",
		},
		{
			"oauth2: client secret missing",
			mutateOAuth2(func(o *clientOptions) { o.clientSecret = "" }),
			"client secret",
		},
		{
			"oauth2: scope missing",
			mutateOAuth2(func(o *clientOptions) { o.scope = nil }),
			"scope",
		},
		{
			"token: access token missing",
			clientOptions{baseURL: "http://localhost:8181", authMode: "token"},
			"access token",
		},
		{
			"unknown auth mode",
			clientOptions{baseURL: "http://localhost:8181", authMode: "saml"},
			`unknown auth mode "saml"`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := tc.opts.validate()
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.wantMsg)
		})
	}
}

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
	assert.Contains(t, err.Error(), "server URL is required")
}

func TestClientOptionsValidate(t *testing.T) {
	t.Parallel()

	full := clientOptions{
		server:       "http://localhost:8181",
		authURL:      "http://localhost:8080/realms/iceberg/protocol/openid-connect/token",
		clientID:     "id",
		clientSecret: "secret",
		scope:        []string{"lakekeeper"},
	}

	require.NoError(t, full.validate())

	cases := []struct {
		name    string
		mutate  func(*clientOptions)
		wantMsg string
	}{
		{"server missing", func(o *clientOptions) { o.server = "" }, "server"},
		{"auth url missing", func(o *clientOptions) { o.authURL = "" }, "auth url"},
		{"client id missing", func(o *clientOptions) { o.clientID = "" }, "client id"},
		{"client secret missing", func(o *clientOptions) { o.clientSecret = "" }, "client secret"},
		{"scope missing", func(o *clientOptions) { o.scope = nil }, "scope"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			opts := full
			tc.mutate(&opts)
			err := opts.validate()
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.wantMsg)
		})
	}
}

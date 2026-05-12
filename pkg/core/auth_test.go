package core

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

func TestK8sAuthSource(t *testing.T) {
	t.Parallel()

	t.Run("Provided ServiceAccountTokenPath", func(t *testing.T) {
		t.Parallel()

		path := "/tmp/service-account-go-lakekeeper-test"
		if err := os.WriteFile(path, []byte("service-account-token"), 0o600); err != nil {
			t.Fatalf("Failed to create service account token file: %v", err)
		}

		t.Cleanup(func() {
			_ = os.Remove(path)
		})

		as := K8sServiceAccountAuthSource{
			ServiceAccountTokenPath: Ptr(path),
		}

		err := as.Init(context.Background())
		require.NoError(t, err)

		key, value, err := as.Header(context.Background())
		require.NoError(t, err)

		assert.Equal(t, "Authorization", key)
		assert.Contains(t, "Bearer service-account-token", value)
	})

	t.Run("Default ServiceAccountTokenPath", func(t *testing.T) {
		t.Parallel()

		as := K8sServiceAccountAuthSource{}

		err := as.Init(context.Background())
		require.Error(t, err, "failed to read service account token")

		assert.Equal(t, Ptr(DefaultK8sServiceAccountTokenPath), as.ServiceAccountTokenPath)
	})

	t.Run("Token re-read picks up rotation", func(t *testing.T) {
		t.Parallel()

		path := "/tmp/service-account-go-lakekeeper-rotation-test"
		require.NoError(t, os.WriteFile(path, []byte("first-token\n"), 0o600))
		t.Cleanup(func() { _ = os.Remove(path) })

		as := K8sServiceAccountAuthSource{ServiceAccountTokenPath: Ptr(path)}
		require.NoError(t, as.Init(context.Background()))

		_, value, err := as.Header(context.Background())
		require.NoError(t, err)
		assert.Equal(t, "Bearer first-token", value, "trailing newline must be trimmed")

		// Simulate a kubelet rotation by overwriting the file with a new token.
		// The auth source must surface the new value on the very next Header()
		// call without any explicit re-init from the caller.
		require.NoError(t, os.WriteFile(path, []byte("rotated-token"), 0o600))

		_, value, err = as.Header(context.Background())
		require.NoError(t, err)
		assert.Equal(t, "Bearer rotated-token", value)
	})

	t.Run("Empty token file rejected on read", func(t *testing.T) {
		t.Parallel()

		path := "/tmp/service-account-go-lakekeeper-empty-test"
		require.NoError(t, os.WriteFile(path, []byte("\n"), 0o600))
		t.Cleanup(func() { _ = os.Remove(path) })

		as := K8sServiceAccountAuthSource{ServiceAccountTokenPath: Ptr(path)}
		err := as.Init(context.Background())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "empty")
	})
}

func TestAccessTokenAuthSource_RejectsEmpty(t *testing.T) {
	t.Parallel()

	as := AccessTokenAuthSource{}
	err := as.Init(context.Background())
	require.Error(t, err, "empty token must be rejected at Init so misconfiguration surfaces eagerly")

	good := AccessTokenAuthSource{Token: "abc"}
	require.NoError(t, good.Init(context.Background()))
}

// stubTokenSource is a controllable oauth2.TokenSource for testing the
// OAuth init path without hitting a real token endpoint.
type stubTokenSource struct {
	token *oauth2.Token
	err   error
	calls int
}

func (s *stubTokenSource) Token() (*oauth2.Token, error) {
	s.calls++
	return s.token, s.err
}

func TestOAuthClientCredentialsAuthSource_InitFetchesToken(t *testing.T) {
	t.Parallel()

	t.Run("Init returns nil and primes the source on success", func(t *testing.T) {
		t.Parallel()
		stub := &stubTokenSource{
			token: &oauth2.Token{AccessToken: "abc", TokenType: "Bearer", Expiry: time.Now().Add(time.Hour)},
		}
		as := OAuthClientCredentialsAuthSource{TokenSource: stub}
		require.NoError(t, as.Init(context.Background()))
		assert.GreaterOrEqual(t, stub.calls, 1, "Init must touch the token source so misconfiguration surfaces here")
	})

	t.Run("Init surfaces token endpoint failures", func(t *testing.T) {
		t.Parallel()
		stub := &stubTokenSource{err: errors.New("token URL unreachable")}
		as := OAuthClientCredentialsAuthSource{TokenSource: stub}
		err := as.Init(context.Background())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "initial token fetch failed")
	})

	t.Run("Init rejects nil TokenSource", func(t *testing.T) {
		t.Parallel()
		as := OAuthClientCredentialsAuthSource{}
		err := as.Init(context.Background())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "TokenSource is nil")
	})
}

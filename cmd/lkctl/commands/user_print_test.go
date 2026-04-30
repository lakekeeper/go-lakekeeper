package commands

import (
	"bytes"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	managementv1 "github.com/lakekeeper/go-lakekeeper/pkg/apis/management/v1"
)

func TestPrintUsersText(t *testing.T) {
	t.Parallel()

	created := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	user := managementv1.User{
		Id:              "oidc~abc",
		Name:            "Alice",
		Email:           *managementv1.NewNullableString(managementv1.PtrString("alice@example.com")),
		UserType:        managementv1.USERTYPE_HUMAN,
		CreatedAt:       created,
		LastUpdatedWith: managementv1.USERLASTUPDATEDWITH_CREATE_ENDPOINT,
	}

	var buf bytes.Buffer
	require.NoError(t, printUsers(&buf, "text", nil, &user))

	got := buf.String()
	assert.Contains(t, got, "ID")
	assert.Contains(t, got, "NAME")
	assert.Contains(t, got, "oidc~abc")
	assert.Contains(t, got, "Alice")
	assert.Contains(t, got, "alice@example.com")
	assert.Contains(t, got, string(managementv1.USERTYPE_HUMAN))
}

func TestPrintUsersWide(t *testing.T) {
	t.Parallel()

	created := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	user := managementv1.User{
		Id:              "oidc~abc",
		Name:            "Alice",
		UserType:        managementv1.USERTYPE_HUMAN,
		CreatedAt:       created,
		LastUpdatedWith: managementv1.USERLASTUPDATEDWITH_CREATE_ENDPOINT,
	}

	var buf bytes.Buffer
	require.NoError(t, printUsers(&buf, "wide", nil, &user))
	assert.Contains(t, buf.String(), "CREATED AT")
	assert.Contains(t, buf.String(), "LAST UPDATED WITH")
}

func TestPrintUsersWithNextPageToken(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	tok := "next-token"
	require.NoError(t, printUsers(&buf, "text", &tok))
	assert.Contains(t, buf.String(), "next-token")
}

func TestPrintUsersUnknownFormat(t *testing.T) {
	t.Parallel()

	err := printUsers(&bytes.Buffer{}, "yaml", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown output format")
}

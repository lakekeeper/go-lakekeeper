//go:build e2e_cli

package clie2e

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUserCRUD(t *testing.T) {
	requireBackend(t, BackendCompose)
	t.Parallel()

	id := MustProvisionUser(t) // create + cleanup registered

	getOut := runOK(t, "user", "get", id, "--output", "json")
	var got struct {
		ID string `json:"id"`
	}
	decodeJSON(t, getOut, &got)
	assert.Equal(t, id, got.ID)
}

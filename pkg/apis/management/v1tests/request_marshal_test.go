package v1tests

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	managementv1 "github.com/lakekeeper/go-lakekeeper/pkg/apis/management/v1"
)

// Optional nullable request fields are NullableX structs, which `omitempty`
// cannot drop. Only the generated MarshalJSON (generateMarshalJSON, on by
// default) omits unset ones; without it every unset field goes out as an
// explicit `null`, which the server may read as "clear this field".

func TestRequestMarshalOmitsUnsetNullableFields(t *testing.T) {
	t.Parallel()

	b, err := json.Marshal(managementv1.NewUpdateUserRequest("alice", managementv1.USERTYPE_HUMAN))
	require.NoError(t, err)
	assert.JSONEq(t, `{"name":"alice","user-type":"human"}`, string(b))
}

func TestRequestMarshalKeepsExplicitNull(t *testing.T) {
	t.Parallel()

	req := managementv1.NewUpdateUserRequest("alice", managementv1.USERTYPE_HUMAN)
	req.SetEmailNil()

	b, err := json.Marshal(req)
	require.NoError(t, err)
	assert.JSONEq(t, `{"email":null,"name":"alice","user-type":"human"}`, string(b))
}

package v1tests

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	managementv1 "github.com/lakekeeper/go-lakekeeper/pkg/apis/management/v1"
)

// Models the prepare patches of the Lakekeeper+ spec type: the Cedar lists,
// the inline enums and the queue configs. Fixtures follow Cedar's JSON policy
// and entity formats.

func TestCedarPoliciesDecodeTyped(t *testing.T) {
	t.Parallel()

	const body = `{"policies": [
		{
			"effect": "permit",
			"principal": {"op": "==", "entity": {"type": "User", "id": "alice"}},
			"action": {"op": "in", "entities": [{"type": "Action", "id": "view"}]},
			"resource": {"op": "is", "entity_type": "Folder", "in": {"entity": {"type": "Folder", "id": "f"}}},
			"conditions": [{"kind": "when", "body": {"Var": "context"}}],
			"annotations": {"id": "p1", "flag": null}
		},
		{"effect": "forbid", "principal": {"op": "All"}, "action": {"op": "All"}, "resource": {"op": "All"}, "conditions": []}
	]}`

	var got managementv1.GetCedarPoliciesResponse
	require.NoError(t, json.Unmarshal([]byte(body), &got))

	require.Len(t, got.Policies, 2)
	p := got.Policies[0]
	assert.Equal(t, managementv1.POLICYEFFECT_PERMIT, p.Effect)
	assert.Equal(t, managementv1.CedarEntityUid{Type: "User", Id: "alice"}, *p.Principal.Entity)
	assert.Equal(t, []managementv1.CedarEntityUid{{Type: "Action", Id: "view"}}, p.Action.Entities)
	assert.Equal(t, "Folder", p.Resource.GetEntityType())
	assert.Equal(t, managementv1.CEDARPOLICYCONDITIONKIND_WHEN, p.Conditions[0].Kind)
	assert.Equal(t, managementv1.POLICYEFFECT_FORBID, got.Policies[1].Effect)
	assert.Equal(t, "All", got.Policies[1].Principal.Op)
}

func TestCedarEntitiesDecodeTyped(t *testing.T) {
	t.Parallel()

	const body = `{"entity-count": 1, "entities": [{
		"uid": {"type": "Lakekeeper::User", "id": "alice"},
		"attrs": {"name": "Alice", "manager": {"__entity": {"type": "Lakekeeper::User", "id": "bob"}}},
		"parents": [{"type": "Lakekeeper::Role", "id": "r1"}]
	}]}`

	var got managementv1.ResolveCedarEntitiesResponse
	require.NoError(t, json.Unmarshal([]byte(body), &got))

	require.Len(t, got.Entities, 1)
	e := got.Entities[0]
	assert.Equal(t, managementv1.CedarEntityUid{Type: "Lakekeeper::User", Id: "alice"}, e.Uid)
	assert.Equal(t, "Alice", e.Attrs["name"])
	assert.Equal(t, []managementv1.CedarEntityUid{{Type: "Lakekeeper::Role", Id: "r1"}}, e.Parents)
}

func TestCedarResolveResourceServerRoundTrip(t *testing.T) {
	t.Parallel()

	server := managementv1.CEDARRESOLVERESOURCESERVER_SERVER
	b, err := json.Marshal(managementv1.CedarResolveResourceServerAsCedarResolveResource(&server))
	require.NoError(t, err)
	assert.JSONEq(t, `"server"`, string(b))

	var got managementv1.CedarResolveResource
	require.NoError(t, json.Unmarshal(b, &got))
	assert.Equal(t, &server, got.CedarResolveResourceServer)
}

func TestOptimizeStepsMarshalAsStrings(t *testing.T) {
	t.Parallel()

	payload := managementv1.NewScheduleOptimizeTaskRequestPayload()
	payload.SetSteps([]managementv1.OptimizeStep{
		managementv1.OPTIMIZESTEP_COMPACTION,
		managementv1.OPTIMIZESTEP_REWRITE_MANIFESTS,
	})

	b, err := json.Marshal(payload)
	require.NoError(t, err)
	assert.JSONEq(t, `{"steps":["compaction","rewrite-manifests"]}`, string(b))
}

// A queue config's nullable fields are unset to inherit; the constructors must
// not set them, or every request would pin the inherited values.
func TestQueueConfigConstructorsLeaveInheritFieldsUnset(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		config  any
		inherit []string
	}{
		"expire snapshots": {
			managementv1.NewExpireSnapshotsQueueConfig(),
			[]string{"enable-expire-snapshots", "max-ref-age-ms", "max-snapshot-age-ms", "min-snapshots-to-expire", "min-snapshots-to-keep"},
		},
		"optimize": {
			managementv1.NewOptimizeQueueConfig(),
			[]string{"enable-compaction", "enable-rewrite-manifests"},
		},
		"remove orphan files": {
			managementv1.NewRemoveOrphanFilesQueueConfig(),
			nil,
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			b, err := json.Marshal(tc.config)
			require.NoError(t, err)
			var fields map[string]any
			require.NoError(t, json.Unmarshal(b, &fields))
			for _, f := range tc.inherit {
				assert.NotContains(t, fields, f)
			}
		})
	}
}

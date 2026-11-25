package main

import (
	"fmt"
	"io"
	"strings"

	"go.yaml.in/yaml/v3"
)

// A patch fixes one defect in one spec until the spec is fixed upstream. It is
// named after the defect, checks for the defect before fixing it, and is
// idempotent: once fixed, the defect is gone and the patch is skipped. A patch
// whose defect is gone upstream logs "not needed, skipped"; delete it then.
type patch struct {
	name string
	// defect reports whether the spec still has the defect.
	defect func(spec *yaml.Node) bool
	// fix removes the defect; it runs only when defect reported true.
	fix func(spec *yaml.Node)
}

// lakekeeperPatches fix defects in the Lakekeeper spec. None today.
var lakekeeperPatches = []patch{}

// lakekeeperPlusPatches fix defects in the Lakekeeper+ spec, listed in
// docs/GENERATION.md. Delete a patch once its defect is fixed upstream.
var lakekeeperPlusPatches = []patch{
	licenseIdentifier,
	booleanDefault,
	typedCedarLists,
	namedInlineEnums,
	noInheritDefaults,
}

// applyPatches runs patches on spec in order and logs one line per patch.
func applyPatches(spec *yaml.Node, patches []patch, log io.Writer) error {
	for _, p := range patches {
		if !p.defect(spec) {
			fmt.Fprintf(log, "patch %s: not needed, skipped\n", p.name)
			continue
		}
		p.fix(spec)
		if p.defect(spec) {
			return fmt.Errorf("patch %s: defect still present after the fix", p.name)
		}
		fmt.Fprintf(log, "patch %s: applied\n", p.name)
	}
	return nil
}

// licenseIdentifier: the Lakekeeper+ info.license has a name but neither an
// identifier nor a url, which OpenAPI 3.1 requires one of; openapi-generator's
// spec validation fails on it. The license is not SPDX-listed, so the
// identifier is an SPDX LicenseRef.
var licenseIdentifier = patch{
	name: "license-identifier",
	defect: func(spec *yaml.Node) bool {
		license := get(get(spec, "info"), "license")
		return license != nil && get(license, "identifier") == nil && get(license, "url") == nil
	},
	fix: func(spec *yaml.Node) {
		set(get(get(spec, "info"), "license"), "identifier", &yaml.Node{
			Kind: yaml.ScalarNode, Tag: "!!str", Value: "LicenseRef-Vakamo-Enterprise-License",
		})
	},
}

// booleanDefault: a boolean schema declares its default as the string 'false'
// (today `ExpireSnapshotsQueueConfig.enable-expire-snapshots`). The patch
// turns every such default into the boolean it spells.
var booleanDefault = patch{
	name: "boolean-default",
	defect: func(spec *yaml.Node) bool {
		return len(stringBooleanDefaults(spec)) > 0
	},
	fix: func(spec *yaml.Node) {
		for _, d := range stringBooleanDefaults(spec) {
			d.Tag, d.Style = "!!bool", 0
		}
	},
}

// stringBooleanDefaults returns the defaults anywhere in spec that belong to a
// boolean schema but are strings spelling a boolean.
func stringBooleanDefaults(spec *yaml.Node) []*yaml.Node {
	var found []*yaml.Node
	walk(spec, func(n *yaml.Node) {
		d := get(n, "default")
		if d == nil || d.Kind != yaml.ScalarNode || d.Tag != "!!str" || (d.Value != "true" && d.Value != "false") {
			return
		}
		if hasType(n, "boolean") {
			found = append(found, d)
		}
	})
	return found
}

// noInheritDefaults: in the queue-config schemas, a nullable field is unset to
// inherit the server's value, yet declares that value as its `default`; the
// generated constructors then set every such field and send it explicitly. The
// patch drops `default` from the nullable properties of every schema named
// `*QueueConfig`.
var noInheritDefaults = patch{
	name: "no-inherit-defaults",
	defect: func(spec *yaml.Node) bool {
		return len(inheritDefaults(spec)) > 0
	},
	fix: func(spec *yaml.Node) {
		for _, prop := range inheritDefaults(spec) {
			del(prop, "default")
		}
	},
}

// inheritDefaults returns the nullable properties of the *QueueConfig schemas
// that declare a default.
func inheritDefaults(spec *yaml.Node) []*yaml.Node {
	var found []*yaml.Node
	schemas := get(get(spec, "components"), "schemas")
	if schemas == nil {
		return nil
	}
	for i := 0; i < len(schemas.Content); i += 2 {
		if !strings.HasSuffix(schemas.Content[i].Value, "QueueConfig") {
			continue
		}
		props := get(schemas.Content[i+1], "properties")
		if props == nil {
			continue
		}
		for j := 1; j < len(props.Content); j += 2 {
			prop := props.Content[j]
			if hasType(prop, "null") && get(prop, "default") != nil {
				found = append(found, prop)
			}
		}
	}
	return found
}

// namedInlineEnums: two string enums are declared inline, where the generator
// drops them to plain strings without constants: the `server` variant of the
// `CedarResolveResource` union, and the items of the optimize task's
// `payload.steps`. The patch moves each to a named schema and refers to it.
var namedInlineEnums = patch{
	name: "named-inline-enums",
	defect: func(spec *yaml.Node) bool {
		for _, e := range inlineEnums {
			if s := e.slot(spec); s != nil && isInlineEnum(*s) {
				return true
			}
		}
		return false
	},
	fix: func(spec *yaml.Node) {
		schemas := mapping(mapping(spec, "components"), "schemas")
		for _, e := range inlineEnums {
			s := e.slot(spec)
			if s == nil || !isInlineEnum(*s) {
				continue
			}
			if get(schemas, e.name) == nil {
				set(schemas, e.name, *s)
			}
			*s = ref(e.name)
		}
	},
}

// inlineEnums are the inline enums namedInlineEnums names. slot returns the
// place in the spec that holds the enum, or nil when the spec has none.
var inlineEnums = []struct {
	name string
	slot func(spec *yaml.Node) **yaml.Node
}{
	{"CedarResolveResourceServer", func(spec *yaml.Node) **yaml.Node {
		variants := get(schema(spec, "CedarResolveResource"), "oneOf")
		if variants == nil {
			return nil
		}
		for i, v := range variants.Content {
			if hasType(v, "string") && get(v, "enum") != nil {
				return &variants.Content[i]
			}
		}
		return nil
	}},
	{"OptimizeStep", func(spec *yaml.Node) **yaml.Node {
		payload := get(get(schema(spec, "ScheduleOptimizeTaskRequest"), "properties"), "payload")
		return slot(get(get(payload, "properties"), "steps"), "items")
	}},
}

// typedCedarLists: the Cedar policies list (`GetCedarPoliciesResponse.policies`)
// and the resolved entities list (`ResolveCedarEntitiesResponse.entities`)
// declare `items: {}`, so they generate as []interface{}. Their descriptions
// name Cedar's JSON formats for policies and entities; the patch adds item
// schemas with that shape (cedarSchemas) and refers to them. The parts of a
// policy that are open-ended in Cedar's format (condition bodies, the `in` of
// an `is` constraint) and entity attributes and tags stay free-form objects.
var typedCedarLists = patch{
	name: "typed-cedar-lists",
	defect: func(spec *yaml.Node) bool {
		for _, l := range cedarLists {
			if s := l.slot(spec); s != nil && isEmptySchema(*s) {
				return true
			}
		}
		return false
	},
	fix: func(spec *yaml.Node) {
		schemas := mapping(mapping(spec, "components"), "schemas")
		var named yaml.Node
		if err := yaml.Unmarshal([]byte(cedarSchemas), &named); err != nil {
			panic(fmt.Sprintf("cedarSchemas: %v", err))
		}
		for _, l := range cedarLists {
			s := l.slot(spec)
			if s == nil || !isEmptySchema(*s) {
				continue
			}
			for _, name := range l.schemas {
				if get(schemas, name) == nil {
					set(schemas, name, get(named.Content[0], name))
				}
			}
			*s = ref(l.schemas[0])
		}
	},
}

// cedarLists are the lists typedCedarLists types: where their items are, and
// the schemas the items need, the item schema first.
var cedarLists = []struct {
	slot    func(spec *yaml.Node) **yaml.Node
	schemas []string
}{
	{
		func(spec *yaml.Node) **yaml.Node {
			return slot(get(get(schema(spec, "GetCedarPoliciesResponse"), "properties"), "policies"), "items")
		},
		[]string{"CedarPolicy", "CedarPolicyScopeConstraint", "CedarPolicyCondition", "CedarPolicyConditionKind", "CedarEntityUid"},
	},
	{
		func(spec *yaml.Node) **yaml.Node {
			return slot(get(get(schema(spec, "ResolveCedarEntitiesResponse"), "properties"), "entities"), "items")
		},
		[]string{"CedarEntity", "CedarEntityUid"},
	},
}

// cedarSchemas are the item schemas of the Cedar lists, after Cedar's JSON
// policy format (https://docs.cedarpolicy.com/policies/json-format.html) and
// entity format (https://docs.cedarpolicy.com/auth/entities-syntax.html).
const cedarSchemas = `
CedarPolicy:
  type: object
  description: A Cedar policy in Cedar's JSON policy format.
  required:
    - effect
    - principal
    - action
    - resource
  properties:
    effect:
      $ref: '#/components/schemas/PolicyEffect'
    principal:
      $ref: '#/components/schemas/CedarPolicyScopeConstraint'
    action:
      $ref: '#/components/schemas/CedarPolicyScopeConstraint'
    resource:
      $ref: '#/components/schemas/CedarPolicyScopeConstraint'
    conditions:
      type: array
      description: The policy's when and unless clauses, in order.
      items:
        $ref: '#/components/schemas/CedarPolicyCondition'
    annotations:
      type: object
      description: The policy's annotations, by key; an annotation without a value is null.
      additionalProperties:
        type:
          - string
          - 'null'
CedarPolicyScopeConstraint:
  type: object
  description: The constraint a Cedar policy puts on its principal, action or resource.
  required:
    - op
  properties:
    op:
      type: string
      description: 'The operator: All, ==, in or is.'
    entity:
      $ref: '#/components/schemas/CedarEntityUid'
    entities:
      type: array
      description: The actions an action constraint with op in names.
      items:
        $ref: '#/components/schemas/CedarEntityUid'
    entity_type:
      type: string
      description: The entity type an is constraint names.
    slot:
      type: string
      description: The template slot the constraint names, e.g. ?principal.
    in:
      type: object
      description: The in part of an is constraint, with an entity or a slot.
CedarPolicyCondition:
  type: object
  description: A when or unless clause of a Cedar policy.
  required:
    - kind
    - body
  properties:
    kind:
      $ref: '#/components/schemas/CedarPolicyConditionKind'
    body:
      type: object
      description: The clause's expression, in Cedar's JSON expression format.
CedarPolicyConditionKind:
  type: string
  description: Whether a condition is a when or an unless clause.
  enum:
    - when
    - unless
CedarEntity:
  type: object
  description: A Cedar entity in Cedar's JSON entity format.
  required:
    - uid
    - attrs
    - parents
  properties:
    uid:
      $ref: '#/components/schemas/CedarEntityUid'
    attrs:
      type: object
      description: The entity's attributes, by name, as Cedar JSON values.
    parents:
      type: array
      description: The entities this entity is a member of.
      items:
        $ref: '#/components/schemas/CedarEntityUid'
    tags:
      type: object
      description: The entity's tags, by name, as Cedar JSON values.
CedarEntityUid:
  type: object
  description: A Cedar entity's type and id.
  required:
    - type
    - id
  properties:
    type:
      type: string
    id:
      type: string
`

// isEmptySchema reports whether n is the schema {}, which allows any value.
func isEmptySchema(n *yaml.Node) bool {
	return n.Kind == yaml.MappingNode && len(n.Content) == 0
}

// isInlineEnum reports whether n is an enum schema written in place, not a $ref.
func isInlineEnum(n *yaml.Node) bool {
	return get(n, "enum") != nil && get(n, "$ref") == nil
}

// schema returns the component schema name in spec, or nil.
func schema(spec *yaml.Node, name string) *yaml.Node {
	return get(get(get(spec, "components"), "schemas"), name)
}

// ref returns a $ref to the component schema name.
func ref(name string) *yaml.Node {
	return &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map", Content: []*yaml.Node{
		{Kind: yaml.ScalarNode, Tag: "!!str", Value: "$ref"},
		{Kind: yaml.ScalarNode, Tag: "!!str", Value: "#/components/schemas/" + name, Style: yaml.SingleQuotedStyle},
	}}
}

// slot returns the place in mapping node m that holds the value of key, or nil.
func slot(m *yaml.Node, key string) **yaml.Node {
	if m == nil || m.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i < len(m.Content); i += 2 {
		if m.Content[i].Value == key {
			return &m.Content[i+1]
		}
	}
	return nil
}

// del removes key from mapping node m.
func del(m *yaml.Node, key string) {
	for i := 0; i < len(m.Content); i += 2 {
		if m.Content[i].Value == key {
			m.Content = append(m.Content[:i], m.Content[i+2:]...)
			return
		}
	}
}

// hasType reports whether schema n has type t, alone or in a type list.
func hasType(n *yaml.Node, t string) bool {
	typ := get(n, "type")
	if typ == nil {
		return false
	}
	if typ.Kind == yaml.ScalarNode {
		return typ.Value == t
	}
	for _, c := range typ.Content {
		if c.Value == t {
			return true
		}
	}
	return false
}

// walk calls f on n and every mapping node below it.
func walk(n *yaml.Node, f func(*yaml.Node)) {
	if n == nil {
		return
	}
	if n.Kind == yaml.MappingNode {
		f(n)
	}
	for _, c := range n.Content {
		walk(c, f)
	}
}

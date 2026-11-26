package main

import (
	"fmt"
	"reflect"
	"sort"
	"strings"

	"go.yaml.in/yaml/v3"
)

// merge adds to base (the Lakekeeper spec) what plus (the Lakekeeper+ spec)
// has and base lacks. base is modified in place.
func merge(base, plus *yaml.Node) error {
	if err := mergePaths(mapping(base, "paths"), mapping(plus, "paths")); err != nil {
		return err
	}
	if err := mergeComponents(mapping(base, "components"), mapping(plus, "components")); err != nil {
		return err
	}
	mergeTags(base, plus)
	return nil
}

// mergePaths adds the Lakekeeper+ paths and operations Lakekeeper lacks. An
// added operation must not reuse an operationId of another operation.
func mergePaths(base, plus *yaml.Node) error {
	owners := operationIDs(base)
	for i := 0; i < len(plus.Content); i += 2 {
		path, plusItem := plus.Content[i].Value, plus.Content[i+1]
		baseItem := get(base, path)
		if baseItem == nil {
			baseItem = &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
			set(base, path, baseItem)
		}
		for j := 0; j < len(plusItem.Content); j += 2 {
			key, plusOp := plusItem.Content[j].Value, plusItem.Content[j+1]
			if baseOp := get(baseItem, key); baseOp != nil {
				if !sameOperation(baseOp, plusOp) {
					return fmt.Errorf("operation %s %s differs between the Lakekeeper and Lakekeeper+ specs in a way the merge cannot take: only descriptions and extra documented response codes may differ", key, path)
				}
				continue
			}
			if id := get(plusOp, "operationId"); id != nil {
				here := key + " " + path
				if owner, taken := owners[id.Value]; taken {
					return fmt.Errorf("operationId %s of Lakekeeper+ %s is already used by %s", id.Value, here, owner)
				}
				owners[id.Value] = here
			}
			set(baseItem, key, plusOp)
		}
	}
	return nil
}

// operationIDs maps each operationId in paths to "<method> <path>".
func operationIDs(paths *yaml.Node) map[string]string {
	ids := map[string]string{}
	for i := 0; i < len(paths.Content); i += 2 {
		item := paths.Content[i+1]
		for j := 0; j < len(item.Content); j += 2 {
			if id := get(item.Content[j+1], "operationId"); id != nil {
				ids[id.Value] = item.Content[j].Value + " " + paths.Content[i].Value
			}
		}
	}
	return ids
}

func mergeComponents(base, plus *yaml.Node) error {
	if plus == nil {
		return nil
	}
	for i := 0; i < len(plus.Content); i += 2 {
		section, plusSection := plus.Content[i].Value, plus.Content[i+1]
		if section == "securitySchemes" {
			continue // security schemes come from Lakekeeper
		}
		baseSection := get(base, section)
		if baseSection == nil {
			set(base, section, plusSection)
			continue
		}
		for j := 0; j < len(plusSection.Content); j += 2 {
			name, plusItem := plusSection.Content[j].Value, plusSection.Content[j+1]
			baseItem := get(baseSection, name)
			if baseItem == nil {
				set(baseSection, name, plusItem)
				continue
			}
			if section == "schemas" {
				addUnionVariants(baseItem, plusItem)
			}
			if !sameShape(baseItem, plusItem) {
				return fmt.Errorf("%s %s differs between the Lakekeeper and Lakekeeper+ specs in a way the merge cannot take: only descriptions, maxLength and new union variants may differ", singular(section), name)
			}
		}
	}
	return nil
}

func singular(section string) string {
	return strings.TrimSuffix(section, "s")
}

// descriptiveKeys are schema and operation keywords whose differences do not
// change the generated code's behaviour; the Lakekeeper side is kept.
var descriptiveKeys = map[string]bool{
	"description":  true,
	"summary":      true,
	"title":        true,
	"example":      true,
	"examples":     true,
	"externalDocs": true,
	"maxLength":    true,
}

// sameShape reports whether a and b are equal once descriptive keywords are
// ignored and oneOf entries are compared as a set.
func sameShape(a, b *yaml.Node) bool {
	return reflect.DeepEqual(shape(a), shape(b))
}

// sameOperation is sameShape for two definitions of one path item entry (an
// operation, or the path-level parameters), where response codes that only
// plus documents are ignored.
func sameOperation(base, plus *yaml.Node) bool {
	a, b := shape(base), shape(plus)
	aMap, aOK := a.(map[string]any)
	bMap, bOK := b.(map[string]any)
	if aOK && bOK {
		aResp, _ := aMap["responses"].(map[string]any)
		bResp, _ := bMap["responses"].(map[string]any)
		for code := range bResp {
			if _, ok := aResp[code]; !ok {
				delete(bResp, code)
			}
		}
	}
	return reflect.DeepEqual(a, b)
}

func shape(n *yaml.Node) any {
	var v any
	if err := n.Decode(&v); err != nil {
		panic(err) // the node came from a successful parse
	}
	return normalize(v, false)
}

// normalize drops descriptive keywords and sorts oneOf entries. When names is
// true, v's keys are names (properties, mapping), not keywords, and are kept.
func normalize(v any, names bool) any {
	switch t := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(t))
		for k, child := range t {
			if !names && descriptiveKeys[k] {
				continue
			}
			out[k] = normalize(child, !names && (k == "properties" || k == "patternProperties"))
		}
		if oneOf, ok := out["oneOf"].([]any); ok && !names {
			sort.Slice(oneOf, func(i, j int) bool { return fmt.Sprint(oneOf[i]) < fmt.Sprint(oneOf[j]) })
		}
		return out
	case []any:
		out := make([]any, len(t))
		for i, child := range t {
			out[i] = normalize(child, false)
		}
		return out
	default:
		return v
	}
}

// addUnionVariants adds to a discriminated union in base the variants (oneOf
// entry and discriminator mapping) that the same union in plus has and base
// lacks.
func addUnionVariants(base, plus *yaml.Node) {
	baseOneOf, plusOneOf := get(base, "oneOf"), get(plus, "oneOf")
	baseMapping := get(get(base, "discriminator"), "mapping")
	plusMapping := get(get(plus, "discriminator"), "mapping")
	if baseOneOf == nil || plusOneOf == nil || baseMapping == nil || plusMapping == nil {
		return
	}
	refs := map[string]bool{}
	for _, v := range baseOneOf.Content {
		refs[refOf(v)] = true
	}
	for _, v := range plusOneOf.Content {
		if !refs[refOf(v)] {
			baseOneOf.Content = append(baseOneOf.Content, v)
		}
	}
	for i := 0; i < len(plusMapping.Content); i += 2 {
		if get(baseMapping, plusMapping.Content[i].Value) == nil {
			baseMapping.Content = append(baseMapping.Content, plusMapping.Content[i], plusMapping.Content[i+1])
		}
	}
}

func refOf(n *yaml.Node) string {
	if r := get(n, "$ref"); r != nil {
		return r.Value
	}
	return ""
}

// mergeTags appends the Lakekeeper+ tags Lakekeeper does not declare.
func mergeTags(base, plus *yaml.Node) {
	plusTags := get(plus, "tags")
	if plusTags == nil {
		return
	}
	baseTags := get(base, "tags")
	if baseTags == nil {
		baseTags = &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
		set(base, "tags", baseTags)
	}
	known := map[string]bool{}
	for _, tag := range baseTags.Content {
		if name := get(tag, "name"); name != nil {
			known[name.Value] = true
		}
	}
	for _, tag := range plusTags.Content {
		if name := get(tag, "name"); name != nil && !known[name.Value] {
			baseTags.Content = append(baseTags.Content, tag)
		}
	}
}

// get returns the value of key in mapping node m, or nil.
func get(m *yaml.Node, key string) *yaml.Node {
	if m == nil || m.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i < len(m.Content); i += 2 {
		if m.Content[i].Value == key {
			return m.Content[i+1]
		}
	}
	return nil
}

// set replaces the value of key in mapping node m, or appends the pair.
func set(m *yaml.Node, key string, value *yaml.Node) {
	for i := 0; i < len(m.Content); i += 2 {
		if m.Content[i].Value == key {
			m.Content[i+1] = value
			return
		}
	}
	m.Content = append(m.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key}, value)
}

// mapping returns the mapping at key in m, creating an empty one in m when the
// key is absent.
func mapping(m *yaml.Node, key string) *yaml.Node {
	if v := get(m, key); v != nil {
		return v
	}
	v := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	set(m, key, v)
	return v
}

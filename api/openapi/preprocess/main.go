// Command preprocess rewrites the Lakekeeper management OpenAPI spec to work
// around an openapi-generator limitation: it cannot represent
// `allOf(oneOf, extra-properties)` and collapses such constructs into a
// single struct holding the union of all variant fields. This produces both
// a marshal-side bug (irrelevant fields ship as zero values) and an
// unmarshal-side bug (correct payloads are rejected for missing fields they
// shouldn't contain).
//
// The preprocessor walks `components.schemas` and:
//
//  1. For every `oneOf` member shaped as `allOf([{$ref: X}, extras])` where
//     `X` is itself a `oneOf`, replaces that single member with the
//     cartesian product of `X`'s inner members combined with `extras`.
//  2. For every modified `oneOf`, extracts each (now anonymous) member into
//     a new named schema and replaces the inline definition with a `$ref`.
//     Names are synthesized as `<ParentName><PascalCase(discriminator-value)>`.
//  3. Adds a `discriminator` block where a single property has unique enum
//     values across all members. Combined with `useOneOfDiscriminatorLookup`,
//     this lets the generator emit unambiguous unmarshal logic — important
//     when two leaf shapes are otherwise identical (e.g. Az managed-identity
//     and GCS system-identity, both `{type, credential-type}`).
//
// This transformation does not change any wire-format payload. It only
// changes how the spec is expressed to the generator.
package main

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

const refPrefix = "#/components/schemas/"

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: preprocess <input> <output>")
		os.Exit(2)
	}
	if err := run(os.Args[1], os.Args[2]); err != nil {
		fmt.Fprintf(os.Stderr, "preprocess: %v\n", err)
		os.Exit(1)
	}
}

func run(inPath, outPath string) error {
	data, err := os.ReadFile(inPath)
	if err != nil {
		return fmt.Errorf("read input: %w", err)
	}

	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return fmt.Errorf("parse yaml: %w", err)
	}

	root := docRoot(&doc)
	schemas := lookup(lookup(root, "components"), "schemas")
	if schemas == nil || schemas.Kind != yaml.MappingNode {
		return errors.New("components.schemas not found")
	}

	// For each named schema with a oneOf, attempt the full transformation
	// (expand → extract → discriminate). If discrimination fails (e.g. no
	// single property has unique enum values across all expanded members),
	// roll back so the schema is left unchanged. This means we only rewrite
	// schemas where the new shape is provably better than the original.
	transformed := 0
	for _, name := range schemaNames(schemas) {
		schema := lookupSchema(schemas, name)
		if schema == nil {
			continue
		}
		oneOf := lookup(schema, "oneOf")
		if oneOf == nil || oneOf.Kind != yaml.SequenceNode {
			continue
		}
		original := append([]*yaml.Node(nil), oneOf.Content...)
		schemasSnapshot := append([]*yaml.Node(nil), schemas.Content...)

		if !expandOneOf(schema, schemas) {
			continue
		}
		if err := extractAndDiscriminate(name, schema, schemas); err != nil {
			oneOf.Content = original
			schemas.Content = schemasSnapshot
			fmt.Fprintf(os.Stderr, "preprocess: skipped %s (%v)\n", name, err)
			continue
		}
		transformed++
	}

	out, err := os.Create(outPath)
	if err != nil {
		return fmt.Errorf("create output: %w", err)
	}
	defer out.Close()

	enc := yaml.NewEncoder(out)
	enc.SetIndent(2)
	if err := enc.Encode(&doc); err != nil {
		return fmt.Errorf("write yaml: %w", err)
	}
	if err := enc.Close(); err != nil {
		return fmt.Errorf("close encoder: %w", err)
	}

	fmt.Fprintf(os.Stderr, "preprocess: transformed %d schema(s)\n", transformed)
	return nil
}

func docRoot(n *yaml.Node) *yaml.Node {
	if n.Kind == yaml.DocumentNode && len(n.Content) > 0 {
		return n.Content[0]
	}
	return n
}

func schemaNames(schemas *yaml.Node) []string {
	names := make([]string, 0, len(schemas.Content)/2)
	for i := 0; i+1 < len(schemas.Content); i += 2 {
		names = append(names, schemas.Content[i].Value)
	}
	return names
}

func lookupSchema(schemas *yaml.Node, name string) *yaml.Node {
	for i := 0; i+1 < len(schemas.Content); i += 2 {
		if schemas.Content[i].Value == name {
			return schemas.Content[i+1]
		}
	}
	return nil
}

// expandOneOf rewrites `schema.oneOf` members that are
// `allOf([{$ref: X}, extras])` where `X.oneOf` exists, into the cartesian
// product of `X`'s inner members combined with `extras`. Returns true if
// any member was expanded.
func expandOneOf(schema, schemas *yaml.Node) bool {
	if schema == nil || schema.Kind != yaml.MappingNode {
		return false
	}
	oneOf := lookup(schema, "oneOf")
	if oneOf == nil || oneOf.Kind != yaml.SequenceNode {
		return false
	}

	expanded := false
	var newMembers []*yaml.Node
	for _, member := range oneOf.Content {
		replacement, didExpand := expandMember(member, schemas)
		if didExpand {
			expanded = true
		}
		newMembers = append(newMembers, replacement...)
	}
	oneOf.Content = newMembers
	return expanded
}

func expandMember(member, schemas *yaml.Node) ([]*yaml.Node, bool) {
	allOf := lookup(member, "allOf")
	if allOf == nil || allOf.Kind != yaml.SequenceNode || len(allOf.Content) == 0 {
		return []*yaml.Node{member}, false
	}
	first := allOf.Content[0]
	refNode := lookup(first, "$ref")
	if refNode == nil {
		return []*yaml.Node{member}, false
	}
	refName, ok := refNameFromPointer(refNode.Value)
	if !ok {
		return []*yaml.Node{member}, false
	}
	target := lookupSchema(schemas, refName)
	if target == nil {
		return []*yaml.Node{member}, false
	}
	innerOneOf := lookup(target, "oneOf")
	if innerOneOf == nil || innerOneOf.Kind != yaml.SequenceNode {
		return []*yaml.Node{member}, false
	}

	extras := allOf.Content[1:]
	expanded := make([]*yaml.Node, 0, len(innerOneOf.Content))
	for _, inner := range innerOneOf.Content {
		expanded = append(expanded, buildAllOf(inner, extras))
	}
	return expanded, true
}

func buildAllOf(inner *yaml.Node, extras []*yaml.Node) *yaml.Node {
	var elements []*yaml.Node
	if innerAllOf := lookup(inner, "allOf"); innerAllOf != nil && innerAllOf.Kind == yaml.SequenceNode {
		elements = append(elements, innerAllOf.Content...)
	} else {
		elements = append(elements, inner)
	}
	elements = append(elements, extras...)

	return &yaml.Node{
		Kind: yaml.MappingNode,
		Content: []*yaml.Node{
			{Kind: yaml.ScalarNode, Value: "allOf"},
			{Kind: yaml.SequenceNode, Content: elements},
		},
	}
}

// extractAndDiscriminate moves each anonymous `oneOf` member into a new
// top-level schema, replaces it with a `$ref`, and adds a `discriminator`
// block to the parent if a viable property is found.
func extractAndDiscriminate(parentName string, parent, schemas *yaml.Node) error {
	oneOf := lookup(parent, "oneOf")
	if oneOf == nil || oneOf.Kind != yaml.SequenceNode {
		return nil
	}

	// Discriminator candidates: property name -> per-member single-enum value.
	// A property is a viable discriminator if every member has it set to a
	// distinct single-value enum.
	infos := make([]memberInfo, len(oneOf.Content))
	for i, m := range oneOf.Content {
		infos[i] = memberInfo{props: collectSingleEnums(m)}
	}

	discProp := pickDiscriminator(infos)
	if discProp == "" {
		return errors.New("no viable discriminator property")
	}

	// Build new named schemas and ref-replacements.
	mapping := make(map[string]string)
	newMembers := make([]*yaml.Node, len(oneOf.Content))
	for i, m := range oneOf.Content {
		discValue := infos[i].props[discProp]
		schemaName := parentName + toPascalCase(discValue)
		if lookupSchema(schemas, schemaName) != nil {
			return fmt.Errorf("synthesized name %q already exists in components.schemas", schemaName)
		}
		appendSchema(schemas, schemaName, m)
		ref := &yaml.Node{
			Kind: yaml.MappingNode,
			Content: []*yaml.Node{
				{Kind: yaml.ScalarNode, Value: "$ref"},
				{Kind: yaml.ScalarNode, Value: refPrefix + schemaName},
			},
		}
		newMembers[i] = ref
		mapping[discValue] = refPrefix + schemaName
	}
	oneOf.Content = newMembers

	setDiscriminator(parent, discProp, mapping)
	return nil
}

// collectSingleEnums walks a oneOf member (possibly an allOf chain) and
// gathers any property whose schema is `type: string, enum: [SINGLE]` into a
// map of propertyName -> singleEnumValue.
func collectSingleEnums(member *yaml.Node) map[string]string {
	out := make(map[string]string)
	if member == nil || member.Kind != yaml.MappingNode {
		return out
	}
	if allOf := lookup(member, "allOf"); allOf != nil && allOf.Kind == yaml.SequenceNode {
		for _, sub := range allOf.Content {
			for k, v := range collectSingleEnums(sub) {
				out[k] = v
			}
		}
	}
	if props := lookup(member, "properties"); props != nil && props.Kind == yaml.MappingNode {
		for i := 0; i+1 < len(props.Content); i += 2 {
			propName := props.Content[i].Value
			propSchema := props.Content[i+1]
			enumNode := lookup(propSchema, "enum")
			if enumNode == nil || enumNode.Kind != yaml.SequenceNode {
				continue
			}
			if len(enumNode.Content) != 1 {
				continue
			}
			val := enumNode.Content[0]
			if val.Kind != yaml.ScalarNode {
				continue
			}
			out[propName] = val.Value
		}
	}
	return out
}

// pickDiscriminator returns the property name common to every member that
// also has a unique single-enum value across the set, or "" if none exists.
// Preference: property with the most unique values; ties broken by name.
func pickDiscriminator(infos []memberInfo) string {
	if len(infos) == 0 {
		return ""
	}
	candidates := make(map[string]bool)
	for k := range infos[0].props {
		candidates[k] = true
	}
	for _, info := range infos[1:] {
		for k := range candidates {
			if _, ok := info.props[k]; !ok {
				delete(candidates, k)
			}
		}
	}

	type score struct {
		name   string
		unique int
	}
	var scored []score
	for k := range candidates {
		seen := make(map[string]bool)
		dup := false
		for _, info := range infos {
			v := info.props[k]
			if seen[v] {
				dup = true
				break
			}
			seen[v] = true
		}
		if !dup {
			scored = append(scored, score{name: k, unique: len(seen)})
		}
	}
	if len(scored) == 0 {
		return ""
	}
	sort.Slice(scored, func(i, j int) bool {
		if scored[i].unique != scored[j].unique {
			return scored[i].unique > scored[j].unique
		}
		return scored[i].name < scored[j].name
	})
	return scored[0].name
}

type memberInfo struct {
	props map[string]string
}

// appendSchema adds a new entry `name: schema` to the schemas mapping.
func appendSchema(schemas *yaml.Node, name string, schema *yaml.Node) {
	schemas.Content = append(schemas.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Value: name},
		schema,
	)
}

// setDiscriminator adds (or replaces) a discriminator block on parent.
func setDiscriminator(parent *yaml.Node, propertyName string, mapping map[string]string) {
	if parent.Kind != yaml.MappingNode {
		return
	}

	// Build mapping node with deterministic key order.
	keys := make([]string, 0, len(mapping))
	for k := range mapping {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	mappingContent := make([]*yaml.Node, 0, 2*len(keys))
	for _, k := range keys {
		mappingContent = append(mappingContent,
			&yaml.Node{Kind: yaml.ScalarNode, Value: k},
			&yaml.Node{Kind: yaml.ScalarNode, Value: mapping[k]},
		)
	}

	disc := &yaml.Node{
		Kind: yaml.MappingNode,
		Content: []*yaml.Node{
			{Kind: yaml.ScalarNode, Value: "propertyName"},
			{Kind: yaml.ScalarNode, Value: propertyName},
			{Kind: yaml.ScalarNode, Value: "mapping"},
			{Kind: yaml.MappingNode, Content: mappingContent},
		},
	}

	for i := 0; i+1 < len(parent.Content); i += 2 {
		if parent.Content[i].Value == "discriminator" {
			parent.Content[i+1] = disc
			return
		}
	}
	parent.Content = append(parent.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Value: "discriminator"},
		disc,
	)
}

// toPascalCase converts e.g. "access-key" -> "AccessKey",
// "aws-system-identity" -> "AwsSystemIdentity".
func toPascalCase(s string) string {
	parts := strings.FieldsFunc(s, func(r rune) bool { return r == '-' || r == '_' || r == ' ' })
	var b strings.Builder
	for _, p := range parts {
		if p == "" {
			continue
		}
		b.WriteString(strings.ToUpper(p[:1]))
		if len(p) > 1 {
			b.WriteString(p[1:])
		}
	}
	return b.String()
}

func lookup(n *yaml.Node, key string) *yaml.Node {
	if n == nil || n.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(n.Content); i += 2 {
		k := n.Content[i]
		if k.Kind == yaml.ScalarNode && k.Value == key {
			return n.Content[i+1]
		}
	}
	return nil
}

func refNameFromPointer(ptr string) (string, bool) {
	if !strings.HasPrefix(ptr, refPrefix) {
		return "", false
	}
	return strings.TrimPrefix(ptr, refPrefix), true
}

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
//  0. Extracts presence-discriminated inner oneOfs (e.g. `UserOrRole`,
//     whose members are `{required: [user]}` xor `{required: [role]}` with
//     no enum) into named leaf schemas. Detection is purely structural so
//     the rule is generic — any schema where every member is an object
//     with exactly one required field name and no single-value enum is
//     a candidate.
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
//  4. Hierarchical-expansion fallback: when no single property has unique
//     enum values (e.g. `*Assignment` schemas, where the outer enum repeats
//     across user-variant and role-variant members) the preprocessor groups
//     members by the shared enum and emits a middle-schema layer per group.
//     The inner level relies on disjoint required-field sets — exactly what
//     Phase 0 guarantees by extracting the presence-discriminated inner.
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

	// Phase 0: extract presence-discriminated inner variants (e.g. UserOrRole)
	// into named leaf schemas so cartesian expansion produces clean $ref
	// members instead of duplicated inline shapes.
	if err := extractPresenceVariants(schemas); err != nil {
		return fmt.Errorf("extract presence variants: %w", err)
	}

	// Phase 1: drop fields from `required:` lists where Lakekeeper retains a
	// deprecated alias of a renamed field, so the SDK accepts older servers
	// that emit only the original name. See loosenRequiredFields for the
	// table of (schema, field) pairs.
	loosenRequired(schemas)

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

// loosenRequiredFields lists (schema, field) pairs to remove from `required:`
// arrays. Used when Lakekeeper retains a deprecated alias of a renamed field
// but the SDK must accept older servers that emit only the original name.
var loosenRequiredFields = []struct{ schema, field string }{
	{"ServerInfo", "lakekeeper-version"}, // v0.10.4 sends only "version"
}

// loosenRequired removes the configured fields from each schema's `required:`
// list. Idempotent: missing schemas or already-absent fields are no-ops, so
// re-runs and partial spec updates stay safe.
func loosenRequired(schemas *yaml.Node) {
	for _, m := range loosenRequiredFields {
		s := lookupSchema(schemas, m.schema)
		if s == nil {
			continue
		}
		req := lookup(s, "required")
		if req == nil || req.Kind != yaml.SequenceNode {
			continue
		}
		kept := req.Content[:0]
		for _, n := range req.Content {
			if n.Value != m.field {
				kept = append(kept, n)
			}
		}
		req.Content = kept
	}
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
		return extractHierarchical(parentName, parent, schemas, infos)
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

// refNode constructs a `{$ref: '#/components/schemas/<name>'}` mapping.
func refNode(name string) *yaml.Node {
	return &yaml.Node{
		Kind: yaml.MappingNode,
		Content: []*yaml.Node{
			{Kind: yaml.ScalarNode, Value: "$ref"},
			{Kind: yaml.ScalarNode, Value: refPrefix + name},
		},
	}
}

// extractPresenceVariants walks `components.schemas` and, for every named
// `oneOf` whose members are presence-discriminated objects (each member is
// `{required: [<single-prop>]}` with no single-value enum on any property),
// extracts each member into a named leaf and replaces the inline definition
// with a `$ref`. The leaf name is `<parent><PascalCase(requiredProp)>`.
//
// The detection rule is purely structural — no schema names are hardcoded.
// Any future inner-oneOf with the same shape is picked up automatically.
func extractPresenceVariants(schemas *yaml.Node) error {
	// schemaNames returns a snapshot; newly-appended leaf schemas are
	// intentionally not revisited in this loop.
	for _, name := range schemaNames(schemas) {
		schema := lookupSchema(schemas, name)
		if schema == nil {
			continue
		}
		oneOf := lookup(schema, "oneOf")
		if oneOf == nil || oneOf.Kind != yaml.SequenceNode {
			continue
		}
		if !isPresenceDiscriminated(oneOf.Content) {
			continue
		}
		newMembers := make([]*yaml.Node, 0, len(oneOf.Content))
		for _, member := range oneOf.Content {
			req := getSingleRequired(member)
			leafName := name + toPascalCase(req)
			if lookupSchema(schemas, leafName) != nil {
				return fmt.Errorf("synthesized presence leaf %q already exists in components.schemas", leafName)
			}
			appendSchema(schemas, leafName, member)
			newMembers = append(newMembers, refNode(leafName))
		}
		oneOf.Content = newMembers
	}
	return nil
}

// isPresenceDiscriminated reports whether every member of a oneOf is a plain
// object with exactly one `required` field name and no single-value enum
// constraint on any property — i.e. variants distinguished only by which
// field is set.
func isPresenceDiscriminated(members []*yaml.Node) bool {
	if len(members) < 2 {
		return false
	}
	seenReq := make(map[string]bool)
	for _, m := range members {
		if m == nil || m.Kind != yaml.MappingNode {
			return false
		}
		if lookup(m, "oneOf") != nil || lookup(m, "allOf") != nil || lookup(m, "$ref") != nil {
			return false
		}
		if len(collectSingleEnums(m)) > 0 {
			return false
		}
		req := getSingleRequired(m)
		if req == "" {
			return false
		}
		if seenReq[req] {
			return false
		}
		seenReq[req] = true
	}
	return true
}

// getSingleRequired returns the sole required-field name of an object schema,
// or "" if the schema has zero or multiple required fields.
func getSingleRequired(n *yaml.Node) string {
	req := lookup(n, "required")
	if req == nil || req.Kind != yaml.SequenceNode || len(req.Content) != 1 {
		return ""
	}
	v := req.Content[0]
	if v.Kind != yaml.ScalarNode {
		return ""
	}
	return v.Value
}

// extractHierarchical handles the case where flat single-property
// discrimination is impossible because the outer enum repeats across members
// (e.g. *Assignment schemas after cartesian expansion: `type=ownership` for
// both the user-variant and the role-variant). It groups members by the
// shared enum value, builds one middle schema per group whose `oneOf`
// trial-and-error matches the disjoint required-sets of the inner leaves,
// and rewrites the parent as a discriminator-keyed oneOf over the middles.
func extractHierarchical(parentName string, parent, schemas *yaml.Node, infos []memberInfo) error {
	if len(infos) == 0 {
		return errors.New("hierarchical: no members")
	}
	oneOf := lookup(parent, "oneOf")
	if oneOf == nil || oneOf.Kind != yaml.SequenceNode {
		return errors.New("hierarchical: parent has no oneOf")
	}

	// Find a grouping property: present (single-enum) on every member.
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
	if len(candidates) == 0 {
		return errors.New("no grouping property")
	}

	type score struct {
		name     string
		distinct int
	}
	var scored []score
	for k := range candidates {
		seen := make(map[string]bool)
		for _, info := range infos {
			seen[info.props[k]] = true
		}
		scored = append(scored, score{name: k, distinct: len(seen)})
	}
	sort.Slice(scored, func(i, j int) bool {
		if scored[i].distinct != scored[j].distinct {
			return scored[i].distinct > scored[j].distinct
		}
		return scored[i].name < scored[j].name
	})
	discProp := scored[0].name

	groupOrder := make([]string, 0)
	groups := make(map[string][]int)
	for i, info := range infos {
		v := info.props[discProp]
		if _, exists := groups[v]; !exists {
			groupOrder = append(groupOrder, v)
		}
		groups[v] = append(groups[v], i)
	}

	mapping := make(map[string]string, len(groupOrder))
	newParentMembers := make([]*yaml.Node, 0, len(groupOrder))

	for _, value := range groupOrder {
		middleName := parentName + toPascalCase(value)
		if lookupSchema(schemas, middleName) != nil {
			return fmt.Errorf("synthesized middle name %q already exists in components.schemas", middleName)
		}
		middleMembers := make([]*yaml.Node, 0, len(groups[value]))
		seenSuffix := make(map[string]bool)
		for _, idx := range groups[value] {
			member := oneOf.Content[idx]
			suffix, err := leafSuffix(member, schemas)
			if err != nil {
				return err
			}
			if seenSuffix[suffix] {
				return fmt.Errorf("hierarchical: duplicate leaf suffix %q in group %q of %s", suffix, value, parentName)
			}
			seenSuffix[suffix] = true
			leafName := middleName + suffix
			if lookupSchema(schemas, leafName) != nil {
				return fmt.Errorf("synthesized leaf name %q already exists in components.schemas", leafName)
			}
			appendSchema(schemas, leafName, member)
			middleMembers = append(middleMembers, refNode(leafName))
		}
		middleSchema := &yaml.Node{
			Kind: yaml.MappingNode,
			Content: []*yaml.Node{
				{Kind: yaml.ScalarNode, Value: "oneOf"},
				{Kind: yaml.SequenceNode, Content: middleMembers},
			},
		}
		appendSchema(schemas, middleName, middleSchema)
		newParentMembers = append(newParentMembers, refNode(middleName))
		mapping[value] = refPrefix + middleName
	}

	oneOf.Content = newParentMembers
	setDiscriminator(parent, discProp, mapping)
	return nil
}

// leafSuffix derives a name suffix for a hierarchical-fallback leaf. The
// member is expected to be `allOf [{$ref: <presenceLeaf>}, extras...]`; the
// suffix is PascalCase of the presenceLeaf's single required-field name
// (e.g. ref `UserOrRoleUser` with required `[user]` -> suffix `User`).
func leafSuffix(member, schemas *yaml.Node) (string, error) {
	allOf := lookup(member, "allOf")
	if allOf == nil || allOf.Kind != yaml.SequenceNode || len(allOf.Content) == 0 {
		return "", errors.New("hierarchical: member has no allOf")
	}
	ref := lookup(allOf.Content[0], "$ref")
	if ref == nil {
		return "", errors.New("hierarchical: first allOf element is not a $ref")
	}
	refName, ok := refNameFromPointer(ref.Value)
	if !ok {
		return "", fmt.Errorf("hierarchical: invalid $ref %q", ref.Value)
	}
	target := lookupSchema(schemas, refName)
	if target == nil {
		return "", fmt.Errorf("hierarchical: ref target %q not found", refName)
	}
	req := getSingleRequired(target)
	if req == "" {
		return "", fmt.Errorf("hierarchical: ref target %q is not presence-discriminated", refName)
	}
	return toPascalCase(req), nil
}

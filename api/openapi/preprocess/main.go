// Command preprocess rewrites the Lakekeeper management OpenAPI spec before
// code generation. It is deliberately minimal: the spec is the source of
// truth, and this tool only patches what the spec gets wrong for the
// generated client.
//
// It forces `style: deepObject, explode: false` on every `relations` array
// query parameter. The Lakekeeper backend deserializes the parameter via
// `axum::extract::Query` over `serde_urlencoded`, which only accepts bracket
// notation (`?relations[0]=a&relations[1]=b`) for a `Vec<T>` field. The
// generator's runtime helper applies the `[i]` suffix when
// `style == "deepObject"` and the value is a slice, so this single switch
// flips every `Get*AssignmentsById` call from the rejected multi-explode form
// to the accepted bracket form.
package main

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

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

	// The OpenAPI spec leaves `style`/`explode` unset on `relations`; OAS
	// defaults query-array params to `explode: true`, which makes
	// openapi-generator emit `?relations=a&relations=b` — rejected by the
	// backend with 400. Idempotent — already-styled params are overwritten
	// in place.
	bracketed := forceBracketsOnRelations(docRoot(&doc))

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

	fmt.Fprintf(os.Stderr, "preprocess: forced bracket notation on %d relations param(s)\n", bracketed)
	return nil
}

func docRoot(n *yaml.Node) *yaml.Node {
	if n.Kind == yaml.DocumentNode && len(n.Content) > 0 {
		return n.Content[0]
	}
	return n
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

// forceBracketsOnRelations walks every `paths.<path>.<verb>.parameters`
// entry and, for every `relations` array query parameter, sets
// `style: deepObject` and `explode: false` in place. The deepObject style
// makes the openapi-generator runtime helper emit `relations[i]=value` per
// element — the only form serde_urlencoded accepts for a `Vec<T>` query
// field on the server side. Returns the number of parameters rewritten.
func forceBracketsOnRelations(root *yaml.Node) int {
	paths := lookup(root, "paths")
	if paths == nil || paths.Kind != yaml.MappingNode {
		return 0
	}
	var count int
	for i := 1; i < len(paths.Content); i += 2 {
		pathItem := paths.Content[i]
		if pathItem == nil || pathItem.Kind != yaml.MappingNode {
			continue
		}
		for j := 1; j < len(pathItem.Content); j += 2 {
			op := pathItem.Content[j]
			if op == nil || op.Kind != yaml.MappingNode {
				continue
			}
			params := lookup(op, "parameters")
			if params == nil || params.Kind != yaml.SequenceNode {
				continue
			}
			for _, p := range params.Content {
				if p == nil || p.Kind != yaml.MappingNode {
					continue
				}
				if scalarValueOf(p, "name") != "relations" {
					continue
				}
				if scalarValueOf(lookup(p, "schema"), "type") != "array" {
					continue
				}
				setOrInsertScalar(p, "style", "deepObject")
				setOrInsertScalar(p, "explode", "false")
				count++
			}
		}
	}
	return count
}

// scalarValueOf returns the scalar value of m[key], or "" if the key is
// absent or the value is not a scalar.
func scalarValueOf(m *yaml.Node, key string) string {
	v := lookup(m, key)
	if v == nil || v.Kind != yaml.ScalarNode {
		return ""
	}
	return v.Value
}

// setOrInsertScalar sets m[key] to a scalar value, replacing any existing
// child of that key in place or appending if absent.
func setOrInsertScalar(m *yaml.Node, key, value string) {
	if m == nil || m.Kind != yaml.MappingNode {
		return
	}
	for i := 0; i+1 < len(m.Content); i += 2 {
		if m.Content[i].Value == key {
			m.Content[i+1] = &yaml.Node{Kind: yaml.ScalarNode, Value: value}
			return
		}
	}
	m.Content = append(m.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Value: key},
		&yaml.Node{Kind: yaml.ScalarNode, Value: value},
	)
}

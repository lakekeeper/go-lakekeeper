// Command prepare builds the spec the SDK is generated from.
//
// The repo commits two OpenAPI specs: the Lakekeeper spec and the Lakekeeper+
// spec, each copied unmodified from its own release. prepare reads both,
// applies each spec's patches (see patches.go), merges them, and writes the
// merged spec for openapi-generator. It runs first in the `make generate`
// pipeline; the merged spec is a git-ignored build artifact. It also writes
// spec_releases.go into the generated package from spec-releases.yaml, the
// release each spec was copied from.
//
// Merge rules (see docs/GENERATION.md):
//   - Lakekeeper is the base and wins on every path, operation and schema both
//     specs define.
//   - Lakekeeper+ adds the paths, operations and component entries Lakekeeper
//     lacks, and new variants (oneOf entry plus discriminator mapping) on
//     discriminated unions both define.
//   - Any other difference on a shared schema or operation, or an operationId
//     used by two different operations, stops prepare with an error naming the
//     item. Descriptive differences (descriptions, summaries, examples,
//     maxLength, extra documented response codes) are not conflicts.
//   - info, servers, security and security schemes come from Lakekeeper; tags
//     only Lakekeeper+ declares are appended.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"go.yaml.in/yaml/v3"
)

// spec is one input spec with the patches that fix its defects.
type spec struct {
	name    string
	data    []byte
	patches []patch
}

func main() {
	lkPath := flag.String("lakekeeper", "", "path to the Lakekeeper spec")
	plusPath := flag.String("lakekeeper-plus", "", "path to the Lakekeeper+ spec")
	outPath := flag.String("out", "", "path to write the merged spec to")
	releasesPath := flag.String("releases", "", "path to spec-releases.yaml, the release each spec was copied from")
	releasesOut := flag.String("releases-out", "", "path to write the Go file with the spec releases to")
	flag.Parse()
	if *lkPath == "" || *plusPath == "" || *outPath == "" || *releasesPath == "" || *releasesOut == "" {
		fmt.Fprintln(os.Stderr, "usage: prepare -lakekeeper <spec> -lakekeeper-plus <spec> -out <merged-spec> "+
			"-releases <spec-releases.yaml> -releases-out <go-file>")
		os.Exit(2)
	}
	if err := run(*lkPath, *plusPath, *outPath); err != nil {
		fmt.Fprintf(os.Stderr, "prepare: %v\n", err)
		os.Exit(1)
	}
	if err := writeReleases(*releasesPath, *releasesOut); err != nil {
		fmt.Fprintf(os.Stderr, "prepare: %v\n", err)
		os.Exit(1)
	}
}

func writeReleases(releasesPath, outPath string) error {
	in, err := os.ReadFile(releasesPath)
	if err != nil {
		return err
	}
	src, err := specReleasesGo(in)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(outPath, src, 0o644); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "prepare: wrote %s\n", outPath)
	return nil
}

func run(lkPath, plusPath, outPath string) error {
	lk, err := os.ReadFile(lkPath)
	if err != nil {
		return err
	}
	plus, err := os.ReadFile(plusPath)
	if err != nil {
		return err
	}
	out, err := prepare(
		spec{name: "lakekeeper", data: lk, patches: lakekeeperPatches},
		spec{name: "lakekeeper-plus", data: plus, patches: lakekeeperPlusPatches},
		os.Stderr,
	)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(outPath, out, 0o644); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "prepare: wrote %s\n", outPath)
	return nil
}

// prepare patches both specs, merges plus into lk and returns the merged spec.
// Patch results are logged to log.
func prepare(lk, plus spec, log io.Writer) ([]byte, error) {
	base, err := load(lk, log)
	if err != nil {
		return nil, err
	}
	extra, err := load(plus, log)
	if err != nil {
		return nil, err
	}
	if err := merge(base, extra); err != nil {
		return nil, fmt.Errorf("merge: %w", err)
	}
	return encode(base)
}

// load parses a spec and applies its patches.
func load(s spec, log io.Writer) (*yaml.Node, error) {
	var doc yaml.Node
	if err := yaml.Unmarshal(s.data, &doc); err != nil {
		return nil, fmt.Errorf("parse %s spec: %w", s.name, err)
	}
	if len(doc.Content) == 0 || doc.Content[0].Kind != yaml.MappingNode {
		return nil, fmt.Errorf("parse %s spec: not a YAML mapping", s.name)
	}
	root := doc.Content[0]
	if err := applyPatches(root, s.patches, log); err != nil {
		return nil, fmt.Errorf("%s spec: %w", s.name, err)
	}
	return root, nil
}

func encode(root *yaml.Node) ([]byte, error) {
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(root); err != nil {
		return nil, fmt.Errorf("encode merged spec: %w", err)
	}
	if err := enc.Close(); err != nil {
		return nil, fmt.Errorf("encode merged spec: %w", err)
	}
	return buf.Bytes(), nil
}

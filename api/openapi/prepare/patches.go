package main

import (
	"fmt"
	"io"

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

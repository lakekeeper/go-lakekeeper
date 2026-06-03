// Command postprocess relaxes the generated management SDK's response decoding.
//
// openapi-generator (with generateUnmarshalJSON) emits a per-model UnmarshalJSON
// that first validates required-property presence and then strict-decodes via
// `decoder.DisallowUnknownFields()`. The strict decode is the wrong posture for
// a client library: when Lakekeeper adds a field to a response (e.g. latest-main
// added `allowed-format-versions` to the warehouse object after v0.12.3), a
// client generated against an older spec would reject the whole response.
//
// This tool removes every per-model `decoder.DisallowUnknownFields()` call so
// the decoder ignores unknown fields, while leaving intact:
//   - the required-property presence check (a separate phase that runs first),
//   - oneOf discriminator dispatch, and
//   - the `dec.DisallowUnknownFields()` inside utils.go's newStrictDecoder,
//     which only decodes the discriminator lookup into a map (where the call is
//     a no-op anyway).
//
// It is the minimal lever: presence-discriminated unions still disambiguate by
// required-field presence, and enum unions decode by discriminator — neither
// depends on strict unknown-field rejection. The generated tree is rewritten in
// place; run it from the `make generate` pipeline after the generator and before
// gofmt/goimports.
//
// It performs a second, related relaxation: generated enum UnmarshalJSON methods
// reject any value not in the AllowedXEnumValues list with
// `return fmt.Errorf("%+v is not a valid X", value)`. For the same forward-compat
// reason — a newer Lakekeeper may return an enum value this SDK's spec predates
// (e.g. latest-main's `set_format_version_policy` WarehouseAction) and that must
// not fail decoding of the whole response — the error return is rewritten to
// store the unknown value (`*v = enumTypeValue; return nil`). The enum type is a
// `string` alias, so the raw value is preserved for the caller, who can compare
// against the known AllowedXEnumValues. The subsequent goimports pass drops the
// now-unused `fmt` import where applicable.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// stripTarget is the exact (whitespace-trimmed) statement to remove. The
// generator names the decoder variable `decoder` in model UnmarshalJSON; the
// `dec.DisallowUnknownFields()` in newStrictDecoder uses a different receiver
// and is intentionally not matched.
const stripTarget = "decoder.DisallowUnknownFields()"

// enumErr{Prefix,Suffix} bracket the generated enum-validation error line, e.g.
// `return fmt.Errorf("%+v is not a valid WarehouseAction", value)`. The type
// name in the middle varies per enum, so we match on the surrounding literal.
const (
	enumErrPrefix = `return fmt.Errorf("%+v is not a valid `
	enumErrSuffix = `", value)`
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: postprocess <generated-dir>")
		os.Exit(2)
	}
	if err := run(os.Args[1]); err != nil {
		fmt.Fprintf(os.Stderr, "postprocess: %v\n", err)
		os.Exit(1)
	}
}

func run(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("read dir: %w", err)
	}

	files, removed, relaxedEnums := 0, 0, 0
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") {
			continue
		}
		path := filepath.Join(dir, e.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read %s: %w", path, err)
		}
		out, n := stripDisallowUnknownFields(string(data))
		out, m := tolerateUnknownEnumValues(out)
		if n == 0 && m == 0 {
			continue
		}
		if err := os.WriteFile(path, []byte(out), 0o644); err != nil {
			return fmt.Errorf("write %s: %w", path, err)
		}
		files++
		removed += n
		relaxedEnums += m
	}

	fmt.Fprintf(os.Stderr, "postprocess: removed %d DisallowUnknownFields call(s), relaxed %d enum validation(s), across %d file(s)\n", removed, relaxedEnums, files)
	return nil
}

// stripDisallowUnknownFields removes every line whose trimmed content is exactly
// stripTarget, returning the rewritten source and the number of lines removed.
// Other lines (including newStrictDecoder's `dec.DisallowUnknownFields()`) are
// preserved verbatim.
func stripDisallowUnknownFields(src string) (string, int) {
	lines := strings.Split(src, "\n")
	kept := make([]string, 0, len(lines))
	removed := 0
	for _, line := range lines {
		if strings.TrimSpace(line) == stripTarget {
			removed++
			continue
		}
		kept = append(kept, line)
	}
	if removed == 0 {
		return src, 0
	}
	return strings.Join(kept, "\n"), removed
}

// tolerateUnknownEnumValues rewrites each generated enum-validation error return
// (`return fmt.Errorf("%+v is not a valid X", value)`) into storing the value
// and returning nil (`*v = enumTypeValue` / `return nil`). The generator uses
// the receiver `v` and local `enumTypeValue` uniformly across enum models, so
// the replacement is identical everywhere. Indentation of the matched line is
// preserved. Returns the rewritten source and the number of enums relaxed.
func tolerateUnknownEnumValues(src string) (string, int) {
	lines := strings.Split(src, "\n")
	out := make([]string, 0, len(lines)+1)
	count := 0
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, enumErrPrefix) && strings.HasSuffix(trimmed, enumErrSuffix) {
			indent := line[:len(line)-len(strings.TrimLeft(line, " \t"))]
			out = append(out, indent+"*v = enumTypeValue", indent+"return nil")
			count++
			continue
		}
		out = append(out, line)
	}
	if count == 0 {
		return src, 0
	}
	return strings.Join(out, "\n"), count
}

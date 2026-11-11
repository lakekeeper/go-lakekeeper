//go:build ignore

// coverage reports which Management API operations in the OpenAPI spec are
// generated into the SDK (pkg/apis/management/v1) and wrapped by a pkg/client
// façade. Operations are matched by HTTP method + path, never by name, so a
// renamed operationId or façade method cannot hide a gap.
//
// Run from the repo root:
//
//	go run .claude/skills/api-coverage/apicoverage.go            # markdown table
//	go run .claude/skills/api-coverage/apicoverage.go -missing   # gaps only
//	go run .claude/skills/api-coverage/apicoverage.go -json
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	specPath  = "api/openapi/management-open-api.yaml"
	sdkDir    = "pkg/apis/management/v1"
	facadeDir = "pkg/client"
)

var httpMethods = []string{"get", "put", "post", "delete", "patch", "head", "options"}

type op struct {
	Method      string   `json:"method"`
	Path        string   `json:"path"`
	OperationID string   `json:"operationId"`
	Tag         string   `json:"tag"`
	Deprecated  bool     `json:"deprecated,omitempty"`
	SDK         string   `json:"sdk,omitempty"`     // Service.Method
	Facades     []string `json:"facades,omitempty"` // Type.Method
}

func key(method, path string) string { return strings.ToUpper(method) + " " + path }

func main() {
	missingOnly := flag.Bool("missing", false, "list only operations missing from the SDK or façades")
	asJSON := flag.Bool("json", false, "emit JSON instead of markdown")
	flag.Parse()

	ops, err := loadSpec(specPath)
	check(err)
	sdk, err := loadSDK(sdkDir) // key -> Service.Method
	check(err)
	facades, err := loadFacades(facadeDir, sdk) // SDK method name -> []Type.Method
	check(err)

	sdkByKey := map[string]bool{}
	for i := range ops {
		k := key(ops[i].Method, ops[i].Path)
		if s, ok := sdk[k]; ok {
			ops[i].SDK = s
			sdkByKey[k] = true
			ops[i].Facades = facades[s[strings.Index(s, ".")+1:]]
		}
	}
	// SDK operations absent from the spec mean the generated code is stale.
	var stale []string
	specKeys := map[string]bool{}
	for _, o := range ops {
		specKeys[key(o.Method, o.Path)] = true
	}
	for k, s := range sdk {
		if !specKeys[k] {
			stale = append(stale, k+" ("+s+")")
		}
	}
	sort.Strings(stale)

	if *asJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		check(enc.Encode(map[string]any{"operations": ops, "staleSDK": stale}))
		return
	}
	printMarkdown(ops, stale, *missingOnly)
}

func loadSpec(path string) ([]op, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var doc struct {
		Paths map[string]map[string]yaml.Node `yaml:"paths"`
	}
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	var ops []op
	for p, item := range doc.Paths {
		for _, m := range httpMethods {
			n, ok := item[m]
			if !ok {
				continue
			}
			var meta struct {
				OperationID string   `yaml:"operationId"`
				Tags        []string `yaml:"tags"`
				Deprecated  bool     `yaml:"deprecated"`
			}
			if err := n.Decode(&meta); err != nil {
				return nil, fmt.Errorf("%s %s: %w", m, p, err)
			}
			tag := "untagged"
			if len(meta.Tags) > 0 {
				tag = meta.Tags[0]
			}
			ops = append(ops, op{Method: strings.ToUpper(m), Path: p, OperationID: meta.OperationID, Tag: tag, Deprecated: meta.Deprecated})
		}
	}
	sort.Slice(ops, func(i, j int) bool {
		if ops[i].Tag != ops[j].Tag {
			return ops[i].Tag < ops[j].Tag
		}
		if ops[i].Path != ops[j].Path {
			return ops[i].Path < ops[j].Path
		}
		return ops[i].Method < ops[j].Method
	})
	return ops, nil
}

var (
	execRe   = regexp.MustCompile(`(?m)^func \(a \*(\w+APIService)\) (\w+)Execute\(`)
	methodRe = regexp.MustCompile(`localVarHTTPMethod\s*=\s*http\.Method(\w+)`)
	pathRe   = regexp.MustCompile(`localVarPath := localBasePath \+ "([^"]+)"`)
)

// loadSDK reads each generated XExecute body for its HTTP method and path.
func loadSDK(dir string) (map[string]string, error) {
	files, err := filepath.Glob(filepath.Join(dir, "api_*.go"))
	if err != nil {
		return nil, err
	}
	out := map[string]string{}
	for _, f := range files {
		src, err := os.ReadFile(f)
		if err != nil {
			return nil, err
		}
		s := string(src)
		locs := execRe.FindAllStringSubmatchIndex(s, -1)
		for i, l := range locs {
			end := len(s)
			if i+1 < len(locs) {
				end = locs[i+1][0]
			}
			body := s[l[1]:end]
			mm, pm := methodRe.FindStringSubmatch(body), pathRe.FindStringSubmatch(body)
			if mm == nil || pm == nil {
				return nil, fmt.Errorf("%s: cannot find method/path for %s", f, s[l[4]:l[5]])
			}
			out[key(mm[1], pm[1])] = s[l[2]:l[3]] + "." + s[l[4]:l[5]]
		}
	}
	return out, nil
}

// loadFacades finds every call in non-test pkg/client code whose selector is
// a generated SDK method name, and credits it to the exported Type.Method (or
// func) that reaches it, following calls into unexported same-package helpers
// such as `p.update(...)`.
func loadFacades(dir string, sdk map[string]string) (map[string][]string, error) {
	sdkNames := map[string]bool{}
	for _, s := range sdk {
		sdkNames[s[strings.Index(s, ".")+1:]] = true
	}
	type fnInfo struct {
		exported bool
		sdkCalls map[string]bool
		callees  map[string]bool
	}
	funcs := map[string]*fnInfo{}
	fset := token.NewFileSet()
	files, err := filepath.Glob(filepath.Join(dir, "*.go"))
	if err != nil {
		return nil, err
	}
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(fset, f, nil, 0)
		if err != nil {
			return nil, err
		}
		for _, d := range file.Decls {
			fn, ok := d.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				continue
			}
			name, recvType, recvVar := fn.Name.Name, "", ""
			if fn.Recv != nil && len(fn.Recv.List) == 1 {
				recvType = recvName(fn.Recv.List[0].Type)
				if len(fn.Recv.List[0].Names) == 1 {
					recvVar = fn.Recv.List[0].Names[0].Name
				}
				name = recvType + "." + name
			}
			info := &fnInfo{
				exported: ast.IsExported(fn.Name.Name) && (recvType == "" || ast.IsExported(recvType)),
				sdkCalls: map[string]bool{},
				callees:  map[string]bool{},
			}
			funcs[name] = info
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				switch fun := call.Fun.(type) {
				case *ast.SelectorExpr:
					if sdkNames[fun.Sel.Name] {
						info.sdkCalls[fun.Sel.Name] = true
					}
					if id, ok := fun.X.(*ast.Ident); ok && recvVar != "" && id.Name == recvVar {
						info.callees[recvType+"."+fun.Sel.Name] = true
					}
				case *ast.Ident:
					info.callees[fun.Name] = true
				case *ast.IndexExpr: // generic call: helper[T](...)
					if id, ok := fun.X.(*ast.Ident); ok {
						info.callees[id.Name] = true
					}
				}
				return true
			})
		}
	}
	var reach func(name string, seen map[string]bool, out map[string]bool)
	reach = func(name string, seen map[string]bool, out map[string]bool) {
		info, ok := funcs[name]
		if !ok || seen[name] {
			return
		}
		seen[name] = true
		for c := range info.sdkCalls {
			out[c] = true
		}
		for c := range info.callees {
			if f, ok := funcs[c]; ok && !f.exported {
				reach(c, seen, out)
			}
		}
	}
	owners := map[string]map[string]bool{}
	for name, info := range funcs {
		if !info.exported {
			continue
		}
		calls := map[string]bool{}
		reach(name, map[string]bool{}, calls)
		for c := range calls {
			if owners[c] == nil {
				owners[c] = map[string]bool{}
			}
			owners[c][name] = true
		}
	}
	out := map[string][]string{}
	for c, os := range owners {
		for o := range os {
			out[c] = append(out[c], o)
		}
		sort.Strings(out[c])
	}
	return out, nil
}

func recvName(e ast.Expr) string {
	switch t := e.(type) {
	case *ast.StarExpr:
		return recvName(t.X)
	case *ast.Ident:
		return t.Name
	case *ast.IndexExpr:
		return recvName(t.X)
	}
	return "?"
}

func printMarkdown(ops []op, stale []string, missingOnly bool) {
	var nSDK, nFacade int
	for _, o := range ops {
		if o.SDK != "" {
			nSDK++
		}
		if len(o.Facades) > 0 {
			nFacade++
		}
	}
	fmt.Printf("Spec operations: %d · in SDK: %d · behind a façade: %d\n\n", len(ops), nSDK, nFacade)
	tag := ""
	for _, o := range ops {
		if missingOnly && o.SDK != "" && len(o.Facades) > 0 {
			continue
		}
		if o.Tag != tag {
			tag = o.Tag
			fmt.Printf("\n## %s\n\n| Method | Path | operationId | SDK | Façade |\n|---|---|---|---|---|\n", tag)
		}
		sdk, fac := o.SDK, strings.Join(o.Facades, ", ")
		if sdk == "" {
			sdk = "**MISSING**"
		}
		if fac == "" {
			fac = "—"
		}
		id := o.OperationID
		if o.Deprecated {
			id += " (deprecated)"
		}
		fmt.Printf("| %s | `%s` | %s | %s | %s |\n", o.Method, strings.TrimPrefix(o.Path, "/management/v1"), id, sdk, fac)
	}
	if len(stale) > 0 {
		fmt.Printf("\n## SDK operations not in the spec (stale generation)\n\n")
		for _, s := range stale {
			fmt.Println("- " + s)
		}
	}
}

func check(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "coverage:", err)
		os.Exit(1)
	}
}

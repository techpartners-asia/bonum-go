// Package architecture_test enforces the dependency rule from
// docs/superpowers/specs/2026-09-14-ddd-layered-contexts-design.md: inside each bounded
// context, imports point inward (adapters -> ports/domain, application -> ports/domain,
// ports -> domain, domain -> nothing), contexts never import each other, and only adapters
// and facades may reach third-party code or internal/rest.
package architecture_test

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const module = "github.com/techpartners-asia/bonum-go"

var contexts = []string{"gateway", "wallet"}

// classify maps a directory (module-relative, slash separated) to its context and layer.
// It returns ok=false for directories the rule does not cover (tests, internal).
func classify(dir string) (ctx, layer string, ok bool) {
	if dir == "." {
		return "gateway", "facade", true
	}
	parts := strings.Split(dir, "/")
	for _, c := range contexts {
		if parts[0] != c {
			continue
		}
		if len(parts) == 1 {
			return c, "facade", true
		}
		return c, parts[1], true
	}
	return "", "", false
}

func isStdlib(imp string) bool {
	first, _, _ := strings.Cut(imp, "/")
	return !strings.Contains(first, ".")
}

// allowed reports whether a file in (ctx, layer) may import imp.
func allowed(ctx, layer, imp string) bool {
	if isStdlib(imp) {
		return true
	}
	rel, internal := strings.CutPrefix(imp, module+"/")
	if !internal {
		return layer == "adapters" || layer == "facade"
	}
	for _, other := range contexts {
		if other != ctx && (rel == other || strings.HasPrefix(rel, other+"/")) {
			return false
		}
	}
	domain := strings.HasPrefix(rel, ctx+"/domain")
	ports := rel == ctx+"/ports"
	application := strings.HasPrefix(rel, ctx+"/application/")
	switch layer {
	case "domain":
		return domain
	case "ports":
		return domain
	case "application":
		// application also covers its own commands/<aggregate> and queries/<aggregate>
		// subpackages (CQRS-lite): a facade file imports those, and each of those
		// imports only domain + ports, same as any other application-layer file.
		return domain || ports || application
	case "adapters":
		return domain || ports || rel == "internal/rest"
	case "facade":
		return strings.HasPrefix(rel, ctx+"/") || rel == "internal/rest"
	}
	return false
}

func TestImportsPointInward(t *testing.T) {
	root := filepath.Join("..", "..")
	var violations []string
	fset := token.NewFileSet()
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if name := d.Name(); name == ".git" || name == "tests" || name == "docs" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		rel, _ := filepath.Rel(root, path)
		dir := filepath.ToSlash(filepath.Dir(rel))
		ctx, layer, ok := classify(dir)
		if !ok {
			return nil
		}
		f, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		for _, imp := range f.Imports {
			p := strings.Trim(imp.Path.Value, `"`)
			if !allowed(ctx, layer, p) {
				violations = append(violations, rel+": "+layer+" layer may not import "+p)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(violations) > 0 {
		t.Fatalf("dependency rule violated:\n  %s", strings.Join(violations, "\n  "))
	}
}

func TestRuleCatchesViolations(t *testing.T) {
	bad := []struct{ ctx, layer, imp string }{
		{"gateway", "domain", module + "/internal/rest"},
		{"gateway", "domain", "resty.dev/v3"},
		{"gateway", "application", module + "/gateway/adapters/httpapi"},
		{"gateway", "ports", module + "/gateway/application"},
		{"gateway", "adapters", module + "/wallet/domain"},
		{"wallet", "facade", module + "/gateway/domain"},
		{"gateway", "application", module + "/wallet/application/commands/payment"},
	}
	for _, b := range bad {
		if allowed(b.ctx, b.layer, b.imp) {
			t.Errorf("%s/%s importing %s must be rejected", b.ctx, b.layer, b.imp)
		}
	}
	good := []struct{ ctx, layer, imp string }{
		{"gateway", "domain", "encoding/json"},
		{"gateway", "domain", module + "/gateway/domain/checkout"},
		{"gateway", "application", module + "/gateway/ports"},
		{"gateway", "application", module + "/gateway/application/commands/card"},
		{"wallet", "application", module + "/wallet/application/queries/payment"},
		{"gateway", "adapters", "resty.dev/v3"},
		{"wallet", "facade", module + "/wallet/application"},
	}
	for _, g := range good {
		if !allowed(g.ctx, g.layer, g.imp) {
			t.Errorf("%s/%s importing %s must be allowed", g.ctx, g.layer, g.imp)
		}
	}
}

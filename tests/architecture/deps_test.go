// Package architecture_test enforces the dependency rule from
// docs/superpowers/specs/2026-09-14-ddd-layered-contexts-design.md: inside each bounded
// context, imports point inward (adapters -> ports/domain, application -> ports/domain,
// ports -> domain, domain -> nothing), contexts never import each other, and only adapters
// and facades may reach third-party code or internal/rest. Each context's domain, ports,
// application and adapters live under internal/<context>/ so nothing outside this module can
// import them at all; only the facade packages (bonum at the module root, wallet at wallet/)
// are the public surface.
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
// It returns ok=false for directories the rule does not cover (tests, internal/rest, or
// internal/<context> itself, which holds no .go files directly).
func classify(dir string) (ctx, layer string, ok bool) {
	if dir == "." {
		return "gateway", "facade", true
	}
	if dir == "wallet" {
		return "wallet", "facade", true
	}
	rel, isInternal := strings.CutPrefix(dir, "internal/")
	if !isInternal {
		return "", "", false
	}
	parts := strings.Split(rel, "/")
	for _, c := range contexts {
		if parts[0] != c {
			continue
		}
		if len(parts) == 1 {
			return "", "", false // internal/<context> itself: CONTEXT.md only, no .go files
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
	rel, isModuleInternal := strings.CutPrefix(imp, module+"/")
	if !isModuleInternal {
		return layer == "adapters" || layer == "facade"
	}
	// Neither context may reach the other's internal tree or facade package.
	for _, other := range contexts {
		if other == ctx {
			continue
		}
		if rel == other || strings.HasPrefix(rel, other+"/") ||
			rel == "internal/"+other || strings.HasPrefix(rel, "internal/"+other+"/") {
			return false
		}
	}
	ownPrefix := "internal/" + ctx + "/"
	own := strings.HasPrefix(rel, ownPrefix)
	domain := own && strings.HasPrefix(rel, ownPrefix+"domain")
	ports := rel == ownPrefix+"ports"
	// application also covers its own commands/<aggregate> and queries/<aggregate>
	// subpackages (CQRS-lite): a facade file imports those, and each of those imports
	// only domain + ports, same as any other application-layer file.
	application := own && strings.HasPrefix(rel, ownPrefix+"application/")
	switch layer {
	case "domain":
		return domain
	case "ports":
		return domain
	case "application":
		return domain || ports || application
	case "adapters":
		return domain || ports || rel == "internal/rest"
	case "facade":
		return own || rel == "internal/rest"
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
		{"gateway", "application", module + "/internal/gateway/adapters/httpapi"},
		{"gateway", "ports", module + "/internal/gateway/application"},
		{"gateway", "adapters", module + "/internal/wallet/domain"},
		{"gateway", "facade", module + "/wallet"},
		{"wallet", "facade", module + "/internal/gateway/domain"},
		{"gateway", "application", module + "/internal/wallet/application/commands/payment"},
	}
	for _, b := range bad {
		if allowed(b.ctx, b.layer, b.imp) {
			t.Errorf("%s/%s importing %s must be rejected", b.ctx, b.layer, b.imp)
		}
	}
	good := []struct{ ctx, layer, imp string }{
		{"gateway", "domain", "encoding/json"},
		{"gateway", "domain", module + "/internal/gateway/domain/checkout"},
		{"gateway", "application", module + "/internal/gateway/ports"},
		{"gateway", "application", module + "/internal/gateway/application/commands/card"},
		{"wallet", "application", module + "/internal/wallet/application/queries/payment"},
		{"gateway", "adapters", "resty.dev/v3"},
		{"wallet", "facade", module + "/internal/wallet/application"},
	}
	for _, g := range good {
		if !allowed(g.ctx, g.layer, g.imp) {
			t.Errorf("%s/%s importing %s must be allowed", g.ctx, g.layer, g.imp)
		}
	}
}

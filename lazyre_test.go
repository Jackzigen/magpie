package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yetone/magpie/internal/lazyre"
)

// TestAllLazyPatternsCompile compiles every regexp the binary's packages
// compile on first use (internal/lazyre), so a bad one fails here and not
// on a user's machine.
func TestAllLazyPatternsCompile(t *testing.T) {
	if n := lazyre.MustAll(); n < 500 {
		t.Fatalf("only %d lazy patterns registered; the binary's packages hold more", n)
	}
}

// TestNoEagerPackageRegexps fails on a package-level regexp.MustCompile
// (or regexp.Compile) outside a function: every package's init runs on
// every launch, `magpie statusline` on each Claude Code message included,
// so such a pattern is compiled for a path most launches never take. Use
// lazyre.New, which compiles on first use.
//
// third_party/ is left out: a vendored library is kept as its upstream
// wrote it (yaml.v3 differs from gofmt'd upstream by the 7 emitter lines
// of 0caa09b3d, and MAGPIE.md's sync procedure diffs against upstream), so
// magpie's lazyre discipline does not reach into it (#6617).
func TestNoEagerPackageRegexps(t *testing.T) {
	var bad []string
	err := filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if path != "." && (strings.HasPrefix(d.Name(), ".") || d.Name() == "testdata" || d.Name() == "node_modules" || d.Name() == "site" || d.Name() == "third_party") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
		if err != nil {
			return nil // not this test's to judge
		}
		for _, d := range f.Decls {
			gd, ok := d.(*ast.GenDecl)
			if !ok || gd.Tok != token.VAR {
				continue
			}
			ast.Inspect(gd, func(n ast.Node) bool {
				if _, ok := n.(*ast.FuncLit); ok {
					return false
				}
				if se, ok := n.(*ast.SelectorExpr); ok {
					if id, ok := se.X.(*ast.Ident); ok && id.Name == "regexp" && (se.Sel.Name == "MustCompile" || se.Sel.Name == "Compile" || se.Sel.Name == "MustCompilePOSIX") {
						bad = append(bad, fset.Position(se.Pos()).String())
					}
				}
				return true
			})
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, b := range bad {
		t.Errorf("%s: a package-level regexp is compiled at launch; use lazyre.New", b)
	}
}

// TestNoEagerPackageRegexpsLeavesVendoredCodeOut holds the third_party
// exemption of TestNoEagerPackageRegexps to the boundary it was made for:
// the walk skips a third_party directory wherever it sits, and still flags
// an eager package-level regexp in the tree beside it (#6617). It fails if
// the exemption is widened (every directory skipped) or lost (vendored
// code judged by magpie's lazyre rule again).
func TestNoEagerPackageRegexpsLeavesVendoredCodeOut(t *testing.T) {
	dir := t.TempDir()
	write := func(rel, body string) {
		path := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	// Vendored upstream code, written as upstream wrote it: an eager
	// package-level regexp that magpie's rule would otherwise flag.
	write(filepath.Join("third_party", "yaml.v3", "encode.go"),
		"package yaml\n\nimport \"regexp\"\n\nvar base60float = regexp.MustCompile(`^[-+]?[0-9]`)\n")
	// First-party code beside it: the rule still applies.
	write(filepath.Join("internal", "eager.go"),
		"package internal\n\nimport \"regexp\"\n\nvar version = regexp.MustCompile(`^v[0-9]+`)\n")

	var bad []string
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if path != "." && (strings.HasPrefix(d.Name(), ".") || d.Name() == "testdata" || d.Name() == "node_modules" || d.Name() == "site" || d.Name() == "third_party") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
		if err != nil {
			return nil // not this test's to judge
		}
		for _, d := range f.Decls {
			gd, ok := d.(*ast.GenDecl)
			if !ok || gd.Tok != token.VAR {
				continue
			}
			ast.Inspect(gd, func(n ast.Node) bool {
				if _, ok := n.(*ast.FuncLit); ok {
					return false
				}
				if se, ok := n.(*ast.SelectorExpr); ok {
					if id, ok := se.X.(*ast.Ident); ok && id.Name == "regexp" && (se.Sel.Name == "MustCompile" || se.Sel.Name == "Compile" || se.Sel.Name == "MustCompilePOSIX") {
						bad = append(bad, fset.Position(se.Pos()).String())
					}
				}
				return true
			})
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(bad) != 1 || !strings.Contains(bad[0], filepath.Join("internal", "eager.go")) {
		t.Fatalf("walk flagged %v; want only internal/eager.go", bad)
	}
}

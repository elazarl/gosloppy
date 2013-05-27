package visitors

import (
	"go/ast"
	"go/token"

	"github.com/elazarl/gosloppy/imports"
	"github.com/elazarl/gosloppy/patch"
	"github.com/elazarl/gosloppy/scopes"
)

// NewAutoImporter returns an AutoImporter visitor that generates patches to add missing
// import statements to the standard library.
//     auto := NewAutoImporter(patchable.File)
//     scopes.WalkFile(patchable.File, auto)
//     patchable.FprintPatched(os.Stdout, patchable.All(), auto.Patches)
func NewAutoImporter(file *ast.File) *AutoImporter {
	auto := &AutoImporter{patch.Patches{}, make(map[*ast.Ident]bool), make(map[string]bool), file.Name.End()}
	for _, imp := range file.Imports {
		auto.m[imports.GetNameOrGuess(imp)] = true
	}
	return auto
}

// AutoImporter is a visitor for scopes.Walk* functions, it generate patches to add missing
// import statements from the standard library. Note that it will not add ambigious import
// (i.e. template, which can either be text/template or html/template).
type AutoImporter struct {
	Patches    patch.Patches
	Irrelevant map[*ast.Ident]bool
	m          map[string]bool
	pkg        token.Pos
}

func (v *AutoImporter) VisitExpr(scope *ast.Scope, expr ast.Expr) scopes.Visitor {
	switch expr := expr.(type) {
	case *ast.Ident:
		if v.Irrelevant[expr] {
			return v
		}
		if importname, ok := imports.RevStdlib[expr.Name]; ok && len(importname) == 1 &&
			!v.m[expr.Name] && scopes.Lookup(scope, expr.Name) == nil {
			v.m[expr.Name] = true // don't add it again
			v.Patches = append(v.Patches, patch.Insert(v.pkg, "; import "+importname[0]))
		}
	case *ast.SelectorExpr:
		v.Irrelevant[expr.Sel] = true
	case *ast.KeyValueExpr:
		// if we get a := struct {Count int} {Count: 1}, disregard Count
		if id, ok := expr.Key.(*ast.Ident); ok {
			v.Irrelevant[id] = true
		}
	}
	return v
}

func (v *AutoImporter) VisitDecl(scope *ast.Scope, decl ast.Decl) scopes.Visitor {
	return v
}

func (v *AutoImporter) VisitStmt(scope *ast.Scope, stmt ast.Stmt) scopes.Visitor {
	return v
}

func (v *AutoImporter) ExitScope(scope *ast.Scope, node ast.Node, last bool) scopes.Visitor {
	return v
}

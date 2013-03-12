package main

import (
	"go/ast"
	"go/token"

	"github.com/elazarl/gosloppy/imports"
	"github.com/elazarl/gosloppy/patch"
)

func NewAutoImporter(file *ast.File) *AutoImporter {
	auto := &AutoImporter{patch.Patches{}, make(map[string]bool), file.Name.End()}
	for _, imp := range file.Imports {
		auto.m[imports.GetNameOrGuess(imp)] = true
	}
	return auto
}

type AutoImporter struct {
	Patches patch.Patches
	m       map[string]bool
	pkg     token.Pos
}

func (v *AutoImporter) VisitExpr(scope *ast.Scope, expr ast.Expr) ScopeVisitor {
	switch expr := expr.(type) {
	case *ast.Ident:
		if importname, ok := imports.RevStdlib[expr.Name]; ok && len(importname) == 1 &&
			!v.m[expr.Name] && Lookup(scope, expr.Name) == nil {
			v.Patches = append(v.Patches, patch.Insert(v.pkg, "; import "+importname[0]))
		}
	case *ast.SelectorExpr:
		v.VisitExpr(scope, expr.X)
		return nil
	}
	return v
}

func (v *AutoImporter) VisitStmt(scope *ast.Scope, stmt ast.Stmt) ScopeVisitor {
	return v
}

func (v *AutoImporter) ExitScope(scope *ast.Scope, node ast.Node, last bool) ScopeVisitor {
	return v
}

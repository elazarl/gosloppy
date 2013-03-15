package main

import (
	"go/ast"

	"github.com/elazarl/gosloppy/patch"
)

type ShortError struct {
	Patches patch.Patches
}

func (v *ShortError) VisitExpr(scope *ast.Scope, expr ast.Expr) ScopeVisitor {
	return v
}

func (v *ShortError) VisitStmt(scope *ast.Scope, stmt ast.Stmt) ScopeVisitor {
	return v
}

func (v *ShortError) ExitScope(scope *ast.Scope, node ast.Node, last bool) ScopeVisitor {
	return v
}

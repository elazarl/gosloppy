package main

import (
	"go/ast"
)

type MultiVisitor []ScopeVisitor

func (v MultiVisitor) VisitExpr(scope *ast.Scope, expr ast.Expr) ScopeVisitor {
	any := false
	for i, w := range v {
		if w != nil {
			v[i] = w.VisitExpr(scope, expr)
			any = true
		}
	}
	if !any {
		return nil
	}
	return v
}

func (v MultiVisitor) VisitStmt(scope *ast.Scope, stmt ast.Stmt) ScopeVisitor {
	any := false
	for i, w := range v {
		if w != nil {
			v[i] = w.VisitStmt(scope, stmt)
			any = true
		}
	}
	if !any {
		return nil
	}
	return v
}

func (v MultiVisitor) ExitScope(scope *ast.Scope, node ast.Node, last bool) ScopeVisitor {
	any := false
	for i, w := range v {
		if w != nil {
			v[i] = w.ExitScope(scope, node, last)
			any = true
		}
	}
	if !any {
		return nil
	}
	return v
}

package main

import (
	"go/ast"
)

type MultiVisitor []ScopeVisitor

func (v MultiVisitor) VisitExpr(scope *ast.Scope, expr ast.Expr) ScopeVisitor {
	allnil := true
	for _, w := range v {
		if w.VisitExpr(scope, expr) != nil {
			allnil = false
		}
	}
	if allnil == true {
		return nil
	}
	return v
}

func (v MultiVisitor) VisitStmt(scope *ast.Scope, stmt ast.Stmt) ScopeVisitor {
	allnil := true
	for _, w := range v {
		if w.VisitStmt(scope, stmt) != nil {
			allnil = false
		}
	}
	if allnil == true {
		return nil
	}
	return v
}

func (v MultiVisitor) ExitScope(scope *ast.Scope, node ast.Node, last bool) ScopeVisitor {
	allnil := true
	for _, w := range v {
		if w.ExitScope(scope, node, last) != nil {
			allnil = false
		}
	}
	if allnil == true {
		return nil
	}
	return v
}

package main

import (
	"go/ast"
)

type aggr struct {
	v       ScopeVisitor
	visited bool
}

type MultiVisitor []ScopeVisitor

func (v MultiVisitor) VisitExpr(scope *ast.Scope, expr ast.Expr) ScopeVisitor {
	nrnil := 0
	for _, w := range v {
		if w.VisitExpr(scope, expr) == nil {
			nrnil++
		}
	}
	if nrnil != len(v) && nrnil != 0 {
		panic("all visitors must decide together whether or not to descend")
	}
	if nrnil > 0 {
		return nil
	}
	return v
}

func (v MultiVisitor) VisitStmt(scope *ast.Scope, stmt ast.Stmt) ScopeVisitor {
	nrnil := 0
	for _, w := range v {
		if w.VisitStmt(scope, stmt) == nil {
			nrnil++
		}
	}
	if nrnil != len(v) && nrnil != 0 {
		panic("all visitors must decide together whether or not to descend")
	}
	if nrnil > 0 {
		return nil
	}
	return v
}

func (v MultiVisitor) ExitScope(scope *ast.Scope, node ast.Node, last bool) ScopeVisitor {
	nrnil := 0
	for _, w := range v {
		if w.ExitScope(scope, node, last) == nil {
			nrnil++
		}
	}
	if nrnil != len(v) && nrnil != 0 {
		panic("all visitors must decide together whether or not to descend")
	}
	if nrnil > 0 {
		return nil
	}
	return v
}

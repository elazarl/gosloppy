package main

import (
	"go/ast"
)

// Walks through all unused objects in package
func UnusedInPackage(pkg *ast.Package, f func(obj *ast.Object) bool) {
	panic("Not implemented")
}

func UnusedInFile(file *ast.File, f func(obj *ast.Object)) {
	WalkFile(&UnusedVisitor{make(map[*ast.Object]bool), f}, file)
}

type UnusedVisitor struct {
	used map[*ast.Object]bool
	f    func(obj *ast.Object)
}

func (v *UnusedVisitor) VisitStmt(*ast.Scope, ast.Stmt) ScopeVisitor {
	return v
}

func (v *UnusedVisitor) VisitExpr(scope *ast.Scope, expr ast.Expr) ScopeVisitor {
	switch expr := expr.(type) {
	case *ast.Ident:
		v.used[expr.Obj] = true
	case *ast.SelectorExpr:
		v.VisitExpr(scope, expr.X)
		return nil
	}
	return v
}

func (v *UnusedVisitor) ExitScope(scope *ast.Scope) ScopeVisitor {
	for _, obj := range scope.Objects {
		if !v.used[obj] {
			v.f(obj)
		}
	}
	return v
}

package main

import (
	"go/ast"
)

// Walks through all unused objects in package
func UnusedInPackage(pkg *ast.Package, f func(obj *ast.Object) bool) {
	panic("Not implemented")
}

type Visitor interface {
	UnusedObj(obj *ast.Object)
	UnusedImport(imp *ast.ImportSpec)
}

func anonymousImport(name *ast.Ident) bool {
	return name != nil && (name.Name == "_" || name.Name == ".")
}

func UnusedInFile(file *ast.File, v Visitor) {
	uv := newUnusedVisitor(v)
	WalkFile(uv, file)
	for _, imp := range file.Imports {
		if !uv.usedImports[imp.Path.Value] && !anonymousImport(imp.Name) {
			v.UnusedImport(imp)
		}
	}
}

func newUnusedVisitor(v Visitor) *unusedVisitor {
	return &unusedVisitor{make(map[*ast.Object]bool), make(map[string]bool), v}
}

type unusedVisitor struct {
	used        map[*ast.Object]bool
	usedImports map[string]bool
	visitor     Visitor
}

func (v *unusedVisitor) VisitStmt(*ast.Scope, ast.Stmt) ScopeVisitor {
	return v
}

func lookup(scope *ast.Scope, name string) *ast.Object {
	for scope != nil {
		if obj := scope.Lookup(name); obj != nil {
			return obj
		}
		scope = scope.Outer
	}
	return nil
}

func (v *unusedVisitor) VisitExpr(scope *ast.Scope, expr ast.Expr) ScopeVisitor {
	switch expr := expr.(type) {
	case *ast.Ident:
		if def := lookup(scope, expr.Name); def != nil {
			v.used[def] = true
		} else {
			v.usedImports[expr.Name] = true
		}
	case *ast.SelectorExpr:
		v.VisitExpr(scope, expr.X)
		return nil
	}
	return v
}

func (v *unusedVisitor) ExitScope(scope *ast.Scope) ScopeVisitor {
	for _, obj := range scope.Objects {
		if !v.used[obj] {
			v.visitor.UnusedObj(obj)
		}
	}
	return v
}

package main

import (
	"github.com/elazarl/gosloppy/imports"
	"go/ast"
)

type Visitor interface {
	UnusedObj(obj *ast.Object)
	UnusedImport(imp *ast.ImportSpec)
}

func anonymousImport(name *ast.Ident) bool {
	return name != nil && (name.Name == "_" || name.Name == ".")
}

func NewUnusedVisitor(v Visitor) *UnusedVisitor {
	return &UnusedVisitor{make(map[*ast.Object]ast.Node), make(map[string]bool), v}
}

type UnusedVisitor struct {
	Used        map[*ast.Object]ast.Node
	UsedImports map[string]bool
	Visitor     Visitor
}

func (v *UnusedVisitor) VisitStmt(*ast.Scope, ast.Stmt, ast.Node) ScopeVisitor {
	return v
}

func (v *UnusedVisitor) VisitExpr(scope *ast.Scope, expr ast.Expr, parent ast.Node) ScopeVisitor {
	switch expr := expr.(type) {
	case *ast.Ident:
		if def := Lookup(scope, expr.Name); def != nil {
			v.Used[def] = parent
		} else {
			v.UsedImports[expr.Name] = true
		}
	case *ast.SelectorExpr:
		v.VisitExpr(scope, expr.X, parent)
		return nil
	}
	return v
}

func (v *UnusedVisitor) ExitScope(scope *ast.Scope, node ast.Node, last bool) ScopeVisitor {
	for _, obj := range scope.Objects {
		if v.Used[obj] == nil {
			v.Visitor.UnusedObj(obj)
		}
	}
	if file, ok := node.(*ast.File); ok {
		for _, imp := range file.Imports {
			name := imports.GetNameOrGuess(imp)
			if !v.UsedImports[name] && !anonymousImport(imp.Name) {
				v.Visitor.UnusedImport(imp)
			}
		}
	}
	return v
}

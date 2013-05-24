package visitors

import (
	"go/ast"

	"github.com/elazarl/gosloppy/imports"
	"github.com/elazarl/gosloppy/scopes"
)

type Visitor interface {
	UnusedObj(obj *ast.Object, parent ast.Node)
	UnusedImport(imp *ast.ImportSpec)
}

func anonymousImport(name *ast.Ident) bool {
	return name != nil && (name.Name == "_" || name.Name == ".")
}

func NewUnusedVisitor(v Visitor) *UnusedVisitor {
	return &UnusedVisitor{make(map[*ast.Object]bool), make(map[*ast.Ident]bool), make(map[string]bool), v}
}

type UnusedVisitor struct {
	Used        map[*ast.Object]bool
	Irrelevant  map[*ast.Ident]bool
	UsedImports map[string]bool
	Visitor     Visitor
}

func (v *UnusedVisitor) VisitStmt(*ast.Scope, ast.Stmt) scopes.Visitor {
	return v
}

func (v *UnusedVisitor) VisitDecl(*ast.Scope, ast.Decl) scopes.Visitor {
	return v
}

func (v *UnusedVisitor) VisitExpr(scope *ast.Scope, expr ast.Expr) scopes.Visitor {
	switch expr := expr.(type) {
	case *ast.Ident:
		if v.Irrelevant[expr] {
			return v
		}
		if def := scopes.Lookup(scope, expr.Name); def != nil {
			v.Used[def] = true
		} else {
			v.UsedImports[expr.Name] = true
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

func (v *UnusedVisitor) ExitScope(scope *ast.Scope, node ast.Node, last bool) scopes.Visitor {
	for _, obj := range scope.Objects {
		if !v.Used[obj] {
			v.Visitor.UnusedObj(obj, node)
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

package visitors

import (
	"go/ast"

	"github.com/elazarl/gosloppy/imports"
	"github.com/elazarl/gosloppy/scopes"
)

// UnusedVisitor visits unused idientifier and unused import statements
type UnusedVisitor interface {
	UnusedObj(obj *ast.Object, parent ast.Node)
	UnusedImport(imp *ast.ImportSpec)
}

func anonymousImport(name *ast.Ident) bool {
	return name != nil && (name.Name == "_" || name.Name == ".")
}

// NewUnused returns a scopes.Visitor that visits unused identifiers
func NewUnused(v UnusedVisitor) *Unused {
	return &Unused{make(map[*ast.Object]bool), make(map[*ast.Ident]bool), make(map[string]bool), v}
}

// Unused is a scopes.Visitor that would visit all unused variables with the
// given UnusedVisitor
type Unused struct {
	Used        map[*ast.Object]bool
	Irrelevant  map[*ast.Ident]bool
	UsedImports map[string]bool
	Visitor     UnusedVisitor
}

func (v *Unused) VisitStmt(*ast.Scope, ast.Stmt) scopes.Visitor {
	return v
}

func (v *Unused) VisitDecl(*ast.Scope, ast.Decl) scopes.Visitor {
	return v
}

func (v *Unused) VisitExpr(scope *ast.Scope, expr ast.Expr) scopes.Visitor {
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

func (v *Unused) ExitScope(scope *ast.Scope, node ast.Node, last bool) scopes.Visitor {
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

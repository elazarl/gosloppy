package scopes_test

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"

	"github.com/elazarl/gosloppy/scopes"
)

type WarnShadow struct {
	*token.FileSet
}

func (w WarnShadow) VisitExpr(scope *ast.Scope, expr ast.Expr) scopes.Visitor {
	return w
}

func (w WarnShadow) VisitStmt(scope *ast.Scope, stmt ast.Stmt) scopes.Visitor {
	if stmt, ok := stmt.(*ast.DeclStmt); ok {
		if decl, ok := stmt.Decl.(*ast.GenDecl); ok {
			for _, spec := range decl.Specs {
				if spec, ok := spec.(*ast.ValueSpec); ok {
					for _, name := range spec.Names {
						if scopes.Lookup(scope.Outer, name.Name) != nil {
							fmt.Print(w.Position(name.Pos()).Line, ": Warning, shadowed ", name, "\n")
						}
					}
				}
			}
		}
	}
	return w
}

func (w WarnShadow) VisitDecl(scope *ast.Scope, decl ast.Decl) scopes.Visitor {
	return w
}

func (w WarnShadow) ExitScope(scope *ast.Scope, parent ast.Node, last bool) scopes.Visitor {
	return w
}

func Example() {
	file, fset := parse(`package main
		func init() {
			i := 1
			if true {
				var i = 2
			}
		}`)
	scopes.WalkFile(WarnShadow{fset}, file)
	// Output: 5: Warning, shadowed i
}

func parse(code string) (*ast.File, *token.FileSet) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "", code, parser.DeclarationErrors)
	if err != nil {
		panic("Cannot parse code:" + err.Error())
	}
	return file, fset
}

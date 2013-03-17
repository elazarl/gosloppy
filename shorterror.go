package main

import (
	"fmt"
	"go/ast"

	"github.com/elazarl/gosloppy/patch"
)

type ShortError struct {
	file    *patch.PatchableFile
	patches *patch.Patches
	stmt    ast.Stmt
	block   *ast.BlockStmt
	tmpvar  int
}

func NewShortError(file *patch.PatchableFile) *ShortError {
	patches := make(patch.Patches, 0, 10)
	return &ShortError{file, &patches, nil, nil, 0}
}

func (v *ShortError) Patches() patch.Patches {
	return *v.patches
}

func (v *ShortError) tempVar(stem string, scope *ast.Scope) string {
	for ; v.tmpvar < 10*1000; v.tmpvar++ {
		name := fmt.Sprint(stem, v.tmpvar)
		if Lookup(scope, name) == nil {
			v.tmpvar++
			return name
		}
	}
	panic(">100,000 temporary variables used. Either the code is crazy, or I am.")
}

func (v *ShortError) VisitExpr(scope *ast.Scope, expr ast.Expr) ScopeVisitor {
	if expr, ok := expr.(*ast.CallExpr); ok {
		if fun, ok := expr.Fun.(*ast.Ident); ok && fun.Name == "must" {
			mustexpr := v.file.Slice(expr.Lparen+1, expr.Rparen)
			tmpVar, tmpErr := v.tempVar("tmp_", scope), v.tempVar("err_", scope)
			*v.patches = append(*v.patches, patch.Insert(v.stmt.Pos(),
				fmt.Sprint(tmpVar, ", ", tmpErr, " := ", mustexpr, "; ",
					"if ", tmpErr, " != nil {panic(", tmpErr, ")};")))
			*v.patches = append(*v.patches, patch.Replace(expr, tmpVar))
		}
	}
	return v
}

func (v *ShortError) VisitStmt(scope *ast.Scope, stmt ast.Stmt) ScopeVisitor {
	v.stmt = stmt
	if stmt, ok := stmt.(*ast.BlockStmt); ok {
		return &ShortError{v.file, v.patches, v.stmt, stmt, 0}
	}
	return v
}

func (v *ShortError) ExitScope(scope *ast.Scope, node ast.Node, last bool) ScopeVisitor {
	return v
}

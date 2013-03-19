package main

import (
	"fmt"
	"go/ast"
	"go/token"

	"github.com/elazarl/gosloppy/patch"
)

type ShortError struct {
	file    *patch.PatchableFile
	patches *patch.Patches
	// This stmt is a bit evil, it's the last stmt before this visit
	// I made a premature optimization, and to prevent large allocation
	// I changed the stmt inline during visiting.
	stmt    ast.Stmt
	block   *ast.BlockStmt
	tmpvar  int
	initTxt *[]byte
}

func NewShortError(file *patch.PatchableFile) *ShortError {
	patches := make(patch.Patches, 0, 10)
	return &ShortError{file, &patches, nil, nil, 0, new([]byte)}
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

var MustKeyword = "must"

// Yeah yeah, O(n^2) in the worst case. If you use so much must
// YOU are the worst case.
func findinit(file *ast.File) *ast.FuncDecl {
	for _, d := range file.Decls {
		if d, ok := d.(*ast.FuncDecl); ok && d.Name.Name == "init" {
			return d
		}
	}
	return nil
}

func afterImports(file *ast.File) token.Pos {
	if len(file.Imports) > 0 {
		return file.Imports[len(file.Imports)-1].End()
	}
	return file.Name.End()
}

func (v *ShortError) addToInit(txt string) {
	*v.initTxt = append(*v.initTxt, txt...)
}

func (v *ShortError) VisitExpr(scope *ast.Scope, expr ast.Expr) ScopeVisitor {
	if expr, ok := expr.(*ast.CallExpr); ok {
		if fun, ok := expr.Fun.(*ast.Ident); ok && fun.Name == MustKeyword {
			tmpVar, tmpErr := v.tempVar("tmp_", scope), v.tempVar("err_", scope)
			mustexpr := v.file.Slice(expr.Lparen+1, expr.Rparen)
			if v.block == nil {
				// if in top level decleration
				v.addToInit("if " + tmpErr + " != nil {panic(" + tmpErr + ")};")
				*v.patches = append(*v.patches,
					patch.Replace(expr, tmpVar),
					patch.Insert(afterImports(v.file.File), ";var "+tmpVar+", "+tmpErr+" = "+mustexpr))
			} else {
				*v.patches = append(*v.patches, patch.Insert(v.stmt.Pos(),
					fmt.Sprint("var ", tmpVar, ", ", tmpErr, " = ", mustexpr, "; ",
						"if ", tmpErr, " != nil {panic(", tmpErr, ")};")))
				*v.patches = append(*v.patches, patch.Replace(expr, tmpVar))
			}
		}
	}
	return v
}

func (v *ShortError) VisitDecl(scope *ast.Scope, decl ast.Decl) ScopeVisitor {
	if decl, ok := decl.(*ast.GenDecl); ok {
		var _ = decl
		return nil
	}
	return v
}

func (v *ShortError) VisitStmt(scope *ast.Scope, stmt ast.Stmt) ScopeVisitor {
	v.stmt = stmt
	switch stmt := stmt.(type) {
	case *ast.BlockStmt:
		return &ShortError{v.file, v.patches, v.stmt, stmt, 0, new([]byte)}
	case *ast.AssignStmt:
		if len(stmt.Rhs) != 1 {
			return v
		}
		if rhs, ok := stmt.Rhs[0].(*ast.CallExpr); ok {
			if fun, ok := rhs.Fun.(*ast.Ident); ok && fun.Name == MustKeyword {
				tmpVar := v.tempVar("assignerr_", scope)
				*v.patches = append(*v.patches,
					patch.Insert(stmt.TokPos, ", "+tmpVar+" "),
					patch.Replace(fun, ""),
					patch.Insert(stmt.End(),
						"; if "+tmpVar+" != nil "+
							"{ panic("+tmpVar+") };"),
				)
				for _, arg := range rhs.Args {
					v.VisitExpr(scope, arg)
				}
				return nil
			}
		}
	}
	return v
}

func (v *ShortError) ExitScope(scope *ast.Scope, node ast.Node, last bool) ScopeVisitor {
	if node, ok := node.(*ast.File); ok && len(*v.initTxt) > 0 {
		if init := findinit(node); init != nil {
			*v.patches = append(*v.patches, patch.Insert(init.Body.Lbrace+1, string(*v.initTxt)))
		} else {
			*v.patches = append(*v.patches, patch.Insert(afterImports(node),
				";func init() {"+string(*v.initTxt)+"}"))
		}
	}
	return v
}

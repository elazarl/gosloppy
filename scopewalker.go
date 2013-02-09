package main

import (
	"go/ast"
	"go/token"
	"log"
)

type ScopeVisitor interface {
	VisitExpr(scope *ast.Scope, expr ast.Expr) (w ScopeVisitor)
	VisitStmt(scope *ast.Scope, stmt ast.Stmt) (w ScopeVisitor)
	ExitScope(scope *ast.Scope) (w ScopeVisitor)
}

/*
func WalkStmt(v ScopeVisitor, stmt ast.Stmt, scope *ast.Scope) (newscope *ast.Scope) {
	w := v.VisitStmt(scope, stmt)
	if w==nil {
		return
	}
	newscope = scope
	switch stmt := stmt.(type) {
	case *ast.RangeStmt:
		if stmt.Tok == token.ASSIGN {
			// in assignment statement (eg, a, b := f()), all componente must be *ast.Ident
			newscope = ast.NewScope(scope)
			if stmt.Key!=nil {
				scope.Insert(stmt.Key.(*ast.Ident).Obj)
				WalkExpr(w, stmt.Key, scope)
			}
			if stmt.Value!=nil {
				scope.Insert(stmt.Value.(*ast.Ident).Obj)
				WalkExpr(w, stmt.Value, scope)
			}
			WalkBlock(w, stmt.Body, scope)
		}
	case *ast.ForStmt:
		newscope = ast.NewScope(scope)
		if stmt.Init != nil {
		}
		if stmt.Init != nil {
			WalkStmt(w, stmt.Init, scope)
		}
		if stmt.Cond != nil {
			WalkExpr(w, stmt.Cond, scope)
		}
		if stmt.Post != nil {
			WalkStmt(w, stmt.Post, scope)
		}
		WalkBlock(w, stmt.Body, scope)
	}
	w.PostVisit(scope, stmt)
	return
}

func WalkBlock(v ScopeVisitor, block *ast.BlockStmt, funscope *ast.Scope) {
	w := v.PreVisit(funscope, block)
	if w==nil {
		return
	}
	scope := ast.NewScope(funscope)
	for _, stmt := range block.List {
		WalkStmt(w, stmt, scope)
	}
	w.PostVisit(funscope, block)
}
*/

func WalkStmt(v ScopeVisitor, stmt ast.Stmt, scope *ast.Scope) (newscope *ast.Scope) {
	newscope = scope
	switch stmt := stmt.(type) {
	case *ast.AssignStmt:
		if stmt.Tok == token.DEFINE {
			newscope = ast.NewScope(scope)
			for _, expr := range stmt.Rhs {
				v = v.VisitExpr(scope, expr)
			}
			for _, expr := range stmt.Lhs {
				newscope.Insert(expr.(*ast.Ident).Obj)
			}
		} else {
			v.VisitStmt(scope, stmt)
		}
	case *ast.DeclStmt:
		switch decl := stmt.Decl.(type) {
		case *ast.GenDecl:
			for _, spec := range decl.Specs {
				newscope = ast.NewScope(newscope)
				switch spec := spec.(type) {
				case *ast.TypeSpec:
					newscope.Insert(spec.Name.Obj)
					v = v.VisitExpr(scope, spec.Type)
				case *ast.ValueSpec:
					for _, name := range spec.Names {
						newscope.Insert(name.Obj)
					}
					for _, value := range spec.Values {
						v = v.VisitExpr(scope, value)
					}
				default:
					panic("cannot have an import in a statement (or so I hope)")
				}
			}
		default:
			panic("declstmt")
		}
		/*
	case *ast.IfStmt:
		scope := ast.NewScope(scope)
		if stmt.Init != nil {
			scope = WalkStmt(v, stmt.Init, scope)
		}
		v = v.VisitExpr(scope, stmt.Cond)
		bodyscope := ast.NewScope(newscope)
		WalkStmt(v, stmt.Body, bodyscope)
		v = v.ExitScope(bodyscope)
		if stmt.Else != nil {
			elsescope := ast.NewScope(scope)
			WalkStmt(v, stmt.Else, elsescope)
			v = v.ExitScope(elsescope)
		}
		*/
	case *ast.BlockStmt:
		stmtscope := ast.NewScope(scope)
		for _, s := range stmt.List {
			stmtscope = WalkStmt(v, s, stmtscope)
		}
		for stmtscope != scope {
			if stmtscope == nil {
				log.Fatal("Oh my")
			}
			v.ExitScope(stmtscope)
			stmtscope = stmtscope.Outer
		}
	}
	return
}

func WalkFile(v ScopeVisitor, file *ast.File) {
	if v==nil {
		return
	}
	for _, d := range file.Decls {
		switch d := d.(type) {
		case *ast.FuncDecl:
			scope := ast.NewScope(file.Scope)
			if d.Recv!=nil {
				scope.Insert(d.Recv.List[0].Names[0].Obj)
			}
			for _, fields := range d.Type.Params.List {
				for _, p := range fields.Names {
					scope.Insert(p.Obj)
				}
			}
			WalkStmt(v, d.Body ,scope)
			v.ExitScope(scope)
		case *ast.GenDecl:
		}
	}
	v.ExitScope(file.Scope)
}

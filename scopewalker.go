package main

import (
	"go/ast"
	"go/token"
)

type ScopeVisitor interface {
	VisitExpr(scope *ast.Scope, expr ast.Expr) (w ScopeVisitor)
	VisitStmt(scope *ast.Scope, stmt ast.Stmt) (w ScopeVisitor)
	ExitScope(scope *ast.Scope) (w ScopeVisitor)
}


// TODO(elazar): scope of func literal
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
			panic("only GenDecl can appear in statement")
		}
	case *ast.IfStmt:
		inner := scope
		if stmt.Init != nil {
			inner = WalkStmt(v, stmt.Init, inner)
		}
		v = v.VisitExpr(inner, stmt.Cond)
		WalkStmt(v, stmt.Body, inner)
		if stmt.Else != nil {
			WalkStmt(v, stmt.Else, inner)
		}
		exitScopes(v, inner, scope)
	case *ast.ForStmt:
		inner := scope
		if stmt.Init != nil {
			inner = WalkStmt(v, stmt.Init, inner)
		}
		if stmt.Cond != nil {
			v = v.VisitExpr(inner, stmt.Cond)
		}
		if stmt.Post != nil {
			WalkStmt(v, stmt.Post, scope)
		}
		WalkStmt(v, stmt.Body, scope)
		exitScopes(v, inner, scope)
	case *ast.RangeStmt:
		inner := scope
		if stmt.Tok == token.ASSIGN {
			v = v.VisitExpr(inner, stmt.Key)
			v = v.VisitExpr(inner, stmt.Value)
		} else if stmt.Tok == token.DEFINE {
			inner = ast.NewScope(inner)
			// TODO(elazar): make sure Scope is smart enough not to insert _
			inner.Insert(stmt.Key.(*ast.Ident).Obj)
			inner.Insert(stmt.Value.(*ast.Ident).Obj)
		} else {
			panic("range statement must have := or = token")
		}
		WalkStmt(v, stmt.Body, scope)
		exitScopes(v, inner, scope)
	case *ast.CaseClause:
		inner := ast.NewScope(scope)
		for _, expr := range stmt.List {
			v = v.VisitExpr(scope, expr)
		}
		for _, s := range stmt.Body {
			inner = WalkStmt(v, s, inner)
		}
		exitScopes(v, inner, scope)
	case *ast.SwitchStmt:
		inner := scope
		if stmt.Init != nil {
			inner = WalkStmt(v, stmt.Init, inner)
		}
		v = v.VisitExpr(inner, stmt.Tag)
		WalkStmt(v, stmt.Body, inner)
		exitScopes(v, inner, scope)
	case *ast.TypeSwitchStmt:
		panic("TODO: not yet implemented")
	case *ast.SelectStmt:
		panic("TODO: not yet implemented")
	case *ast.BlockStmt:
		inner := ast.NewScope(scope)
		for _, s := range stmt.List {
			inner = WalkStmt(v, s, inner)
		}
		exitScopes(v, inner, scope)
	}
	return
}

func exitScopes(v ScopeVisitor, inner, limit *ast.Scope) {
		for inner != limit {
			if inner == nil {
				panic("exitScopes must be bounded")
			}
			v.ExitScope(inner)
			inner = inner.Outer
		}
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

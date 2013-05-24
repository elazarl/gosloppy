package visitors

import (
	"go/ast"

	"github.com/elazarl/gosloppy/scopes"
)

type MultiVisitor struct {
	*cow
}

func NewMultiVisitor(v ...scopes.Visitor) MultiVisitor {
	return MultiVisitor{newCow(v...)}
}

func (v MultiVisitor) AllNil() bool {
	for _, elt := range v.ar {
		if elt != nil {
			return false
		}
	}
	return true
}

func (v MultiVisitor) VisitExpr(scope *ast.Scope, expr ast.Expr) scopes.Visitor {
	for i, w := range v.ar {
		if w == nil {
			continue
		}
		v = MultiVisitor{v.Set(i, w.VisitExpr(scope, expr))}
	}
	if v.AllNil() {
		return nil
	}
	return v
}

func (v MultiVisitor) VisitStmt(scope *ast.Scope, stmt ast.Stmt) scopes.Visitor {
	for i, w := range v.ar {
		if w == nil {
			continue
		}
		v = MultiVisitor{v.Set(i, w.VisitStmt(scope, stmt))}
	}
	if v.AllNil() {
		return nil
	}
	return v
}

func (v MultiVisitor) VisitDecl(scope *ast.Scope, decl ast.Decl) scopes.Visitor {
	for i, w := range v.ar {
		if w == nil {
			continue
		}
		v = MultiVisitor{v.Set(i, w.VisitDecl(scope, decl))}
	}
	if v.AllNil() {
		return nil
	}
	return v
}

func (v MultiVisitor) ExitScope(scope *ast.Scope, node ast.Node, last bool) scopes.Visitor {
	for i, w := range v.ar {
		if w == nil {
			continue
		}
		v = MultiVisitor{v.Set(i, w.ExitScope(scope, node, last))}
	}
	if v.AllNil() {
		return nil
	}
	return v
}

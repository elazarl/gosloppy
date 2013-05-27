package visitors

import (
	"go/ast"

	"github.com/elazarl/gosloppy/scopes"
)

// MultiVisitor will apply multiple visitors to all ast nodes, in a single scopes.Walk* run
type MultiVisitor struct {
	*cow
}

// NewMultiVisitor returns a new MultiVisitor applying all visitors vs
func NewMultiVisitor(vs ...scopes.Visitor) MultiVisitor {
	return MultiVisitor{newCow(vs...)}
}

// AllNil returns whether or not all visitors of the MultiVisitor are nil.
// This can happen since a Visitor can modify itself by returning a different
// Visitor in the Visit* function.
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

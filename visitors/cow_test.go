package visitors

import (
	"fmt"
	"go/ast"
	"testing"

	"github.com/elazarl/gosloppy/scopes"
)

type vis int

func (v vis) ExitScope(*ast.Scope, ast.Node, bool) scopes.Visitor { return v }
func (v vis) VisitExpr(*ast.Scope, ast.Expr) scopes.Visitor       { return v }
func (v vis) VisitStmt(*ast.Scope, ast.Stmt) scopes.Visitor       { return v }
func (v vis) VisitDecl(*ast.Scope, ast.Decl) scopes.Visitor       { return v }

func expect(t *testing.T, vs *cow, expected ...int) {
	intvisitor := []int{}
	for _, v := range vs.ar {
		intvisitor = append(intvisitor, int(v.(vis)))
	}
	exp, act := fmt.Sprint(expected), fmt.Sprint(intvisitor)
	if exp != act {
		t.Errorf("Expected %s got %s", exp, act)
	}
}

func TestSimpleChange(t *testing.T) {
	c := newCow(vis(0), vis(1), vis(2))
	expect(t, c, 0, 1, 2)
	expect(t, c.Set(0, vis(5)), 5, 1, 2)
	expect(t, c.Set(0, vis(5)).Set(2, vis(100)), 5, 1, 100)
	if c.Set(0, vis(5)) != c.Set(0, vis(5)) {
		t.Error("Does not cache equal arrays")
	}
}

package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"sort"
	"testing"
)

type VerifyVisitor []string

func (v VerifyVisitor) rest() []string {
	for i, s := range v {
		if s != "" {
			return v[i:]
		}
	}
	return []string{}
}

func (v VerifyVisitor) pop() string {
	for i, s := range v {
		if s != "" {
			v[i] = ""
			return s
		}
	}
	return ""
}

func tostring(n ast.Node) string {
	buf := new(bytes.Buffer)
	printer.Fprint(buf, token.NewFileSet(), n)
	return buf.String()
}

var t *testing.T

func setT(globalT *testing.T) {
	t = globalT
}

func (v VerifyVisitor) verify(n ast.Node) {
	exp := v.pop()
	if tostring(n)!=exp {
		t.Error("Expected to visit", exp, "got", tostring(n))
	}
}

func (v VerifyVisitor) VisitStmt(scope *ast.Scope, expr ast.Stmt) ScopeVisitor {
	v.verify(expr)
	return v
}

func (v VerifyVisitor) VisitExpr(scope *ast.Scope, expr ast.Expr) ScopeVisitor {
	v.verify(expr)
	return v
}

func (v VerifyVisitor) ExitScope(scope *ast.Scope) ScopeVisitor {
	return v
}


var simpleVisitorTestCases = []struct {
	body string
	expected []string
} {
	{`
		a := 1
		var z, w int = 2
		var (
			x = 3
			y int
		)
		a = b`,
		[]string{"1", "2", "3", "a = b"},
	},
	{`
		var (
			x = 3
			y int = x
		)`,
		[]string{"3", "x"},
	},
	{`
		x := 1
		{
			x = 2 + 1
		}`,
		[]string{"1", "x = 2 + 1"},
	},
}

func parse(code string, t *testing.T) (*ast.File, *token.FileSet) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "", code, parser.DeclarationErrors)
	if err != nil {
		t.Fatal("Cannot parse code", err)
	}
	return file, fset
}

func TestSimpleVisitor(t *testing.T) {
	setT(t)
	for _, c := range simpleVisitorTestCases {
		file, _ := parse(`
		package main
		func f() {
	            `+c.body+`
		}`, t)
		visitor := VerifyVisitor(c.expected)
		WalkFile(visitor, file)
		if len(visitor.rest())!= 0 {
			t.Error("not all expected values consumed", visitor.rest())
		}
		//ScopeVisitor(w, file)
	}
}

type VerifyExitScope struct {
	v [][]string
}

func (pv *VerifyExitScope) pop() []string {
	v := pv.v
	if len(v)==0 {
		return nil
	}
	last := v[len(v)-1]
	pv.v = v[:len(v)-1]
	sort.Strings(last)
	return last
}

func scopeNames(scope *ast.Scope) (names []string) {
	for _, obj := range scope.Objects {
		names = append(names, obj.Name)
	}
	sort.Strings(names)
	return
}

func (v *VerifyExitScope) VisitStmt(scope *ast.Scope, expr ast.Stmt) ScopeVisitor {
	return v
}

func (v *VerifyExitScope) VisitExpr(scope *ast.Scope, expr ast.Expr) ScopeVisitor {
	return v
}

func (v *VerifyExitScope) ExitScope(scope *ast.Scope) ScopeVisitor {
	expected := v.pop()
	if fmt.Sprint(expected) != fmt.Sprint(scopeNames(scope)) {
		t.Error("Expected", expected, "got", scopeNames(scope))
	}
	return v
}

func TestExitScope(t *testing.T) {
	setT(t)
	for _, c := range ScopeOrderTestCases {
		file, _ := parse(c.body, t)
		expected := &VerifyExitScope{c.scopes}
		WalkFile(expected, file)
		if len(expected.v) > 0 {
			t.Error("Unsatisfied expected scopes", expected)
		}
	}
}

var ScopeOrderTestCases = []struct {
	body string
	scopes [][]string
} {
	{`
		package main
		func f(a int) {
		}
	`,
	[][]string{ {"f"}, {"a"} },
	},
}

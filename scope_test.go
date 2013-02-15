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
	if tostring(n) != exp {
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
	body     string
	expected []string
}{
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
	return
	setT(t)
	for _, c := range simpleVisitorTestCases {
		file, _ := parse(`
		package main
		func f() {
	            `+c.body+`
		}`, t)
		visitor := VerifyVisitor(c.expected)
		WalkFile(visitor, file)
		if len(visitor.rest()) != 0 {
			t.Error("not all expected values consumed", visitor.rest())
		}
		//ScopeVisitor(w, file)
	}
}

type VerifyExitScope struct {
	v  [][]string
	t  *testing.T
	ix int
}

func (pv *VerifyExitScope) pop() []string {
	v := pv.v
	if len(v) == 0 {
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
		v.t.Error("Expected", append(v.v, expected), "got", scopeNames(scope), "test case", v.ix)
	}
	return v
}

func TestExitScope(t *testing.T) {
	for i, c := range ScopeOrderTestCases {
		file, _ := parse(c.body, t)
		expected := &VerifyExitScope{c.scopes, t, i}
		WalkFile(expected, file)
		if len(expected.v) > 0 {
			t.Error("Unsatisfied expected scopes", expected)
		}
	}
}

// TODO(elazar): think and enable visiting empty block statements
var ScopeOrderTestCases = []struct {
	body   string
	scopes [][]string
}{
	{`
		package main
		func f(a int) {
		}
	`,
		[][]string{{"f"}, {"a"}, {}},
	},
	{`
		package main
		func f(a int) {
			{
			}
		}
	`,
		[][]string{{"f"}, {"a"}, {}, {}},
	},
	{`
		package main
		func f(funcscope int) {
			type T int
		}
	`,
		[][]string{{"f"}, {"funcscope"}, {}, {"T"}},
	},
	{`
		package main
		/* empty scope of func's arguments */
		func f() {
			/* empty block scope */
			var x, y int
		}
	`,
		[][]string{{"f"}, {}, {}, {"x", "y"}},
	},
	{`
		package main
		/* empty scope of func's arguments */
		func f() {
			/* empty block scope */
			var x, y int
			x = y
		}
	`,
		[][]string{{"f"}, {}, {}, {"x", "y"}},
	},
	{`
		package main
		func f() {
			var (
				x int
				y int
			)
			x = y
		}
	`,
		[][]string{{"f"}, {}, {}, {"x"}, {"y"}},
	},
	{`
		package main
		func f() {
			a, b := 1, 2
		}
	`,
		[][]string{{"f"}, {}, {}, {"a", "b"}},
	},
	{`
		package main
		func f() {
			a := 1
			b := 1
		}
	`,
		[][]string{{"f"}, {}, {}, {"a"}, {"b"}},
	},
	{`
		package main
		/* empty func scope */
		func f() {
			/* emtpy block */
			if a == 1 {
				/* emtpy block */
			}
		}
	`,
		[][]string{{"f"}, {}, {}, {}},
	},
	{`
		package main
		/* empty func scope */
		func f(funscope int) {
			/* emtpy block */
			if a == 1 {
				/* emtpy block */
				a := 1
			}
		}
	`,
		[][]string{{"f"}, {"funscope"}, {}, {}, {"a"}},
	},
	{`
		package main
		func f(funscope int) {
			if ifscope := 1; a == 1 {
				/* emtpy block */
				x := 1
			}
		}
	`,
		[][]string{{"f"}, {"funscope"}, {}, {"ifscope"}, {}, {"x"}},
	},
	{`
		package main
		func f(funscope int) {
			if ifscope := 1; a == 1 {
				x := 1
			} else {
				/* note: extra empty scope for stmtblocck */
				elsescope := 1
			}
		}
	`,
		[][]string{{"f"}, {"funscope"}, {}, {"ifscope"}, {}, {"elsescope"}, {}, {"x"}},
	},
	{`
		package main
		func f(funscope int) {
			if ifscope := 1; a == 1 {
				x := 1
			} else if nestedifscope := 1; 1 == 1 {
				/* note: extra empty scope for stmtblocck */
				elsescope := 1
			}
		}
	`,
		[][]string{{"f"}, {"funscope"}, {}, {"ifscope"}, {"nestedifscope"}, {}, {"elsescope"}, {}, {"x"}},
	},
	{`
		package main
		func f(funscope int) {
			/* empty stmt block */
			for 1 == 1 {
				/* empty stmt block */
				forscope := 1
			}
		}
	`,
		[][]string{{"f"}, {"funscope"}, {}, {}, {"forscope"}},
	},
	{`
		package main
		func f(funscope int) {
			/* empty stmt block */
			for i := 1; 1 == 1; {
				/* empty stmt block */
				forscope := 1
			}
		}
	`,
		[][]string{{"f"}, {"funscope"}, {}, {"i"}, {}, {"forscope"}},
	},
	{`
		package main
		func f(funscope int) {
			/* empty stmt block */
			for k, v := range m {
				/* empty stmt block */
				forscope := 1
			}
		}
	`,
		[][]string{{"f"}, {"funscope"}, {}, {"k", "v"}, {}, {"forscope"}},
	},
	{`
		package main
		func f(funscope int) {
			/* empty stmt block */
			for k, v = range m {
				/* empty stmt block */
				forscope := 1
			}
		}
	`,
		[][]string{{"f"}, {"funscope"}, {}, {}, {"forscope"}},
	},
	{`
		package main
		func f(funscope int) {
			/* empty stmt block */
			switch a, b := f(); a {
			/* empty stmt block */
			case 1:
				/* empty case block */
				switchscope := 1
			}
		}
	`,
		[][]string{{"f"}, {"funscope"}, {}, {"a", "b"}, {}, {}, {"switchscope"}},
	},
	{`
		package main
		func f(funscope int) {
			/* empty stmt block */
			switch a, b := f(); a := a.(type) {
			/* empty stmt block */
			case int:
				/* empty case block */
				switchscope := 1
			case string:
				/* empty case block */
				a = a
			}
		}
	`,
		[][]string{{"f"}, {"funscope"}, { /* func block */},
			{"a", "b"}, {"a"}, { /* switch */},
			{ /* case string: */},
			{ /* case int */}, {"switchscope"}},
	},
	{`
		package main
		func f(funscope int) {
			/* empty stmt block */
			select {
			/* empty stmt block */
			case i := <- ch:
				/* empty case block */
				casescope := 1
			}
		}
	`,
		[][]string{{"f"}, {"funscope"}, { /* func block stmt */},
			{ /* select block */},
			{"i" /* case block stmt*/}, {"casescope"}},
	},
	{`
		package main
		func f(funscope int) {
			a = func(funclit int) {
				infunclit := 1
			}
		}
	`,
		[][]string{{"f"}, {"funscope"}, { /* func block stmt */}, {"funclit"}, { /* funclit body*/}, {"infunclit"}},
	},
	{`
		package main
		func init() {
			var _ = 1
		}
	`,
		[][]string{ {}, },
	},
	{`
		package main
		func f(funscope int) {
			init := func(funclitscope int) {}
		}
	`,
		[][]string{{"f"}, {"funscope"}, {}, {"init"}, {"funclitscope"}, {/* funclit stmt block */}},
	},
}

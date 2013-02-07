package main

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
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
}

func TestSimpleVisitor(t *testing.T) {
	setT(t)
	for _, c := range simpleVisitorTestCases {
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, "", `
		package main
		func f() {
	            `+c.body+`
		}`, parser.DeclarationErrors)
		//ast.Print(fset, file.Decls[0])
		if err!=nil {
			t.Fatal("Can't parser file", err)
		}
		visitor := VerifyVisitor(c.expected)
		WalkFile(visitor, file)
		if len(visitor.rest())!= 0 {
			t.Error("not all expected values consumed", visitor.rest())
		}
		//ScopeVisitor(w, file)
	}
}


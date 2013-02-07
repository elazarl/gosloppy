package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"os"
	"reflect"
	"strings"
)

func fatalOnErr(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func isgofile(info os.FileInfo) bool {
	return strings.HasSuffix(info.Name(), ".go")
}

var i = 100

type VarSet map[ast.Decl]bool

func (vs VarSet) Contains(decl ast.Decl) bool {
	_, ok := vs[decl]
	return ok
}
func (vs VarSet) Add(decl ast.Decl) bool {
	_, ok := vs[decl]
	vs[decl] = true
	return ok
}

var _ = reflect.Array

func prtype(obj interface{}) {
	fmt.Println(reflect.TypeOf(obj))
	fmt.Printf("%+#v\n", obj)
}

type SV bool

func main() {
	fmt.Println("main")
	fset := token.NewFileSet()
	fset = fset
	var a = 1
	a = a
	if a==0 {
	} else {
		a = 1
		a = 11
	}
	pkgs, err := parser.ParseDir(fset, os.Args[1], isgofile, parser.DeclarationErrors)
	fatalOnErr(err)
	for _, p := range pkgs {
		for fname, tree := range p.Files {
			fmt.Println("file", fname)
			var _ = tree
			//WalkFile(SV(true), tree)
		}
	}
	/*for k, p := range pkgs {
		fmt.Println("In", k)
		for fname, tree := range p.Files {
			fmt.Println("File", fname)
			fmt.Println(tree.Scope)
			fmt.Printf("%+#v\n", ast.ExprStmt{&ast.CallExpr{}})
			prtype(&ast.CallExpr{Fun: &ast.SelectorExpr{X: &ast.Ident{Name: "fmt"}, Sel: &ast.Ident{Name: "Println"}}, Args:[]ast.Expr{&ast.BasicLit{Kind: token.STRING, Value: `"main"`}}})
			prtype(tree.Scope.Lookup("main").Decl.(*ast.FuncDecl).Body.List[0].(*ast.ExprStmt).X.(*ast.CallExpr).Fun.(*ast.SelectorExpr).Sel)
		}
	}*/
}

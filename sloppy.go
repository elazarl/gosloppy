package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
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

func prtype(obj interface{}) {
	fmt.Println(reflect.TypeOf(obj))
	fmt.Printf("%+#v\n", obj)
}

func parseDir(path string, mode parser.Mode) (map[string]*PatchableFile, error) {
	m := make(map[string]*PatchableFile)
	lst, err := ioutil.ReadDir(path)
	if err != nil {
		return nil, err
	}
	fset := token.NewFileSet()
	for _, info := range lst {
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".go") {
			buf, err := ioutil.ReadFile(filepath.Join(path, info.Name()))
			if err != nil {
				return nil, err
			}
			file, err := parser.ParseFile(fset, info.Name(), buf, mode)
			if err != nil {
				return nil, err
			}
			m[info.Name()] = &PatchableFile{info.Name(), file, fset, string(buf)}
		}
	}
	return m, nil
}

type patchUnused struct {
	patches Patches
}

func (p *patchUnused) UnusedObj(obj *ast.Object) {
	if obj.Kind == ast.Fun {
		return
	}
	p.patches = append(p.patches, &Patch{obj.Decl.(ast.Node).End(), ";var _ = " + obj.Name})
}

func (p *patchUnused) UnusedImport(imp *ast.ImportSpec) {
	p.patches = append(p.patches, &Patch{imp.Pos(), "_ "})
}

func buildSloppy(path string) error {
	//files, err := parseDir(path, parser.ParseComments)
	lst, err := ioutil.ReadDir(path)
	if err != nil {
		return err
	}
	outdir := filepath.Join(path, "gosloppy")
	if err := os.MkdirAll(outdir, 0766); err != nil {
		return err
	}
	// defer os.RemoveAll(filepath.Join(path, "gosloppy"))
	for _, info := range lst {
		name := info.Name()
		if strings.HasSuffix(name, ".go") {
			patchable, err := ParsePatchable(filepath.Join(path, name))
			if err != nil {
				return err
			}
			patches := Patches{}
			UnusedInFile(patchable.File, &patchUnused{patches})
			file, err := os.Create(filepath.Join(outdir, name))
			if err != nil {
				return err
			}
			patchable.FprintPatched(file, patchable.File, patches)
			if err := file.Close(); err != nil {
				return err
			}
		}
	}
	return nil
}

func main() {
	fmt.Println("main")
	fset := token.NewFileSet()
	fset = fset
	var a = 1
	a = a
	if a == 0 {
	} else {
		a = 1
		a = 11
	}
	if err := buildSloppy(os.Args[1]); err != nil {
		log.Fatal(err)
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

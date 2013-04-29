package main

import (
	"flag"
	"fmt"
	"go/ast"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/elazarl/gosloppy/instrument"
	"github.com/elazarl/gosloppy/patch"
)

type patchUnused struct {
	patches patch.Patches
}

// TL;DR compareAssgn(x,y) should implement internalInterfacePointer(x) == internalInterfacePointer(y)
//
// Before jumping into the wrong conclusions, let me explain the purpose of the sad state of this function
// In Go a case clause can have a scope, for example:
//     select {
//     case scoper := <- c:
//     /* this is a []stmt which is in the scope of scoper */
//     }
// When we report that the assignment statement "scoper := <- c" is unused, we need to check whether or not
// this statement is in a CommClause or in the statement list afterwards.
// Since Go doesn't have (AFAIK) ability to compare pointers in two `interface{}`s, I implement it in an
// awkward way.
// Why don't we have this problem in, say for statment? This is because the for statement always have a
// block statement afterwards, therefor the "parent" object we report to UnusedObj is the block statment,
// thus, we can always be sure if we have an unused variable whose parent is a block statement - that this
// is the Init part of the for statement, ditto for if, etc.
func compareAssgn(lhs interface{}, rhs ast.Stmt) bool {
	switch lhs := lhs.(type) {
	case *ast.AssignStmt:
		rhs, ok := rhs.(*ast.AssignStmt)
		return ok && rhs == lhs
	case *ast.DeclStmt:
		rhs, ok := rhs.(*ast.DeclStmt)
		return ok && rhs == lhs
	}
	return false
}

func (p *patchUnused) UnusedObj(obj *ast.Object, parent ast.Node) {
	// if the unused variable is a function argument, or TLD - ignore
	switch obj.Decl.(type) {
	case *ast.Field, *ast.GenDecl, *ast.TypeSpec:
		return
	case *ast.ValueSpec:
		if _, ok := parent.(*ast.File); ok {
			return
		}
	}
	if obj.Kind == ast.Fun {
		return
	}
	exempter := "_ = " + obj.Name
	switch parent := parent.(type) {
	case *ast.ForStmt:
		p.patches = append(p.patches, patch.Insert(parent.Body.Lbrace+1, exempter+";"))
	case *ast.RangeStmt:
		p.patches = append(p.patches, patch.Insert(parent.Body.Lbrace+1, exempter+";"))
	case *ast.TypeSwitchStmt:
		if len(parent.Body.List) == 0 {
			p.patches = append(p.patches, patch.Insert(parent.Body.Lbrace+1, "default: "+exempter))
		} else {
			// if first statement is not case statement - it won't compile anyhow
			if stmt, ok := parent.Body.List[0].(*ast.CaseClause); ok {
				p.patches = append(p.patches, patch.Insert(stmt.Colon+1, exempter+";"))
			}
		}
	case *ast.SwitchStmt:
		if len(parent.Body.List) == 0 {
			p.patches = append(p.patches, patch.Insert(parent.Body.Lbrace+1, "default: "+exempter))
		} else {
			// if first statement is not case statement - it won't compile anyhow
			if stmt, ok := parent.Body.List[0].(*ast.CaseClause); ok {
				p.patches = append(p.patches, patch.Insert(stmt.Colon+1, exempter+";"))
			}
		}
	case *ast.CommClause:
		if compareAssgn(obj.Decl, parent.Comm) {
			p.patches = append(p.patches, patch.Insert(parent.Colon+1, exempter+";"))
		} else {
			p.patches = append(p.patches, patch.Insert(obj.Decl.(ast.Node).End(), ";"+exempter))
		}
	case *ast.IfStmt:
		p.patches = append(p.patches, patch.Insert(parent.Body.Lbrace+1, exempter+";"))
	default:
		p.patches = append(p.patches, patch.Insert(obj.Decl.(ast.Node).End(), ";"+exempter))
	}
}

func (p *patchUnused) UnusedImport(imp *ast.ImportSpec) {
	if imp.Name != nil {
		p.patches = append(p.patches, patch.Replace(imp.Name, "_"))
	} else {
		p.patches = append(p.patches, patch.Insert(imp.Pos(), "_ "))
	}
}

func usage() {
	fmt.Println(`Usage:
run tests:
gosloppy test <go test switches>
build a binary:
gosloppy build <go build switches>`)
}

type exitCode int

func die(err error) {
	if err != nil {
		os.Stderr.WriteString(err.Error() + "\n")
		panic(exitCode(-1))
	}
}

func mvToDir(srcdir, file, dstdir string) error {
	return os.Rename(filepath.Join(srcdir, file), filepath.Join(dstdir, file))
}

func main() {
	if len(os.Args) == 1 {
		usage()
		return
	}
	defer func() {
		if p := recover(); p != nil {
			if p, ok := p.(exitCode); ok {
				panic(p)
				os.Exit(int(p))
			} else {
				panic(p)
			}
		}
	}()
	f := flag.NewFlagSet("", flag.ContinueOnError)
	basedir := f.String("basedir", "", "instrument all packages decendant f basedir")
	gocmd, err := instrument.NewGoCmdWithFlags(f, ".", os.Args...)
	die(err)
	var pkg *instrument.Instrumentable
	if gocmd.Command == "run" {
		pkg = instrument.ImportFiles(*basedir, gocmd.Params...)
	} else if len(gocmd.Params) == 0 {
		wd, err := os.Getwd()
		if err != nil {
			panic(exitCode(2))
		}
		for _, path := range filepath.SplitList(os.Getenv("GOPATH")) {
			path = filepath.Join(path, "src")
			if strings.Contains(wd, path) {
				rel, err := filepath.Rel(path, wd)
				die(err)
				pkg, err = instrument.Import(*basedir, rel)
				break
			}
		}
		if pkg == nil {
			pkg, err = instrument.ImportDir(*basedir, ".")
		}
	} else {
		pkg, err = instrument.Import(*basedir, gocmd.Params[0])
	}
	die(err)
	shorterror := &ShortError{}
	outdir, err := pkg.Instrument(gocmd.Command == "test", func(p *patch.PatchableFile) patch.Patches {
		patches := &patchUnused{patch.Patches{}}
		shorterror.SetFile(p)
		autoimport := NewAutoImporter(p.File)
		WalkFile(NewMultiVisitor(NewUnusedVisitor(patches), autoimport, shorterror), p.File)
		return append(append(patches.patches, autoimport.Patches...), shorterror.Patches()...)
	})
	if gocmd.BuildFlags["work"] == "true" {
		log.Println("Instrumenting to", outdir)
	}
	defer func() {
		if gocmd.BuildFlags["work"] != "true" {
			if err := os.RemoveAll(outdir); err != nil {
				log.Println("Cannot remove temporary dir", outdir, err)
			}
		}
	}()
	die(err)
	newgocmd, err := gocmd.Retarget(outdir)
	die(err)
	newgocmd.Executable = "go"
	// TODO(elazarl): Support build gofile.go gofile2.go
	if newgocmd.Command != "run" {
		newgocmd.Params = nil
	}
	if f.Lookup("x").Value.String() == "true" {
		log.Println("In:", newgocmd.WorkDir)
		log.Println("Executing:", newgocmd)
	}
	// TODO(elazarl): hackish, find better way
	delete(newgocmd.BuildFlags, "basedir")
	minusC := newgocmd.BuildFlags["c"] != ""
	if newgocmd.Command == "test" {
		newgocmd.BuildFlags["c"] = "true"
	}
	die(newgocmd.Runnable().Run())
	if newgocmd.Command == "test" {
		_, _, err := newgocmd.OutputFileName()
		if err != nil {
			panic("Cannot find package name, not producing test executable")
		}
		oldname, _, err := gocmd.OutputFileName()
		if err != nil {
			panic("Cannot find package name, not producing test executable")
		}
		// output name is unclear: http://code.google.com/p/go/issues/detail?id=5230
		testoutput := filepath.Join(outdir, filepath.Base(outdir)+".test")
		if minusC {
			die(os.Rename(testoutput, oldname+".test"))
		} else {
			r := exec.Command(testoutput, newgocmd.ExtraFlags...)
			r.Dir = gocmd.WorkDir
			r.Stdin = os.Stdin
			r.Stdout = os.Stdout
			r.Stderr = os.Stderr
			err := r.Run()
			if err, ok := err.(*exec.ExitError); ok {
				_ = err
				os.Exit(-1) // TODO(elazar): pry from Go the actual exit code
			}
			die(err)
		}
	}
}

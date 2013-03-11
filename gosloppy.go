package main

import (
	"flag"
	"fmt"
	"go/ast"
	"log"
	"os"
	"path/filepath"

	"github.com/elazarl/gosloppy/instrument"
	"github.com/elazarl/gosloppy/patch"
)

type patchUnused struct {
	patches patch.Patches
}

func (p *patchUnused) UnusedObj(obj *ast.Object) {
	if _, ok := obj.Decl.(*ast.Field); ok {
		return // if the unused variable is a function argument - ignore
	}
	if obj.Kind == ast.Fun {
		return
	}
	p.patches = append(p.patches, patch.Insert(obj.Decl.(ast.Node).End(), ";var _ = "+obj.Name))
}

func (p *patchUnused) UnusedImport(imp *ast.ImportSpec) {
	p.patches = append(p.patches, patch.Insert(imp.Pos(), "_ "))
}

func usage() {
	fmt.Println(`Usage:
run tests:
gosloppy test <go test switches>
build a binary:
gosloppy build <go build switches>`)
}

func die(err error) {
	if err != nil {
		os.Stderr.WriteString(err.Error())
		os.Exit(-1)
	}
}

func mvToDir(srcdir, file, dstdir string) error {
	return os.Rename(filepath.Join(srcdir, file), filepath.Join(dstdir, file))
}

type AutoImporter []*ast.ImportSpec

func (v AutoImporter) VisitExpr(scope *ast.Scope, expr ast.Expr) ScopeVisitor {
	return v
}

func (v AutoImporter) VisitStmt(scope *ast.Scope, stmt ast.Stmt) ScopeVisitor {
	return v
}

func (v AutoImporter) ExitScope(scope *ast.Scope, node ast.Node, last bool) ScopeVisitor {
	return v
}

func main() {
	if len(os.Args) == 1 {
		usage()
		return
	}
	f := flag.NewFlagSet("", flag.ContinueOnError)
	basedir := f.String("basedir", "", "instrument all packages decendant f basedir")
	gocmd, err := instrument.NewGoCmdWithFlags(f, ".", os.Args...)
	die(err)
	var pkg *instrument.Instrumentable
	if len(gocmd.Packages) == 0 {
		pkg, err = instrument.ImportDir(*basedir, ".")
	} else {
		pkg, err = instrument.Import(*basedir, gocmd.Packages[0])
	}
	die(err)
	outdir, err := pkg.Instrument(func(p *patch.PatchableFile) patch.Patches {
		patches := &patchUnused{patch.Patches{}}
		WalkFile(MultiVisitor{NewUnusedVisitor(patches)}, p.File)
		return patches.patches
	})
	defer func() {
		if gocmd.BuildFlags["work"] != "true" {
			if err := os.RemoveAll(outdir); err != nil {
				log.Println("Cannot remove temporary dir", outdir, err)
			}
		}
	}()
	die(err)
	gocmd, err = gocmd.Retarget(outdir)
	die(err)
	if f.Lookup("x").Value.String() == "true" {
		log.Println("Executing:", gocmd)
	}
	die(gocmd.Runnable().Run())
	if gocmd.Command == "test" && gocmd.BuildFlags["c"] != "" {
		// TODO(elazar): caution, we assume outdir is immediately below us
		mvToDir(outdir, outdir+".test", ".")
	}
}

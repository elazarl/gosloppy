package main

import (
	"flag"
	"fmt"
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/elazarl/gosloppy/instrument"
	"github.com/elazarl/gosloppy/patch"
)

func parseDir(path string, mode parser.Mode) (map[string]*patch.PatchableFile, error) {
	m := make(map[string]*patch.PatchableFile)
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
			m[info.Name()] = &patch.PatchableFile{info.Name(), file, fset, string(buf)}
		}
	}
	return m, nil
}

type patchUnused struct {
	patches patch.Patches
}

func (p *patchUnused) UnusedObj(obj *ast.Object) {
	if obj.Kind == ast.Fun {
		return
	}
	p.patches = append(p.patches, patch.Insert(obj.Decl.(ast.Node).End(), ";var _ = "+obj.Name))
}

func (p *patchUnused) UnusedImport(imp *ast.ImportSpec) {
	p.patches = append(p.patches, patch.Insert(imp.Pos(), "_ "))
}

func normalize(imp string) string {
	if imp[0] == '"' {
		return imp[1 : len(imp)-1]
	}
	return imp
}

// given a package, it'll make all subpackages relative paths to a certain base
// ie, if one's base package path is "foo", and he compiles package
// $GOPATH/src/foo, which contains import to "foo/bar", it'll convert this import
// into "./bar", so that it'll compile.
func subpackageToRelative(basepkg, pkg string, file *ast.File) (patches patch.Patches) {
	for _, imp := range file.Imports {
		if filepath.HasPrefix(normalize(imp.Path.Value), basepkg) {
			rel, err := filepath.Rel(imp.Path.Value, pkg)
			if err != nil {
				log.Fatal("can't happen", err)
			}
			patches = append(patches, patch.Replace(imp.Path, rel))
		}
	}
	return
}

func buildSloppy(pkg *build.Package, srcdir, outdir string) error {
	//files, err := parseDir(srcDir, parser.ParseComments)
	lst, err := ioutil.ReadDir(srcdir)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(outdir, 0766); err != nil {
		return err
	}
	// defer os.RemoveAll(filepath.Join(srcDir, "gosloppy"))
	for _, info := range lst {
		name := info.Name()
		if strings.HasSuffix(name, ".go") {
			patchable, err := patch.ParsePatchable(filepath.Join(srcdir, name))
			if err != nil {
				return err
			}
			patches := &patchUnused{patch.Patches{}}
			UnusedInFile(patchable.File, patches)
			file, err := os.Create(filepath.Join(outdir, name))
			if err != nil {
				return err
			}
			patchable.FprintPatched(file, patchable.File, patches.patches)
			if err := file.Close(); err != nil {
				return err
			}
		}
	}
	return nil
}

func args() (args []string) {
	for _, arg := range os.Args[1:] {
		if !strings.HasPrefix(arg, "-") {
			args = append(args, arg)
		}
	}
	return
}

func sloppify(pkgname, outdir string) (sloppified string) {
	pkgdir := "."
	pkg, err := build.Import(pkgname, "", build.FindOnly)
	if err != nil {
		log.Fatal("Can't find package '", pkgname, "': ", err)
	}
	if pkgname != "" {
		pkgdir = pkg.Dir
	}
	if err := buildSloppy(pkg, pkgdir, outdir); err != nil {
		log.Fatal(err)
	}
	return filepath.Join(pkgdir, outdir)
}

func sloppifyHelper(args []string) (outdir string) {
	pkgname := ""
	if len(args) > 0 {
		pkgname = args[0]
	}
	return sloppify(pkgname, "__gosloppy")
}

func gocmd(keeptemp bool, dir string, args ...string) error {
	fmt.Println("EXEC:", args)
	cmd := exec.Command("go", args...)
	if dir != "" {
		cmd.Dir = dir
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		log.Fatal("Cannot execute go tool", err)
	}
	defer func() {
		if !keeptemp {
			os.RemoveAll(dir)
		}
	}()
	return cmd.Wait()
}

func gotest(args []string) error {
	keeptemp := flag.Bool("keeptemp", false, "keep temporary .go files after fixed by goproxy")
	flag.Parse()
	return gocmd(*keeptemp, sloppifyHelper(args), os.Args[1:]...)
}

func gobuild(args []string) error {
	keeptemp := flag.Bool("keeptemp", false, "keep temporary .go files after fixed by goproxy")
	output := flag.String("o", "", "output directory")
	flag.Parse()
	if *output == "" {
		// TODO(elazar): make sure nothing funny happens when having symlinks in path
		// maybe prefer paths withing $GOPATH?
		wd, err := os.Getwd()
		if err != nil {
			log.Fatalln("Can't get CWD:", err)
		}
		name := filepath.Base(wd)
		output = &name
	}
	goargs := append(flag.Args(), "-o", filepath.Join("..", *output))
	return gocmd(*keeptemp, sloppifyHelper(goargs), goargs...)
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

func main() {
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
	outdir, err := pkg.Instrument("__goproxy", func(p *patch.PatchableFile) patch.Patches {
		patches := &patchUnused{patch.Patches{}}
		UnusedInFile(p.File, patches)
		return patches.patches
	})
	die(err)
	gocmd, err = gocmd.Retarget(outdir)
	die(err)
	if f.Lookup("x").Value.String() == "true" {
		log.Println("Executing:", gocmd)
	}
	die(gocmd.Runnable().Run())
}

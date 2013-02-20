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
	"reflect"
	"strings"
)

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

func buildSloppy(srcdir, outdir string) error {
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
			patchable, err := ParsePatchable(filepath.Join(srcdir, name))
			if err != nil {
				return err
			}
			patches := &patchUnused{Patches{}}
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

func sloppify(pkgname string) (sloppified string) {
	pkgdir := "."
	outdir := "__gosloppy"
	if pkgname != "" {
		p, err := build.Import(pkgname, "", build.FindOnly)
		if err != nil {
			log.Fatal("Can't find package '", pkgname, "': ", err)
		}
		pkgdir = p.Dir
	}
	if err := buildSloppy(pkgdir, outdir); err != nil {
		log.Fatal(err)
	}
	return filepath.Join(pkgdir, outdir)
}

func sloppifyHelper(args []string) (outdir string) {
	pkgname := ""
	if len(args) > 0 {
		pkgname = args[0]
	}
	return sloppify(pkgname)
}

func gocmd(keeptemp bool, dir string, args ...string) error {
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
	return gocmd(*keeptemp, sloppifyHelper(args), goargs...)
}

func usage() {
	fmt.Println(`Usage:
run tests:
gosloppy test <go test switches>
build a binary:
gosloppy build <go build switches>`)
}

func main() {
	args := args()
	jumptable := map[string]func([]string) error{
		"test":  gotest,
		"build": gobuild,
	}
	if len(args) == 0 {
		usage()
		return
	}
	if f, ok := jumptable[args[0]]; !ok {
		usage()
		log.Fatal("can't find action", args[0], jumptable[args[0]])
	} else {
		f(args[1:])
	}
}

package instrument

import (
	"go/build"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/elazarl/gosloppy/patch"
)

// Instrumentable is a go package, given either by a GOPATH package or
// by a specific dir
type Instrumentable struct {
	pkg     *build.Package
	basepkg string
}

// Files will give all .go files of a go pacakge
func (i *Instrumentable) Files() (files []string) {
	for _, gofiles := range [][]string{i.pkg.GoFiles, i.pkg.CgoFiles, i.pkg.TestGoFiles} {
		for _, file := range gofiles {
			files = append(files, filepath.Join(i.pkg.Dir, file))
		}
	}
	return
}

func guessBasepkg(importpath string) string {
	path, err := repoRootForImportPathStatic(importpath)
	if err != nil {
		p := filepath.Dir(importpath)
		for strings.Contains(p, "/") {
			parent := filepath.Dir(p)
			if _, err := build.Import(parent, "", 0); err != nil {
				return p
			}
			p = parent
		}
		return p
	}
	return path.root
}

// Import gives an Instrumentable for a given package name, it will instrument pkgname
// and all subpacakges of basepkg that pkgname imports.
// Leave basepkg empty to have Import guess it for you.
// The conservative default for basepkg is basepkg==pkgname.
// For example, if we have packages a/x a/b and a/b/c in GOPATH
//     gopath/src
//         a/
//           x/
//           b/
//             c/
// and package c imports packages a/x and a/b, calling Import("a", "a/b/c") will instrument
// packages a/b/c, a/b and a/x. Calling Import("a/b", "a/b/c") will instrument
// pacakges a/b and a/b/c. Calling Import("a/b/c", "a/b/c") will instrument package "a/b/c"
// alone.
func Import(basepkg, pkgname string) (*Instrumentable, error) {
	if pkg, err := build.Import(pkgname, "", 0); err != nil {
		return nil, err
	} else {
		if basepkg == "" {
			basepkg = guessBasepkg(pkg.ImportPath)
		}
		return &Instrumentable{pkg, basepkg}, nil
	}
	panic("unreachable")
}

// ImportDir gives a single instrumentable golang package. See Import.
func ImportDir(basepkg, pkgname string) (*Instrumentable, error) {
	if pkg, err := build.ImportDir(pkgname, 0); err != nil {
		return nil, err
	} else {
		return &Instrumentable{pkg, basepkg}, nil
	}
	panic("unreachable")
}

// IsInGopath returns whether the Instrumentable is a package in a standalone directory or in GOPATH
func (i *Instrumentable) IsInGopath() bool {
	return i.pkg.ImportPath != "."
}

func (i *Instrumentable) relevantImport(imp string) bool {
	if !i.IsInGopath() && build.IsLocalImport(imp) {
		imp = filepath.Clean(filepath.Join(i.pkg.Dir, imp))
	}
	return filepath.HasPrefix(imp, i.basepkg)
}

func (i *Instrumentable) doimport(pkg string) (*Instrumentable, error) {
	if build.IsLocalImport(pkg) {
		return ImportDir(i.basepkg, filepath.Join(i.pkg.Dir, pkg))
	}
	return Import(i.basepkg, pkg)
}

var tempStem = "__instrument.go"

func (i *Instrumentable) Instrument(f func(file *patch.PatchableFile) patch.Patches) (pkgdir string, err error) {
	d, err := ioutil.TempDir(".", tempStem)
	if err != nil {
		return "", err
	}
	return i.InstrumentTo(d, f)
}

func (i *Instrumentable) InstrumentTo(outdir string, f func(file *patch.PatchableFile) patch.Patches) (pkgdir string, err error) {
	for _, imps := range [][]string{i.pkg.Imports, i.pkg.TestImports} {
		for _, imp := range imps {
			if i.relevantImport(imp) {
				if imp, err := i.doimport(imp); err != nil {
					return "", err
				} else {
					imp.InstrumentTo(outdir, f)
				}
			}
		}
	}
	rel, err := filepath.Rel(i.basepkg, i.pkg.ImportPath)
	// if we used build.ImportDir, we'll get a package with ImportPath "." and Dir as the package's source dir
	if build.IsLocalImport(i.pkg.ImportPath) {
		rel, err = filepath.Rel(i.basepkg, i.pkg.Dir)
	}
	if err != nil {
		return "", err
	}
	finaldir := filepath.Join(outdir, rel)
	if err := os.MkdirAll(finaldir, 0755); err != nil {
		return "", err
	}
	InstrumentTo(finaldir, i.Files(), f)
	return finaldir, nil
}

// Instrument parses all gofiles, invoke f on any file, and then writes it with the same name to outdir
func Instrument(gofiles []string, f func(file *patch.PatchableFile) patch.Patches) error {
	d, err := ioutil.TempDir(".", tempStem)
	if err != nil {
		return err
	}
	return InstrumentTo(d, gofiles, f)
}

// Instrument parses all gofiles, invoke f on any file, and then writes it with the same name to outdir
func InstrumentTo(outdir string, gofiles []string, f func(file *patch.PatchableFile) patch.Patches) error {
	if err := os.MkdirAll(outdir, 0755); err != nil {
		return err
	}
	pkg := patch.NewPatchablePkg()
	if err := pkg.ParseFiles(gofiles...); err != nil {
		return err
	}
	for filename, file := range pkg.Files {
		if outfile, err := os.Create(filepath.Join(outdir, filepath.Base(filename))); err != nil {
			return err
		} else {
			patches := f(file)
			file.FprintPatched(outfile, file.File, patches)
			outfile.Close()
		}
	}
	return nil
}

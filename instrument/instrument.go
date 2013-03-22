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
	// TODO(elazar): do not instrument tests unless called with `gosloppy test`
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
// If our package is not in $GOPATH, (typically built with `cd pkg;go build -o a.out`), the
// default empty basepkg will always import all relative paths.
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

// relevantImport will determine whether this import should be instrumented as well
func (i *Instrumentable) relevantImport(imp string) bool {
	if i.basepkg == "*" {
		return true
	} else if i.IsInGopath() || i.basepkg != "" {
		return filepath.HasPrefix(imp, i.basepkg) || filepath.HasPrefix(i.basepkg, imp)
	} else {
		return build.IsLocalImport(imp)
	}
	panic("unreachable")
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

func localize(pkg string) string {
	if build.IsLocalImport(pkg) {
		// TODO(elazar): check if `import "./a/../a"` is equivalent to "./a"
		pkg := filepath.Clean(pkg)
		return filepath.Join(".", "locals", strings.Replace(pkg, ".", "_", -1))
	}
	return filepath.Join("gopath", pkg)
}

// InstrumentTo will instrument all files in Instrumentable into outdir. It will instrument all subpackages
// as described in Import.
func (i *Instrumentable) InstrumentTo(outdir string, f func(file *patch.PatchableFile) patch.Patches) (pkgdir string, err error) {
	return i.instrumentTo(outdir, "", f)
}

func (i *Instrumentable) instrumentTo(outdir, mypath string, f func(file *patch.PatchableFile) patch.Patches) (pkgdir string, err error) {
	for _, imps := range [][]string{i.pkg.Imports, i.pkg.TestImports} {
		for _, imp := range imps {
			if i.relevantImport(imp) {
				if pkg, err := i.doimport(imp); err != nil {
					return "", err
				} else {
					if _, err := pkg.instrumentTo(filepath.Join(outdir, localize(imp)),
						filepath.Join(mypath, imp), f); err != nil {
						return "", err
					}
				}
			}
		}
	}
	if err := os.MkdirAll(outdir, 0755); err != nil {
		return "", err
	}
	pkg := patch.NewPatchablePkg()
	if err := pkg.ParseFiles(i.Files()...); err != nil {
		return "", err
	}
	if err := i.instrumentPatchable(outdir, mypath, pkg, f); err != nil {
		return "", err
	}
	return outdir, nil
}

func (i *Instrumentable) instrumentPatchable(outdir, mypath string, pkg *patch.PatchablePkg, f func(file *patch.PatchableFile) patch.Patches) error {
	for filename, file := range pkg.Files {
		if outfile, err := os.Create(filepath.Join(outdir, filepath.Base(filename))); err != nil {
			return err
		} else {
			patches := f(file)
			for _, imp := range file.File.Imports {
				v := imp.Path.Value[1 : len(imp.Path.Value)-1]
				if !i.relevantImport(v) {
					continue
				}
				if build.IsLocalImport(v) {
					v = filepath.Clean(filepath.Join(mypath, v))
					patches = appendNoContradict(patches, patch.Replace(imp.Path, `"./locals/`+v+`"`))
				} else {
					patches = appendNoContradict(patches, patch.Replace(imp.Path, `"./gopath/`+v+`"`))
				}
			}
			file.FprintPatched(outfile, file.File, patches)
			if err := outfile.Close(); err != nil {
				return err
			}
		}
	}
	return nil
}

func appendNoContradict(patches patch.Patches, toadd *patch.Patch) patch.Patches {
	for _, p := range patches {
		if toadd.End <= p.End && toadd.End >= p.Start ||
			toadd.Start <= p.End && toadd.Start >= p.Start {
			return patches
		}
	}
	return append(patches, toadd)
}

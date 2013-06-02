package instrument

import (
	"go/build"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/elazarl/gosloppy/patch"
)

// Instrumentable is a go package, given either by a GOPATH package or
// by a specific dir
type Instrumentable struct {
	pkg              *build.Package
	basepkg          string
	name             string
	InstrumentGoroot bool
	gorootPkgs       map[string]bool
}

// Files will give all .go files of a go pacakge
func (i *Instrumentable) Files() (files []string) {
	// TODO(elazar): do not instrument tests unless called with `gosloppy test`
	for _, gofiles := range [][]string{i.pkg.GoFiles, i.pkg.CgoFiles} {
		for _, file := range gofiles {
			files = append(files, filepath.Join(i.pkg.Dir, file))
		}
	}
	return
}

// TestFiles will give all .go files of the _test.go files using the same package
func (i *Instrumentable) TestFiles() (files []string) {
	// TODO(elazar): do not instrument tests unless called with `gosloppy test`
	for _, gofiles := range [][]string{i.pkg.GoFiles, i.pkg.CgoFiles, i.pkg.TestGoFiles} {
		for _, file := range gofiles {
			files = append(files, filepath.Join(i.pkg.Dir, file))
		}
	}
	return
}

// XTestFiles returns paths all files in external test package
func (i *Instrumentable) XTestFiles() (files []string) {
	// TODO(elazar): do not instrument tests unless called with `gosloppy test`
	for _, gofiles := range [][]string{i.pkg.XTestGoFiles} {
		for _, file := range gofiles {
			files = append(files, filepath.Join(i.pkg.Dir, file))
		}
	}
	return
}

func guessBasepkg(importpath string) string {
	path, err := repoRootForImportPathStatic(importpath)
	if err != nil {
		p := importpath
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
	pkg, err := build.Import(pkgname, "", 0)
	if err != nil {
		return nil, err
	}
	if basepkg == "" {
		basepkg = guessBasepkg(pkg.ImportPath)
	}
	return &Instrumentable{pkg, basepkg, pkgname, false, make(map[string]bool)}, nil
}

func ImportFiles(basepkg string, files ...string) *Instrumentable {
	return &Instrumentable{&build.Package{GoFiles: files}, basepkg, "", false, make(map[string]bool)}
}

// ImportDir gives a single instrumentable golang package. See Import.
func ImportDir(basepkg, pkgname string) (*Instrumentable, error) {
	pkg, err := build.ImportDir(pkgname, 0)
	if err != nil {
		return nil, err
	}
	return &Instrumentable{pkg, basepkg, pkgname, false, make(map[string]bool)}, nil
}

// IsInGopath returns whether the Instrumentable is a package in a standalone directory or in GOPATH
func (i *Instrumentable) IsInGopath() bool {
	return i.pkg.ImportPath != "." && !i.pkg.Goroot
}

// relevantImport will determine whether this import should be instrumented as well
func (i *Instrumentable) relevantImport(imp string) bool {
	switch {
	case imp == "C":
		return false
	case i.gorootPkgs[imp] && !i.InstrumentGoroot:
		return false
	case i.basepkg == "*" || build.IsLocalImport(imp):
		return true
	case i.IsInGopath() || i.basepkg != "":
		return filepath.HasPrefix(imp, i.basepkg) || filepath.HasPrefix(i.basepkg, imp)
	}
	return false
}

func (i *Instrumentable) doimport(pkg string) (*Instrumentable, error) {
	if build.IsLocalImport(pkg) {
		return ImportDir(i.basepkg, filepath.Join(i.pkg.Dir, pkg))
	}
	// TODO: A bit hackish
	r, err := Import(i.basepkg, pkg)
	if err != nil {
		return r, err
	}
	r.name = i.name
	r.gorootPkgs = i.gorootPkgs
	r.InstrumentGoroot = i.InstrumentGoroot
	return r, nil
}

var tempStem = "__instrument.go"

func (i *Instrumentable) Instrument(withtests bool, f func(file *patch.PatchableFile) patch.Patches) (pkgdir string, hasGoroot bool, err error) {
	d, err := ioutil.TempDir(os.TempDir(), tempStem)
	if err != nil {
		return "", false, err
	}
	hasGoroot, err = i.InstrumentTo(withtests, d, f)
	return d, hasGoroot, err
}

func (i *Instrumentable) InstrumentInline(f func(file *patch.PatchableFile) patch.Patches) error {
	return i.instrumentInline(make(map[string]bool), f)
}

// returns a string that identifies the package
func (i *Instrumentable) id() string {
	if i.pkg.ImportPath == "" {
		if i.pkg.Dir == "" {
			// It's just a bunch of files
			return strings.Join(i.pkg.GoFiles, ",")
		}
		// A non-gopath package
		return i.pkg.Dir
	}
	return i.pkg.ImportPath
}

func (i *Instrumentable) instrumentInline(processed map[string]bool, f func(file *patch.PatchableFile) patch.Patches) error {
	if processed[i.id()] {
		return nil
	}
	processed[i.id()] = true
	imps := i.pkg.Imports
	if true {
		imps = append(imps, i.pkg.TestImports...)
		imps = append(imps, i.pkg.XTestImports...)
	}
	for _, imp := range imps {
		if i.relevantImport(imp) {
			pkg, err := i.doimport(imp)
			if err != nil {
				return err
			}
			if err := pkg.instrumentInline(processed, f); err != nil {
				return err
			}
		}
	}
	pkg := patch.NewPatchablePkg()
	if err := pkg.ParseFiles(i.Files()...); err != nil {
		return err
	}
	for path, file := range pkg.Files {
		patches := f(file)
		outfile, err := os.OpenFile(path, os.O_WRONLY|os.O_TRUNC, 0644)
		if err != nil {
			return err
		}
		if _, err := file.FprintPatched(outfile, file.All(), patches); err != nil {
			return err
		}
		if err := outfile.Close(); err != nil {
			return err
		}
	}
	return nil
}

// InstrumentTo will instrument all files in Instrumentable into outdir. It will instrument all subpackages
// as described in Import.
func (i *Instrumentable) InstrumentTo(withtests bool, outdir string,
	f func(file *patch.PatchableFile) patch.Patches) (hasGoroot bool, err error) {
	if err := i.instrumentTo(map[string]bool{}, withtests, outdir, "", f); err != nil {
		return false, err
	}
	hasGoroot = len(i.gorootPkgs) > 0 && i.InstrumentGoroot
	if hasGoroot {
		if err := symlinkGoroot(filepath.Join(outdir, "goroot")); err != nil {
			return false, err
		}
	}
	return hasGoroot, nil
}

func (i *Instrumentable) instrumentTo(processed map[string]bool, istest bool, outdir, relpath string, f func(file *patch.PatchableFile) patch.Patches) error {
	if processed[relpath] {
		return nil
	}
	processed[i.pkg.ImportPath] = true
	imps := i.pkg.Imports
	if istest {
		imps = append(imps, i.pkg.TestImports...)
		imps = append(imps, i.pkg.XTestImports...)
	}
	for _, imp := range imps {
		if i.relevantImport(imp) {
			pkg, err := i.doimport(imp)
			if err != nil {
				return err
			}
			if build.IsLocalImport(imp) {
				imp = "./" + filepath.Join(relpath, imp)
			}
			if err := pkg.instrumentTo(processed, false, outdir, imp, f); err != nil {
				return err
			}
		}
	}
	if !istest {
		pkg := patch.NewPatchablePkg()
		if err := pkg.ParseFiles(i.Files()...); err != nil {
			return err
		}
		if err := i.instrumentPatchable(outdir, relpath, pkg, f); err != nil {
			return err
		}
	} else {
		pkg := patch.NewPatchablePkg()
		if err := pkg.ParseFiles(i.TestFiles()...); err != nil {
			return err
		}
		if err := i.instrumentPatchable(outdir, relpath, pkg, f); err != nil {
			return err
		}
		pkg = patch.NewPatchablePkg()
		if err := pkg.ParseFiles(i.XTestFiles()...); err != nil {
			return err
		}
		if err := i.instrumentPatchable(outdir, relpath, pkg, f); err != nil {
			return err
		}
	}
	return nil
}

func cp(dst, src string) error {
	s, err := os.Open(src)
	if err != nil {
		return err
	}
	defer s.Close()
	d, err := os.Create(dst)
	if err != nil {
		return err
	}
	if _, err := io.Copy(d, s); err != nil {
		d.Close()
		return err
	}
	return d.Close()
}

func (i *Instrumentable) instrumentPatchable(outdir, relpath string, pkg *patch.PatchablePkg, f func(file *patch.PatchableFile) patch.Patches) error {
	path := ""
	if build.IsLocalImport(relpath) {
		path = strings.Replace(relpath, "..", "__", -1)
		path = filepath.Join("locals", path)
	} else if i.pkg.Goroot {
		path = filepath.Join("goroot", "src", "pkg", i.pkg.ImportPath)
		i.gorootPkgs[i.pkg.ImportPath] = true
	} else if relpath != "" {
		path = filepath.Join("gopath", i.pkg.ImportPath)
	}
	if err := os.MkdirAll(filepath.Join(outdir, path), 0755); err != nil {
		return err
	}
	// copy all none-go files (TODO: symlink? OTOH you wouldn't have standalone package)
	for _, files := range [][]string{i.pkg.CFiles, i.pkg.HFiles, i.pkg.SFiles, i.pkg.SysoFiles} {
		for _, file := range files {
			if err := cp(filepath.Join(outdir, path, file), filepath.Join(i.pkg.Dir, file)); err != nil {
				return err
			}
		}
	}
	for filename, file := range pkg.Files {
		if outfile, err := os.Create(filepath.Join(outdir, path, filepath.Base(filename))); err != nil {
			return err
		} else {
			patches := f(file)
			// TODO(elazar): check the relative path from current location (aka relpath, path), to the import path
			// (aka v)
			for _, imp := range file.File.Imports {
				switch v := imp.Path.Value[1 : len(imp.Path.Value)-1]; {
				case v == i.pkg.ImportPath:
					patches = appendNoContradict(patches, patch.Replace(imp.Path, `"."`))
				case !i.relevantImport(v) || i.gorootPkgs[v]:
					continue
				case build.IsLocalImport(v):
					rel, err := filepath.Rel(path, filepath.Join("locals", v))
					if err != nil {
						return err
					}
					patches = appendNoContradict(patches, patch.Replace(imp.Path, `"./`+rel+`"`))
				default:
					if v == i.name {
						v = ""
					} else {
						v = filepath.Join("gopath", v)
					}
					rel, err := filepath.Rel(path, v)
					if err != nil {
						return err
					}
					patches = appendNoContradict(patches, patch.Replace(imp.Path, `"./`+rel+`"`))
				}
			}
			if _, err := file.FprintPatched(outfile, file.All(), patches); err != nil {
				return err
			}
			if err := outfile.Close(); err != nil {
				return err
			}
		}
	}
	return nil
}

func appendNoContradict(patches patch.Patches, toadd patch.Patch) patch.Patches {
	for _, p := range patches {
		if toadd.EndPos() <= p.EndPos() && toadd.EndPos() >= p.StartPos() ||
			toadd.StartPos() <= p.EndPos() && toadd.StartPos() >= p.StartPos() {
			return patches
		}
	}
	return append(patches, toadd)
}

package instrument

import (
	"go/build"
	"os"
	"path/filepath"

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

func Import(basepkg, pkgname string) (*Instrumentable, error) {
	if pkg, err := build.Import(pkgname, "", 0); err != nil {
		return nil, err
	} else {
		return &Instrumentable{pkg, basepkg}, nil
	}
	panic("unreachable")
}

func ImportDir(basepkg, pkgname string) (*Instrumentable, error) {
	if pkg, err := build.ImportDir(pkgname, 0); err != nil {
		return nil, err
	} else {
		return &Instrumentable{pkg, basepkg}, nil
	}
	panic("unreachable")
}

func (i *Instrumentable) Instrument(outdir string, f func(file *patch.PatchableFile) patch.Patches) error {
	for _, imp := range i.pkg.Imports {
		if filepath.HasPrefix(imp, i.basepkg) {
			if imp, err := Import(i.basepkg, imp); err != nil {
				return err
			} else {
				imp.Instrument(outdir, f)
			}
		}
	}
	rel, err := filepath.Rel(i.basepkg, i.pkg.ImportPath)
	// if we used build.ImportDir, we'll get a package with ImportPath "." and Dir as the package's source dir
	if build.IsLocalImport(i.pkg.ImportPath) {
		rel, err = filepath.Rel(i.basepkg, i.pkg.Dir)
	}
	if err != nil {
		return err
	}
	finaldir := filepath.Join(outdir, rel)
	if err := os.MkdirAll(finaldir, 0755); err != nil {
		return err
	}
	Instrument(finaldir, i.Files(), f)
	return nil
}

// Instrument parses all gofiles, invoke f on any file, and then writes it with the same name to outdir
func Instrument(outdir string, gofiles []string, f func(file *patch.PatchableFile) patch.Patches) error {
	if err := os.MkdirAll(outdir, 0755); err != nil {
		return err
	}
	for _, filename := range gofiles {
		file, err := patch.ParsePatchable(filename)
		if err != nil {
			return err
		}
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

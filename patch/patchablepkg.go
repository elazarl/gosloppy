package patch

import (
	"go/ast"
)

type PatchablePkg struct {
	Name  string
	Scope *ast.Scope
	Files map[string]*PatchableFile
	// Imports not used, since I don't want to parse all imports
	// Imports map[string]PatchablePkg
}

func NewPatchablePkg() *PatchablePkg {
	return &PatchablePkg{
		Scope: ast.NewScope(nil),
		Files: make(map[string]*PatchableFile),
	}
}

func (pkg *PatchablePkg) ParseFiles(files ...string) error {
	for _, file := range files {
		if err := pkg.ParseFile(file); err != nil {
			return err
		}
	}
	return nil
}

func (pkg *PatchablePkg) ParseFile(file string) error {
	patchable, err := ParsePatchable(file)
	if err != nil {
		return err
	}
	if pkg.Name != "" && pkg.Name != patchable.PkgName {
		panic("ParsePkg called with files in two different packages. Had " +
			pkg.Name + " got " + patchable.PkgName + " from " + file)
	}
	pkg.Name = patchable.File.Name.String()
	if _, ok := pkg.Files[file]; ok {
		panic("File " + file + "parsed twice")
	}
	pkg.Files[file] = patchable
	for _, obj := range patchable.File.Scope.Objects {
		pkg.Scope.Insert(obj)
	}
	return nil
}

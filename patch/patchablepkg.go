package patch

import (
	"go/ast"
	"go/build"
)

type PatchablePkg struct {
	Name  string
	Scope *ast.Scope
	Files map[string]*PatchableFile
	// Imports not used, since I don't want to parse all imports
	// Imports map[string]PatchablePkg
}

// TODO(elazar): this is very basic, and requires more work for edge cases and builds with C
// Warning, file ASTs are SHARED between test and pkg. This might be desired, need to think over.
func ParsePackage(buildpkg *build.Package) (pkg *PatchablePkg, testpkg *PatchablePkg, err error) {
	pkg = NewPatchablePkg()
	testpkg = NewPatchablePkg()
	if err := pkg.ParseFiles(buildpkg.GoFiles...); err != nil {
		return nil, nil, err
	}
	if err := pkg.ParseFiles(buildpkg.CgoFiles...); err != nil {
		return nil, nil, err
	}
	for k, v := range pkg.Files {
		testpkg.Files[k] = v
	}
	for _, obj := range pkg.Scope.Objects {
		testpkg.Scope.Insert(obj)
	}
	if err := testpkg.ParseFiles(buildpkg.TestGoFiles...); err != nil {
		return nil, nil, err
	}
	return pkg, testpkg, nil
}

func ParseFiles(files ...string) (*PatchablePkg, error) {
	pkg := NewPatchablePkg()
	if err := pkg.ParseFiles(files...); err != nil {
		return nil, err
	}
	return pkg, nil
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
	patchable.File.Scope.Outer = pkg.Scope
	return nil
}

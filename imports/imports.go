package imports

import (
	"go/ast"
	"go/build"
	"log"
	"strings"
)

type ImportCache map[string]string

var DefaultImportCache = ImportCache{}

// will get the package name, or guess it if absent
func (cache ImportCache) GetNameOrGuess(imp *ast.ImportSpec) string {
	if rv, ok := cache[imp.Path.Value]; ok {
		return rv
	}
	rv := getNameOrGuess(imp)
	cache[imp.Path.Value] = rv
	return rv
}

func GetNameOrGuess(imp *ast.ImportSpec) string {
	return DefaultImportCache.GetNameOrGuess(imp)
}

func getNameOrGuess(imp *ast.ImportSpec) string {
	// remove quotes
	path := imp.Path.Value[1 : len(imp.Path.Value)-1]
	pkg, err := build.Import(path, ".", build.AllowBinary)
	if err != nil {
		parts := strings.Split(path, "/")
		rv := parts[len(parts)-1]
		// I don't want to fail if I can't find the package
		// maybe the user is smarter than me, so I guess it's name
		log.Println("Cannot find package", path, "guessing it's name is", rv)
		return rv
	}
	return pkg.Name
}

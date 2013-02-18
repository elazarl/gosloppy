package imports

import (
	"go/ast"
	"go/build"
	"log"
	"strings"
)

type ImportCache struct{}

var DefaultImportCache = ImportCache{}

// will get the package name, or guess it if absent
func GetNameOrGuess(imp *ast.ImportSpec) string {
	// remove quotes
	path := imp.Path.Value[1 : len(imp.Path.Value)-1]
	pkg, err := build.Import(path, ".", build.AllowBinary)
	if err != nil {
		log.Println("Cannot find package", path)
		parts := strings.Split(path, "/")
		return parts[len(parts)-1]
	}
	return pkg.Name
}

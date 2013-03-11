package imports

import (
	"go/ast"
	"testing"
)

// TODO(elazar): test subpackages fetching and cache
func TestGetPackageName(t *testing.T) {
	for pkg, name := range DefaultImportCache {
		actual := getNameOrGuess(&ast.ImportSpec{Path: &ast.BasicLit{Value: pkg}})
		if actual != name {
			t.Fatalf("standard package %s name evaluated %s != %s", pkg, actual, name)
		}
	}
}

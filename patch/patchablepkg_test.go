package patch

import (
	"fmt"
	"go/ast"
	"io/ioutil"
	"os"
	"sort"
	"testing"
)

func TestScope(t *testing.T) {
	defer cleanUp()
	pkg := NewPatchablePkg()
	pkg.ParseFile(file(`package main;func f()`))
	ensureScope(t, pkg.Scope, "f")
	pkg.ParseFile(file(`package main;var v = 1`))
	ensureScope(t, pkg.Scope, "f", "v")
	pkg.ParseFile(file(`package main;var _ = 1;func init();func foo(a,b  int, v string) { v = a+b }`))
	ensureScope(t, pkg.Scope, "f", "foo", "v")
	pkg.ParseFile(file(`package main;import "fmt";func p() {fmt.Println("foo")}`))
	ensureScope(t, pkg.Scope, "f", "foo", "p", "v")
}

var tempFiles []string

func file(content string) (filename string) {
	f, err := ioutil.TempFile(os.TempDir(), "gosloppy.patch.test")
	if err != nil {
		panic(err)
	}
	f.WriteString(content)
	f.Close()
	tempFiles = append(tempFiles, f.Name())
	return f.Name()
}

func cleanUp() {
	for _, file := range tempFiles {
		if err := os.Remove(file); err != nil {
			fmt.Println("Warning: Cannot remove", file)
		}
	}
	tempFiles = nil
}

func ensureScope(t *testing.T, scope *ast.Scope, expected ...string) []string {
	names := []string{}
	for name, _ := range scope.Objects {
		names = append(names, name)
	}
	sort.Strings(names)
	if fmt.Sprint(names) != fmt.Sprint(expected) {
		t.Errorf("Expected %v got %v", expected, names)
	}
	return names
}

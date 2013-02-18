package imports

import (
	"go/build"
	"testing"
)

// TODO(elazar): write a real cache, then write a real test...
func TestGetPackageName(t *testing.T) {
	pkg, err := build.Import("fmt", ".", build.AllowBinary)
	if err != nil {
		t.Fatal(err)
	}
	if pkg.Name != "fmt" {
		t.Error("Expected fmt got", pkg.Name)
	}
}

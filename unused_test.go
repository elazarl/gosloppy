package main

import (
	"fmt"
	"go/ast"
	"testing"
)

func equal(a, b []string) {
}

type unusedNames func(string)

func (f unusedNames) UnusedObj(obj *ast.Object) {
	f(obj.Name)
}

func (f unusedNames) UnusedImport(imp *ast.ImportSpec) {
	f(imp.Path.Value)
}

func TestSimpleUnused(t *testing.T) {
	for i, c := range UnusedSimple {
		file, _ := parse(c.body, t)
		unused := []string{}
		WalkFile(NewUnusedVisitor(unusedNames(func(name string) {
			unused = append(unused, name)
		})), file)
		if fmt.Sprint(unused) != fmt.Sprint(c.expUnused) {
			t.Errorf("Case #%d:\n%s\n Expected unused %v got %v", i, c.body, c.expUnused, unused)
		}
	}
}

var UnusedSimple = []struct {
	body      string
	expUnused []string
}{
	{
		`package main
		func f(a int) {
		}
		`,
		[]string{"a", "f"},
	},
	{
		`package main
		func f(a int) {
			a = 1
		}
		`,
		[]string{"f"},
	},
	{
		`package main
		func f(a int) {
			if true {
				a = 1
			}
		}
		`,
		[]string{"f"},
	},
	{
		`package main
		func f(a int) {
			var _ = func () {
				b := a
			}
		}
		`,
		[]string{"b", "f"},
	},
	{
		`package main
		func f(a int) {
			var _ = func () {
				b := a
				b = 1
			}
		}
		`,
		[]string{"f"},
	},
	{
		`package main
		import "fmt"
		`,
		[]string{`"fmt"`},
	},
	{
		`package main
		import "fmt"
		var i = fmt.Println
		`,
		[]string{"i"},
	},
	{
		`package main
		import "fmt"
		func f(_ fmt.Stringer)
		`,
		[]string{"f"},
	},
	{
		`package main
		import "io/ioutil"
		var _ = ioutil.Discard
		`,
		[]string{},
	},
}

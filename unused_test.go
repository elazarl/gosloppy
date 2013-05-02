package main

import (
	"flag"
	"fmt"
	"go/ast"
	"testing"
)

func equal(a, b []string) {
}

type unusedNames func(string)

func (f unusedNames) UnusedObj(obj *ast.Object, parent ast.Node) {
	f(obj.Name)
}

func (f unusedNames) UnusedImport(imp *ast.ImportSpec) {
	f(imp.Path.Value)
}

var ncase = flag.Int("case", -1, "run specific case only")

func init() {
	flag.Parse()
}

// TODO(elazar): more complex tests:
//   1. What should happen when I `import . "foo"`, and use `var foo` from other package?
func TestSimpleUnused(t *testing.T) {
	for i, c := range UnusedSimple {
		if *ncase != i && *ncase > 0 {
			continue
		}
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
		func init() {
			a := 1
			if true {
				a := 2
				_ = a
			}
		}
		`,
		[]string{"a"},
	},
	{
		`package main
		func init() {
			for i := range []int{} {
				println(i)
			}
		}
		`,
		[]string{},
	},
	{"package main;import \"strings\";type T struct {A int};func init() *A { return T{A: strings.Split()} }", []string{}},
	{
		`package main
		import "go/token"
		type T struct { token int }
		`,
		[]string{"T", `"go/token"`},
	},
	{
		`package main
		import "go/token"
		var _ = struct {token int} {token: 1}
		var _ = []struct {unused int} { {unused: 1}, {unused: 2} }
		`,
		[]string{`"go/token"`},
	},
	{
		`package main
		func init() {
			if i := 1; i == 1 {
			}
		}
		`,
		[]string{},
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
	{
		`package main
		import "io/ioutil"
		type T struct {ioutil string}
		var _ = T{}.ioutil
		`,
		[]string{`"io/ioutil"`},
	},
	{
		`package main
		import "bytes"
		func init() {
			var b bytes.Buffer
		}
		`,
		[]string{"b"},
	},
	{
		`package main
		func main() {
			for i := 0; i <= 10; i++ {
			}
		}
		`,
		[]string{"main"},
	},
	{
		`package main
		func main() {
			for i := 0; true; {
			}
		}
		`,
		[]string{"i", "main"},
	},
	{
		`package main
		import "fmt"
		type iface interface { f(fmt.Stringer); z() }`,
		[]string{"iface"},
	},
}

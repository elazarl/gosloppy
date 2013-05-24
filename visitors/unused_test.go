package visitors

import (
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"testing"

	"github.com/elazarl/gosloppy/scopes"
)

func parse(code string, t *testing.T) (*ast.File, *token.FileSet) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "", code, parser.DeclarationErrors)
	if err != nil {
		t.Fatal("Cannot parse code", err)
	}
	return file, fset
}

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
		scopes.WalkFile(NewUnusedVisitor(unusedNames(func(name string) {
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
		`package visitors
		func f(a int) {
		}
		`,
		[]string{"a", "f"},
	},
	{
		`package visitors
		func f(a int) {
			a = 1
		}
		`,
		[]string{"f"},
	},
	{
		`package visitors
		func f(a int) {
			if true {
				a = 1
			}
		}
		`,
		[]string{"f"},
	},
	{
		`package visitors
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
		`package visitors
		func init() {
			for i := range []int{} {
				println(i)
			}
		}
		`,
		[]string{},
	},
	{"package visitors;import \"strings\";type T struct {A int};func init() *A { return T{A: strings.Split()} }", []string{}},
	{
		`package visitors
		import "go/token"
		type T struct { token int }
		`,
		[]string{"T", `"go/token"`},
	},
	{
		`package visitors
		import "go/token"
		var _ = struct {token int} {token: 1}
		var _ = []struct {unused int} { {unused: 1}, {unused: 2} }
		`,
		[]string{`"go/token"`},
	},
	{
		`package visitors
		func init() {
			if i := 1; i == 1 {
			}
		}
		`,
		[]string{},
	},
	{
		`package visitors
		func f(a int) {
			var _ = func () {
				b := a
			}
		}
		`,
		[]string{"b", "f"},
	},
	{
		`package visitors
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
		`package visitors
		import "fmt"
		`,
		[]string{`"fmt"`},
	},
	{
		`package visitors
		import "fmt"
		var i = fmt.Println
		`,
		[]string{"i"},
	},
	{
		`package visitors
		import "fmt"
		func f(_ fmt.Stringer)
		`,
		[]string{"f"},
	},
	{
		`package visitors
		import "io/ioutil"
		var _ = ioutil.Discard
		`,
		[]string{},
	},
	{
		`package visitors
		import "io/ioutil"
		type T struct {ioutil string}
		var _ = T{}.ioutil
		`,
		[]string{`"io/ioutil"`},
	},
	{
		`package visitors
		import "bytes"
		func init() {
			var b bytes.Buffer
		}
		`,
		[]string{"b"},
	},
	{
		`package visitors
		func init() {
			switch x := 1; x {
			}
		}
		`,
		[]string{},
	},
	{
		`package visitors
		func init() {
			x := []string{}
			for _ = range x {
			}
		}
		`,
		[]string{},
	},
	{
		`package visitors
		func main() {
			for i := 0; i <= 10; i++ {
			}
		}
		`,
		[]string{"main"},
	},
	{
		`package visitors
		func main() {
			for i := 0; true; {
			}
		}
		`,
		[]string{"i", "main"},
	},
	{
		`package visitors
		import "fmt"
		type iface interface { f(fmt.Stringer); z() }`,
		[]string{"iface"},
	},
}

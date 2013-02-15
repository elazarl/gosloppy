package main

import (
	"fmt"
	"go/ast"
	"testing"
)

func equal(a, b []string) {
}

func TestSimpleUnused(t *testing.T) {
	for i, c := range UnusedSimple {
		file, _ := parse(c.body, t)
		unused := []string{}
		UnusedInFile(file, func (obj *ast.Object) {
			unused = append(unused, obj.Name)
		})
		if fmt.Sprint(unused) != fmt.Sprint(c.expUnused) {
			t.Errorf("Case #%d:\n%s\n Expected unused %v got %v", i, c.body, c.expUnused, unused)
		}
	}
}

var UnusedSimple = []struct {
	body string
	expUnused []string
} {
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
}

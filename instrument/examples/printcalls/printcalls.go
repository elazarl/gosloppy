package main

import (
	"go/ast"
	"os"

	"github.com/elazarl/gosloppy/instrument"
	"github.com/elazarl/gosloppy/patch"
)

func main() {
	err := instrument.InstrumentCmd(func(p *patch.PatchableFile) (patches patch.Patches) {
		for _, dec := range p.File.Decls {
			if fun, ok := dec.(*ast.FuncDecl); ok && fun.Name != nil && fun.Body != nil {
				patches = append(patches, patch.Insert(fun.Body.Lbrace+1, "println(`"+fun.Name.Name+"`);"))
			}
		}
		return patches
	}, os.Args...)
	if err != nil {
		println(err.Error())
		os.Exit(-1)
	}
}

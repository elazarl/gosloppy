package main

import (
	"fmt"
	"os"

	"github.com/elazarl/gosloppy/instrument"
	"github.com/elazarl/gosloppy/patch"
)

func usage() {
	fmt.Println(`Usage:
run tests:
gosloppy test <go test switches>
build a binary:
gosloppy build <go build switches>`)
}

func main() {
	if len(os.Args) == 1 {
		usage()
		return
	}
	f := func(p *patch.PatchableFile) patch.Patches {
		patches := &patchUnused{patch.Patches{}}
		autoimport := NewAutoImporter(p.File)
		WalkFile(NewMultiVisitor(NewUnusedVisitor(patches), autoimport), p.File)
		return append(patches.patches, autoimport.Patches...)
	}
	if err := instrument.InstrumentCmd(f, os.Args...); err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
}

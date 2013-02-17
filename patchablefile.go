package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"io/ioutil"
	"sort"
)

type PatchableFile struct {
	Name string
	File *ast.File
	Fset *token.FileSet
	Orig string
}

type Patch struct {
	Pos token.Pos
	Insert string
}

func ParsePatchable(name string) (*PatchableFile, error) {
	fset := token.NewFileSet()
	buf, err := ioutil.ReadFile(name)
	if err != nil {
		return nil, err
	}
	file, err := parser.ParseFile(fset, name, buf, parser.ParseComments)
	if err != nil {
		return nil, err
	}
	return &PatchableFile{name, file, fset, string(buf)}, nil
}

func (p *PatchableFile) Fprint(w io.Writer, nd ast.Node) (int, error) {
	start, end := p.Fset.Position(nd.Pos()), p.Fset.Position(nd.End())
	return io.WriteString(w, p.Orig[start.Offset:end.Offset])
}

type Patches []*Patch

func (p Patches) Len() int { return len(p) }
func (p Patches) Swap(i, j int) { p[i], p[j] = p[j], p[i] }
func (p Patches) Less(i, j int) bool { return p[i].Pos < p[j].Pos }

func sorted(patches []*Patch) Patches {
	sorted := make(Patches, len(patches))
	copy(sorted, patches)
	sort.Sort(sorted)
	return sorted
}

func write(oldn *int, err *error, w io.Writer, s string) {
	var n int
	n, *err = io.WriteString(w, s)
	*oldn += n
	if *err != nil {
		panic(*err)
	}
}

func (p *PatchableFile) FprintPatched(w io.Writer, nd ast.Node, patches []*Patch) (total int, err error) {
	defer func() {
		if r := recover(); r != nil && err == nil {
			panic(r)
		}
	}()
	sorted := sorted(patches)
	start, end := p.Fset.Position(nd.Pos()), p.Fset.Position(nd.End())
	prev := start.Offset
	for _, patch := range sorted {
		if nd.Pos() <= patch.Pos && nd.End() >= patch.Pos {
			pos := p.Fset.Position(patch.Pos)
			write(&total, &err, w, p.Orig[prev:pos.Offset])
			write(&total, &err, w, patch.Insert)
			prev = pos.Offset
		}
	}
	if prev < end.Offset {
		write(&total, &err, w, p.Orig[prev:end.Offset])
	}
	return
}


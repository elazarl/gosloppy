package patch

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
	Start  token.Pos
	End    token.Pos
	Insert string
}

func Insert(pos token.Pos, insert string) *Patch {
	return &Patch{pos, pos, insert}
}

func NewReplacePatch(nd ast.Node, replacement string) *Patch {
	return &Patch{nd.Pos(), nd.End(), replacement}
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

func (p Patches) Len() int           { return len(p) }
func (p Patches) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }
func (p Patches) Less(i, j int) bool { return p[i].Start < p[j].Start }

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

// Write the file with patches applied in that order.
// Note: If patches contradicts each other, behaviour is undefined.
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
		if nd.Pos() <= patch.Start && nd.End() >= patch.Start {
			pos := p.Fset.Position(patch.Start)
			write(&total, &err, w, p.Orig[prev:pos.Offset])
			write(&total, &err, w, patch.Insert)
			prev = p.Fset.Position(patch.End).Offset
		}
	}
	if prev < end.Offset {
		write(&total, &err, w, p.Orig[prev:end.Offset])
	}
	return
}

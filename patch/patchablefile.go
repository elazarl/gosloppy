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
	PkgName  string
	FileName string
	File     *ast.File
	Fset     *token.FileSet
	Orig     string
}

type Patch interface {
	StartPos() token.Pos
	EndPos() token.Pos
}

func (p *BasePatch) StartPos() token.Pos {
	return p.Start
}

func (p *BasePatch) EndPos() token.Pos {
	return p.End
}

type BasePatch struct {
	Start token.Pos
	End   token.Pos
}

type InsertPatch struct {
	BasePatch
	Insert string
}

type InsertNodePatch struct {
	BasePatch
	Insert ast.Node
}

type RemovePatch struct {
	ast.Node
}

func (p RemovePatch) StartPos() token.Pos {
	return p.Pos()
}

func (p RemovePatch) EndPos() token.Pos {
	return p.End()
}

func (p *InsertPatch) StartPos() token.Pos {
	return p.Start
}

func (p *InsertPatch) EndPos() token.Pos {
	return p.End
}

func Insert(pos token.Pos, insert string) Patch {
	return &InsertPatch{BasePatch{pos, pos}, insert}
}

func InsertNode(pos token.Pos, insert ast.Node) Patch {
	return &InsertNodePatch{BasePatch{pos, pos}, insert}
}

func Replace(nd ast.Node, replacement string) Patch {
	return &InsertPatch{BasePatch{nd.Pos(), nd.End()}, replacement}
}

func Remove(nd ast.Node) Patch {
	return RemovePatch{nd}
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
	return &PatchableFile{file.Name.Name, name, file, fset, string(buf)}, nil
}

func (p *PatchableFile) Get(node ast.Node) string {
	return p.Slice(node.Pos(), node.End())
}

func (p *PatchableFile) Slice(from, to token.Pos) string {
	start, end := p.Fset.Position(from), p.Fset.Position(to)
	return p.Orig[start.Offset:end.Offset]
}

func (p *PatchableFile) Fprint(w io.Writer, nd ast.Node) (int, error) {
	start, end := p.Fset.Position(nd.Pos()), p.Fset.Position(nd.End())
	return io.WriteString(w, p.Orig[start.Offset:end.Offset])
}

type Patches []Patch

func (p Patches) Len() int           { return len(p) }
func (p Patches) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }
func (p Patches) Less(i, j int) bool { return p[i].StartPos() < p[j].StartPos() }

func sorted(patches []Patch) Patches {
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
func (p *PatchableFile) FprintPatched(w io.Writer, nd ast.Node, patches []Patch) (total int, err error) {
	defer func() {
		if r := recover(); r != nil && err == nil {
			panic(r)
		}
	}()
	sorted := sorted(patches)
	start, end := p.Fset.Position(nd.Pos()), p.Fset.Position(nd.End())
	prev := start.Offset
	for _, patch := range sorted {
		if nd.Pos() <= patch.StartPos() && nd.End() >= patch.StartPos() {
			pos := p.Fset.Position(patch.StartPos())
			write(&total, &err, w, p.Orig[prev:pos.Offset])
			switch patch := patch.(type) {
			case *InsertPatch:
				write(&total, &err, w, patch.Insert)
			case *InsertNodePatch:
				// TODO(elazar): check performance implications
				noremove := Patches{}
				for _, p := range patches {
					// If the patch removes a certain node
					if p.StartPos() == patch.Insert.Pos() && p.EndPos() == patch.Insert.End() {
						continue
					}
					noremove = append(noremove, p)
				}
				p.FprintPatched(w, patch.Insert, noremove)
			}
			prev = p.Fset.Position(patch.EndPos()).Offset
		}
	}
	if prev < end.Offset {
		write(&total, &err, w, p.Orig[prev:end.Offset])
	}
	return
}

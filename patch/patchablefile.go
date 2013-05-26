package patch

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"io/ioutil"
	"sort"
)

// PatchableFile represents a parsed Go file. You can change the Patchable file,
// and preserve original formatting.
type PatchableFile struct {
	PkgName  string
	FileName string
	File     *ast.File
	Fset     *token.FileSet
	Orig     string
}

// Patch represents a change to a source file between StartPos() and EndPos()
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

// InsertPatch will replace text between Start() and End() with Insert
type InsertPatch struct {
	BasePatch
	Insert string
}

// InsertPatch will replce text between Start() and End() with text representation of Insert node
type InsertNodePatch struct {
	BasePatch
	Insert ast.Node
}

// RemovePatch removes a node from the Go source
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

// Insert returns a patch inserting text txt at position pos
func Insert(pos token.Pos, txt string) Patch {
	return &InsertPatch{BasePatch{pos, pos}, txt}
}

// Insert returns a patch inserting text content of node nd at position pos
func InsertNode(pos token.Pos, nd ast.Node) Patch {
	return &InsertNodePatch{BasePatch{pos, pos}, nd}
}

// Replace rerturns a patch replacing node nd with text replacement
func Replace(nd ast.Node, replacement string) Patch {
	return &InsertPatch{BasePatch{nd.Pos(), nd.End()}, replacement}
}

// Remove returns a patch removing a node from the Go source file
func Remove(nd ast.Node) Patch {
	return RemovePatch{nd}
}

// ParsePatchable parses a singlefile, and return corresponding PatchabeFile
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

// Get returns text corresponding to nd `nd` in file
func (p *PatchableFile) Get(nd ast.Node) string {
	return p.Slice(nd.Pos(), nd.End())
}

// Slice returns text between positions `from` and `to` in file
func (p *PatchableFile) Slice(from, to token.Pos) string {
	start, end := p.Fset.Position(from), p.Fset.Position(to)
	return p.Orig[start.Offset:end.Offset]
}

// Fprint writes the text of node `nd` to writer `w`
func (p *PatchableFile) Fprint(w io.Writer, nd ast.Node) (int, error) {
	start, end := p.Fset.Position(nd.Pos()), p.Fset.Position(nd.End())
	return io.WriteString(w, p.Orig[start.Offset:end.Offset])
}

type nodeSlice struct {
	pos, end token.Pos
}

func (n nodeSlice) Pos() token.Pos {
	return n.pos
}

func (n nodeSlice) End() token.Pos {
	return n.end
}

// All returns an ast.Node corresponding to all text in the file
// note that ast.File does not inlclude comments before package statement
func (p *PatchableFile) All() ast.Node {
	pos, end := p.File.Pos(), p.File.End()
	for _, comment := range p.File.Comments {
		if pos > comment.Pos() {
			pos = comment.Pos()
		}
		if end < comment.End() {
			end = comment.End()
		}
	}
	return nodeSlice{pos, end}
}

type Patches []Patch

type stablePatches struct {
	patches Patches
	perm    []int
}

func (p *stablePatches) Len() int { return len(p.patches) }
func (p *stablePatches) Swap(i, j int) {
	p.patches[i], p.patches[j] = p.patches[j], p.patches[i]
	p.perm[i], p.perm[j] = p.perm[j], p.perm[i]
}

func (p *stablePatches) Less(i, j int) bool {
	return p.patches[i].StartPos() < p.patches[j].StartPos() ||
		p.patches[j].StartPos() == p.patches[i].StartPos() && p.perm[i] < p.perm[j]
}

func sorted(patches []Patch) Patches {

	sorted := &stablePatches{make(Patches, len(patches)), make([]int, len(patches))}
	for i := 0; i < len(sorted.perm); i++ {
		sorted.perm[i] = i
	}
	copy(sorted.patches, patches)
	sort.Sort(sorted)
	return sorted.patches
}

func write(oldn *int, err *error, w io.Writer, s string) {
	var n int
	n, *err = io.WriteString(w, s)
	*oldn += n
	if *err != nil {
		panic(*err)
	}
}

// FprintPatched apply patches to p and write the result to w
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

package patch

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/token"
	"runtime"
	"testing"
)

func parse(code string, t *testing.T) *PatchableFile {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "", code, parser.DeclarationErrors|parser.ParseComments)
	if err != nil {
		t.Fatal("Cannot parse code", err)
	}
	return &PatchableFile{file.Name.Name, "", file, fset, code}
}

func TestPatchableFileNoPatches(t *testing.T) {
	var body = `package main
func

f   ( ) {
        }`
	buf := new(bytes.Buffer)
	patchable := parse(body, t)
	patchable.Fprint(buf, patchable.File)
	if buf.String() != body {
		t.Errorf("Orig ===:\n%s\nCopy ===:\n%s\n PatchableFile differ from orig",
			patchable.Orig, buf.String())
	}
	buf.Reset()
	patchable.Fprint(buf, patchable.File.Decls[0])
	exp :=
		`func

f   ( ) {
        }`
	if buf.String() != exp {
		t.Errorf("%s\n===\n%s\nfunction differ from orig", exp, buf.String())
	}
}

func TestHeaderComment(t *testing.T) {
	buf := new(bytes.Buffer)
	body := "//hoho\npackage main"
	patchable := parse(body, t)
	patchable.FprintPatched(buf, patchable.File, nil)
	if buf.String() != body {
		t.Error("Expected:\n", body, "\nGot:\n", buf.String())
	}
}

var body = `package main
func

f   ( ) {
        }`

func TestPatchableFileSimple(t *testing.T) {
	buf := new(bytes.Buffer)
	patchable := parse(body, t)
	patchable.FprintPatched(buf, patchable.File, Patches{Insert(patchable.File.Decls[0].Pos(), "/* before */")})
	exp :=
		`package main
/* before */func

f   ( ) {
        }`
	if buf.String() != exp {
		t.Errorf("Expected ===:\n%s\nActual ===:\n%s\n PatchableFile differ from expected", exp, buf.String())
	}
	buf.Reset()
	patchable.FprintPatched(buf, patchable.File, Patches{Insert(patchable.File.Decls[0].Pos(), "/* before */"),
		Insert(patchable.File.Decls[0].(*ast.FuncDecl).Name.Pos(), "/* f */"),
		Replace(patchable.File.Decls[0].(*ast.FuncDecl).Name, "g"),
		Insert(patchable.File.Package, "/* package */")})
	exp =
		`/* package */package main
/* before */func

/* f */g   ( ) {
        }`
	if buf.String() != exp {
		t.Errorf("Expected ===:\n%s\nActual ===:\n%s\n PatchableFile differ from expected", exp, buf.String())
	}
	buf.Reset()
	patchable.FprintPatched(buf, patchable.File.Decls[0], Patches{Insert(patchable.File.Decls[0].Pos(), "/* before */")})
	exp =
		`/* before */func

f   ( ) {
        }`
	if buf.String() != exp {
		t.Errorf("%s\n===\n%s\nfunction differ from orig", exp, buf.String())
	}
	buf.Reset()
	patchable.FprintPatched(buf, patchable.File.Decls[0], Patches{
		Insert(patchable.File.Decls[0].Pos(), "/* before */"),
		Insert(patchable.File.Decls[0].(*ast.FuncDecl).Name.Pos(), "/* f */"),
		Insert(patchable.File.Package, "/* import */")})
	exp =
		`/* before */func

/* f */f   ( ) {
        }`
	if buf.String() != exp {
		t.Errorf("%s\n===\n%s\nfunction differ from orig", exp, buf.String())
	}
}

func expect(t *testing.T, node ast.Node, patchable *PatchableFile, exp string, patches ...Patch) {
	buf := new(bytes.Buffer)
	patchable.FprintPatched(buf, node, patches)
	if buf.String() != exp {
		_, filename, line, ok := runtime.Caller(1)
		if !ok {
			panic("Cannot call Caller")
		}
		t.Errorf("\n%s:%d: Expected:\n%s\nGot:\n%s\n", filename, line,
			exp, buf.String(),
		)
	}
}

func TestInsertNode(t *testing.T) {
	patchable := parse(body, t)
	funcbody := patchable.File.Decls[0].(*ast.FuncDecl).Body
	expect(t, funcbody, patchable,
		"main{\n        }", InsertNode(funcbody.Pos(), patchable.File.Name))

	patchable = parse("package kola;func fola()", t)
	expect(t, patchable.File, patchable,
		"package func fola()kola;",
		Remove(patchable.File.Decls[0]),
		InsertNode(patchable.File.Name.Pos(), patchable.File.Decls[0]),
	)
}

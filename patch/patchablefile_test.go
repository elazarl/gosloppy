package patch

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

func parse(code string, t *testing.T) (*ast.File, *token.FileSet) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "", code, parser.DeclarationErrors)
	if err != nil {
		t.Fatal("Cannot parse code", err)
	}
	return file, fset
}

func TestPatchableFileNoPatches(t *testing.T) {
	var body = `package main
func

f   ( ) {
        }`
	buf := new(bytes.Buffer)
	file, fset := parse(body, t)
	patchable := &PatchableFile{file.Name.Name, "", file, fset, body}
	patchable.Fprint(buf, file)
	if buf.String() != body {
		t.Errorf("Orig ===:\n%s\nCopy ===:\n%s\n PatchableFile differ from orig", body, buf.String())
	}
	buf.Reset()
	patchable.Fprint(buf, file.Decls[0])
	exp :=
		`func

f   ( ) {
        }`
	if buf.String() != exp {
		t.Errorf("%s\n===\n%s\nfunction differ from orig", exp, buf.String())
	}
}

func TestPatchableFileSimple(t *testing.T) {
	var body = `package main
func

f   ( ) {
        }`
	buf := new(bytes.Buffer)
	file, fset := parse(body, t)
	patchable := &PatchableFile{file.Name.Name, "", file, fset, body}
	patchable.FprintPatched(buf, file, Patches{Insert(file.Decls[0].Pos(), "/* before */")})
	exp :=
		`package main
/* before */func

f   ( ) {
        }`
	if buf.String() != exp {
		t.Errorf("Expected ===:\n%s\nActual ===:\n%s\n PatchableFile differ from expected", exp, buf.String())
	}
	buf.Reset()
	patchable.FprintPatched(buf, file, Patches{Insert(file.Decls[0].Pos(), "/* before */"),
		Insert(file.Decls[0].(*ast.FuncDecl).Name.Pos(), "/* f */"),
		Replace(file.Decls[0].(*ast.FuncDecl).Name, "g"),
		Insert(file.Package, "/* package */")})
	exp =
		`/* package */package main
/* before */func

/* f */g   ( ) {
        }`
	if buf.String() != exp {
		t.Errorf("Expected ===:\n%s\nActual ===:\n%s\n PatchableFile differ from expected", exp, buf.String())
	}
	buf.Reset()
	patchable.FprintPatched(buf, file.Decls[0], Patches{Insert(file.Decls[0].Pos(), "/* before */")})
	exp =
		`/* before */func

f   ( ) {
        }`
	if buf.String() != exp {
		t.Errorf("%s\n===\n%s\nfunction differ from orig", exp, buf.String())
	}
	buf.Reset()
	patchable.FprintPatched(buf, file.Decls[0], Patches{
		Insert(file.Decls[0].Pos(), "/* before */"),
		Insert(file.Decls[0].(*ast.FuncDecl).Name.Pos(), "/* f */"),
		Insert(file.Package, "/* import */")})
	exp =
		`/* before */func

/* f */f   ( ) {
        }`
	if buf.String() != exp {
		t.Errorf("%s\n===\n%s\nfunction differ from orig", exp, buf.String())
	}
}

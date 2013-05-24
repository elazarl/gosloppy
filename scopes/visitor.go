package scopes

import (
	"fmt"
	"go/ast"
	"go/token"
	"log"
)

type Visitor interface {
	VisitExpr(scope *ast.Scope, expr ast.Expr) (w Visitor)
	VisitStmt(scope *ast.Scope, stmt ast.Stmt) (w Visitor)
	VisitDecl(scope *ast.Scope, stmt ast.Decl) (w Visitor)
	// TODO(elazar): rethink the API, we probably want to give here a list of scopes
	ExitScope(scope *ast.Scope, parent ast.Node, last bool) (w Visitor)
}

// We traverse types, since we need them to determine if import is used
func WalkFields(v Visitor, fields []*ast.Field, scope *ast.Scope) {
	for _, field := range fields {
		for _, name := range field.Names {
			insertToScope(scope, name.Obj)
		}
		WalkExpr(v, field.Type, scope)
	}
}

func WalkExpr(v Visitor, expr ast.Expr, scope *ast.Scope) {
	if v = v.VisitExpr(scope, expr); v == nil {
		return
	}
	switch expr := expr.(type) {
	case *ast.Ellipsis:
		if expr.Elt != nil {
			WalkExpr(v, expr.Elt, scope)
		}
	case *ast.FuncLit:
		newscope := ast.NewScope(scope)
		if expr.Type.Params != nil {
			WalkFields(v, expr.Type.Params.List, newscope)
		}
		if expr.Type.Results != nil {
			WalkFields(v, expr.Type.Results.List, newscope)
		}
		WalkStmt(v, expr.Body, newscope)
		v.ExitScope(newscope, expr, true)
	case *ast.BadExpr:
		// nothing to do
	case *ast.ParenExpr:
		WalkExpr(v, expr.X, scope)
	case *ast.SelectorExpr:
		WalkExpr(v, expr.X, scope)
		WalkExpr(v, expr.Sel, scope)
	case *ast.IndexExpr:
		WalkExpr(v, expr.X, scope)
		WalkExpr(v, expr.Index, scope)
	case *ast.SliceExpr:
		WalkExpr(v, expr.X, scope)
		if expr.Low != nil {
			WalkExpr(v, expr.Low, scope)
		}
		if expr.High != nil {
			WalkExpr(v, expr.High, scope)
		}
	case *ast.TypeAssertExpr:
		WalkExpr(v, expr.X, scope)
		if expr.Type != nil {
			WalkExpr(v, expr.Type, scope)
		}
	case *ast.CallExpr:
		WalkExpr(v, expr.Fun, scope)
		for _, e := range expr.Args {
			WalkExpr(v, e, scope)
		}
	case *ast.StarExpr:
		WalkExpr(v, expr.X, scope)
	case *ast.UnaryExpr:
		WalkExpr(v, expr.X, scope)
	case *ast.BinaryExpr:
		WalkExpr(v, expr.X, scope)
		WalkExpr(v, expr.Y, scope)
	case *ast.CompositeLit:
		for _, elt := range expr.Elts {
			WalkExpr(v, elt, scope)
		}
		// For example, in v = []struct{i int} {{1}, {2}}
		// the inner `{1}` and inner `{2}` are two composite literals
		// with no `Type`.
		if expr.Type != nil {
			WalkExpr(v, expr.Type, scope)
		}
	case *ast.KeyValueExpr:
		WalkExpr(v, expr.Key, scope)
		WalkExpr(v, expr.Value, scope)
	case *ast.ArrayType:
		WalkExpr(v, expr.Elt, scope)
		if expr.Len != nil {
			WalkExpr(v, expr.Len, scope)
		}
	case *ast.ChanType:
		WalkExpr(v, expr.Value, scope)
	case *ast.MapType:
		WalkExpr(v, expr.Key, scope)
		WalkExpr(v, expr.Value, scope)
	case *ast.StructType:
		for _, field := range expr.Fields.List {
			// TODO: think of a proper way to walk through field names and let visitor
			//       know you're in a struct type
			WalkExpr(v, field.Type, scope)
		}
	case *ast.FuncType:
		for _, field := range expr.Params.List {
			WalkExpr(v, field.Type, scope)
		}
		if expr.Results != nil {
			for _, field := range expr.Results.List {
				WalkExpr(v, field.Type, scope)
			}
		}
	case *ast.InterfaceType:
		for _, field := range expr.Methods.List {
			// names don't currently interest you
			WalkExpr(v, field.Type, scope)
		}
	case *ast.Ident, *ast.BasicLit:
	default:
		panic(fmt.Sprintf("Canot understand %#v", expr))
	}
}

func WalkStmt(v Visitor, stmt ast.Stmt, scope *ast.Scope) (newscope *ast.Scope) {
	newscope = scope
	if v = v.VisitStmt(scope, stmt); v == nil {
		return
	}
	switch stmt := stmt.(type) {
	case *ast.ExprStmt:
		WalkExpr(v, stmt.X, scope)
	case *ast.IncDecStmt:
		WalkExpr(v, stmt.X, scope)
	case *ast.ReturnStmt:
		for _, expr := range stmt.Results {
			WalkExpr(v, expr, scope)
		}
	case *ast.AssignStmt:
		if stmt.Tok == token.DEFINE {
			newscope = ast.NewScope(scope)
			for _, expr := range stmt.Rhs {
				WalkExpr(v, expr, scope)
			}
			for _, expr := range stmt.Lhs {
				insertToScope(newscope, expr.(*ast.Ident).Obj)
			}
		} else {
			for _, expr := range stmt.Lhs {
				WalkExpr(v, expr, scope)
			}
			for _, expr := range stmt.Rhs {
				WalkExpr(v, expr, scope)
			}
		}
	case *ast.DeclStmt:
		switch decl := stmt.Decl.(type) {
		case *ast.GenDecl:
			for _, spec := range decl.Specs {
				newscope = ast.NewScope(newscope)
				switch spec := spec.(type) {
				case *ast.TypeSpec:
					insertToScope(newscope, spec.Name.Obj)
					WalkExpr(v, spec.Type, scope)
				case *ast.ValueSpec:
					for _, name := range spec.Names {
						insertToScope(newscope, name.Obj)
					}
					for _, value := range spec.Values {
						WalkExpr(v, value, scope)
					}
					if spec.Type != nil {
						WalkExpr(v, spec.Type, newscope)
					}
				default:
					panic("cannot have an import in a statement (or so I hope)")
				}
			}
		default:
			panic("only GenDecl can appear in statement")
		}
	case *ast.SendStmt:
		WalkExpr(v, stmt.Chan, scope)
		WalkExpr(v, stmt.Value, scope)
	case *ast.DeferStmt:
		WalkExpr(v, stmt.Call, scope)
	case *ast.GoStmt:
		WalkExpr(v, stmt.Call, scope)
	case *ast.LabeledStmt:
		WalkStmt(v, stmt.Stmt, scope)
	case *ast.BranchStmt:
		// nothing to do
	case *ast.IfStmt:
		inner := scope
		if stmt.Init != nil {
			inner = WalkStmt(v, stmt.Init, inner)
		}
		WalkExpr(v, stmt.Cond, inner)
		WalkStmt(v, stmt.Body, inner)
		if stmt.Else != nil {
			WalkStmt(v, stmt.Else, inner)
		}
		exitScopes(v, inner, scope, stmt)
	case *ast.ForStmt:
		inner := scope
		if stmt.Init != nil {
			inner = WalkStmt(v, stmt.Init, inner)
		}
		if stmt.Cond != nil {
			WalkExpr(v, stmt.Cond, inner)
		}
		if stmt.Post != nil {
			WalkStmt(v, stmt.Post, inner)
		}
		WalkStmt(v, stmt.Body, inner)
		exitScopes(v, inner, scope, stmt)
	case *ast.RangeStmt:
		inner := scope
		if stmt.Tok == token.ASSIGN {
			WalkExpr(v, stmt.Key, scope)
			// For example, in
			//     for a := range []int{1, 2, 3} {}
			// will have Value == nil
			if stmt.Value != nil {
				WalkExpr(v, stmt.Value, scope)
			}
		} else if stmt.Tok == token.DEFINE {
			inner = ast.NewScope(inner)
			insertToScope(inner, stmt.Key.(*ast.Ident).Obj)
			if stmt.Value != nil {
				insertToScope(inner, stmt.Value.(*ast.Ident).Obj)
			}
		} else {
			panic("range statement must have := or = token")
		}
		WalkExpr(v, stmt.X, inner)
		WalkStmt(v, stmt.Body, inner)
		exitScopes(v, inner, scope, stmt)
	case *ast.CaseClause:
		inner := ast.NewScope(scope)
		for _, expr := range stmt.List {
			WalkExpr(v, expr, scope)
		}
		for _, s := range stmt.Body {
			inner = WalkStmt(v, s, inner)
		}
		exitScopes(v, inner, scope, stmt)
	case *ast.SwitchStmt:
		inner := scope
		if stmt.Init != nil {
			inner = WalkStmt(v, stmt.Init, inner)
		}
		if stmt.Tag != nil {
			WalkExpr(v, stmt.Tag, inner)
		}
		WalkStmt(v, stmt.Body, inner)
		exitScopes(v, inner, scope, stmt)
	case *ast.TypeSwitchStmt:
		inner := scope
		if stmt.Init != nil {
			inner = WalkStmt(v, stmt.Init, inner)
		}
		inner = WalkStmt(v, stmt.Assign, inner)
		WalkStmt(v, stmt.Body, inner)
		exitScopes(v, inner, scope, stmt)
	case *ast.CommClause:
		// Usually: case <- a: cmd1;cmd2;
		// if stmt.Comm == nil: default:
		inner := scope
		if stmt.Comm != nil {
			inner = WalkStmt(v, stmt.Comm, scope)
		}
		for _, s := range stmt.Body {
			inner = WalkStmt(v, s, inner)
		}
		exitScopes(v, inner, scope, stmt)
	case *ast.SelectStmt:
		WalkStmt(v, stmt.Body, scope)
	case *ast.BlockStmt:
		inner := ast.NewScope(scope)
		for _, s := range stmt.List {
			inner = WalkStmt(v, s, inner)
		}
		exitScopes(v, inner, scope, stmt)
	default:
		log.Panicf("Cannot understand %+#v", stmt)
	}
	return
}

func exitScopes(v Visitor, inner, limit *ast.Scope, parent ast.Stmt) {
	for inner != limit {
		if inner == nil {
			panic("exitScopes must be bounded")
		}
		v.ExitScope(inner, parent, inner.Outer == limit)
		inner = inner.Outer
	}
}

func WalkFile(v Visitor, file *ast.File) {
	if v == nil {
		return
	}
	for _, d := range file.Decls {
		w := v.VisitDecl(file.Scope, d)
		if w == nil {
			continue
		}
		switch d := d.(type) {
		case *ast.FuncDecl:
			scope := ast.NewScope(file.Scope)
			// Note that reciever might be anonymous, e.g. crypto/elliptic/p224.go:78
			// func (p224Curve) Add(bigX1, bigY1, bigX2, bigY2 *big.Int) (x, y *big.Int) {
			if d.Recv != nil && len(d.Recv.List) > 0 && len(d.Recv.List[0].Names) > 0 {
				insertToScope(scope, d.Recv.List[0].Names[0].Obj)
			}
			// Params is always non-nil, since we always have parens, and need to know their pos
			WalkFields(w, d.Type.Params.List, scope)
			if d.Type.Results != nil {
				WalkFields(w, d.Type.Results.List, scope)
			}
			// see http://golang.org/ref/spec#Function_declarations
			// "A function declaration may omit the body.
			//  Such a declaration provides the signature for a function implemented outside Go,
			//  such as an assembly routine."
			// for example sigpipe at os/file_posix.go
			if d.Body != nil {
				WalkStmt(w, d.Body, scope)
			}
			w.ExitScope(scope, d, true)
		case *ast.GenDecl:
			for _, spec := range d.Specs {
				switch spec := spec.(type) {
				case *ast.ValueSpec:
					// already in scope insertToScope(file.Scope, spec.Names)
					for _, value := range spec.Values {
						WalkExpr(w, value, file.Scope)
					}
					if spec.Type != nil {
						WalkExpr(w, spec.Type, file.Scope)
					}
				case *ast.TypeSpec:
					// TODO: think what to do with the name, see above
					WalkExpr(w, spec.Type, file.Scope)
				}
			}
		}
	}
	v.ExitScope(file.Scope, file, true)
}

func insertToScope(scope *ast.Scope, obj *ast.Object) {
	if obj.Name == "_" {
		return
	}
	if obj.Kind == ast.Fun && obj.Name == "init" {
		return
	}
	scope.Insert(obj)
}

func Lookup(scope *ast.Scope, name string) *ast.Object {
	for scope != nil {
		if obj := scope.Lookup(name); obj != nil {
			return obj
		}
		scope = scope.Outer
	}
	return nil
}

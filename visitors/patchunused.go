package visitors

import (
	"go/ast"

	"github.com/elazarl/gosloppy/patch"
)

type PatchUnused struct {
	Patches patch.Patches
}

// TL;DR compareAssgn(x,y) should implement internalInterfacePointer(x) == internalInterfacePointer(y)
//
// Before jumping into the wrong conclusions, let me explain the purpose of the sad state of this function
// In Go a case clause can have a scope, for example:
//     select {
//     case scoper := <- c:
//     /* this is a []stmt which is in the scope of scoper */
//     }
// When we report that the assignment statement "scoper := <- c" is unused, we need to check whether or not
// this statement is in a CommClause or in the statement list afterwards.
// Since Go doesn't have (AFAIK) ability to compare pointers in two `interface{}`s, I implement it in an
// awkward way.
// Why don't we have this problem in, say for statment? This is because the for statement always have a
// block statement afterwards, therefor the "parent" object we report to UnusedObj is the block statment,
// thus, we can always be sure if we have an unused variable whose parent is a block statement - that this
// is the Init part of the for statement, ditto for if, etc.
func compareAssgn(lhs interface{}, rhs ast.Stmt) bool {
	switch lhs := lhs.(type) {
	case *ast.AssignStmt:
		rhs, ok := rhs.(*ast.AssignStmt)
		return ok && rhs == lhs
	case *ast.DeclStmt:
		rhs, ok := rhs.(*ast.DeclStmt)
		return ok && rhs == lhs
	}
	return false
}

// UnusedObj will add relevant patch into p (i.e. `; _ = i` if i is unused,
// if the unused object needs to be patched (for instance, unused function
// arguments does not need to be patched)
func (p *PatchUnused) UnusedObj(obj *ast.Object, parent ast.Node) {
	// if the unused variable is a function argument, or TLD - ignore
	switch obj.Decl.(type) {
	case *ast.Field, *ast.GenDecl, *ast.TypeSpec:
		return
	case *ast.ValueSpec:
		if _, ok := parent.(*ast.File); ok {
			return
		}
	}
	if obj.Kind == ast.Fun {
		return
	}
	exempter := "_ = " + obj.Name
	switch parent := parent.(type) {
	case *ast.ForStmt:
		p.Patches = append(p.Patches, patch.Insert(parent.Body.Lbrace+1, exempter+";"))
	case *ast.RangeStmt:
		p.Patches = append(p.Patches, patch.Insert(parent.Body.Lbrace+1, exempter+";"))
	case *ast.TypeSwitchStmt:
		if len(parent.Body.List) == 0 {
			p.Patches = append(p.Patches, patch.Insert(parent.Body.Lbrace+1, "default: "+exempter))
		} else {
			// if first statement is not case statement - it won't compile anyhow
			if stmt, ok := parent.Body.List[0].(*ast.CaseClause); ok {
				p.Patches = append(p.Patches, patch.Insert(stmt.Colon+1, exempter+";"))
			}
		}
	case *ast.SwitchStmt:
		if len(parent.Body.List) == 0 {
			p.Patches = append(p.Patches, patch.Insert(parent.Body.Lbrace+1, "default: "+exempter))
		} else {
			// if first statement is not case statement - it won't compile anyhow
			if stmt, ok := parent.Body.List[0].(*ast.CaseClause); ok {
				p.Patches = append(p.Patches, patch.Insert(stmt.Colon+1, exempter+";"))
			}
		}
	case *ast.CommClause:
		if compareAssgn(obj.Decl, parent.Comm) {
			p.Patches = append(p.Patches, patch.Insert(parent.Colon+1, exempter+";"))
		} else {
			p.Patches = append(p.Patches, patch.Insert(obj.Decl.(ast.Node).End(), ";"+exempter))
		}
	case *ast.IfStmt:
		p.Patches = append(p.Patches, patch.Insert(parent.Body.Lbrace+1, exempter+";"))
	default:
		p.Patches = append(p.Patches, patch.Insert(obj.Decl.(ast.Node).End(), ";"+exempter))
	}
}

// UnusedImport adds relevant patch (_ before import path) to fix unused import error
func (p *PatchUnused) UnusedImport(imp *ast.ImportSpec) {
	if imp.Name != nil {
		p.Patches = append(p.Patches, patch.Replace(imp.Name, "_"))
	} else {
		p.Patches = append(p.Patches, patch.Insert(imp.Pos(), "_ "))
	}
}

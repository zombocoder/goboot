// Package compiler loads Go packages, associates annotation comments with the
// declarations they document, resolves type information, and validates the
// result against the annotation registry. It implements phases 2–5 of the
// compiler pipeline (specification §37.2–§37.5): package loading, annotation
// scanning, comment association, and the type lookup the semantic analyzer and
// dependency resolver build upon.
package compiler

import (
	"go/ast"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/packages"

	"github.com/zombocoder/goboot/annotation"
)

// Declaration is a single Go declaration that carries one or more annotations,
// together with the resolved type information downstream phases need.
//
// Exactly one of the typed handles (TypeName, Func, Field) is non-nil,
// selected by Target: type/struct/interface targets set TypeName; function and
// method targets set Func; field targets set Field. Package-target
// declarations leave all three nil.
type Declaration struct {
	// Name is the declared identifier (the type, function, method, or field
	// name; empty for a package target).
	Name string
	// PkgPath is the import path of the package the declaration belongs to.
	PkgPath string
	// Target is the kind of declaration, used for schema target validation.
	Target annotation.Target
	// Annotations are the parsed annotations attached to this declaration, in
	// source order, with positions resolved to true source locations.
	Annotations []annotation.Annotation
	// Pos is the source position of the declaration's name (or the package
	// clause for a package target).
	Pos token.Position

	// TypeName is set for type/struct/interface targets.
	TypeName *types.TypeName
	// Func is set for function and method targets.
	Func *types.Func
	// Field is set for field targets.
	Field *types.Var
	// Recv is the receiver's named type for a method target, else nil.
	Recv *types.TypeName

	// Node is the underlying AST node (*ast.TypeSpec, *ast.FuncDecl,
	// *ast.Field, or *ast.File), retained for later analyzers that need
	// syntax-level detail such as struct tags or method bodies.
	Node ast.Node
	// Pkg is the loaded package this declaration came from.
	Pkg *packages.Package
}

// Has reports whether the declaration carries an annotation with the given
// name.
func (d *Declaration) Has(name string) bool {
	for i := range d.Annotations {
		if d.Annotations[i].Name == name {
			return true
		}
	}
	return false
}

// Find returns the first annotation with the given name and true, or a zero
// Annotation and false. When an annotation is repeatable, use FindAll.
func (d *Declaration) Find(name string) (annotation.Annotation, bool) {
	for i := range d.Annotations {
		if d.Annotations[i].Name == name {
			return d.Annotations[i], true
		}
	}
	return annotation.Annotation{}, false
}

// FindAll returns every annotation with the given name in source order.
func (d *Declaration) FindAll(name string) []annotation.Annotation {
	var out []annotation.Annotation
	for i := range d.Annotations {
		if d.Annotations[i].Name == name {
			out = append(out, d.Annotations[i])
		}
	}
	return out
}

// Package is a loaded package together with the annotated declarations found in
// it, in deterministic source order.
type Package struct {
	Pkg          *packages.Package
	Declarations []*Declaration
}

// ScanResult is the output of a Load: the loaded packages, a flattened list of
// every annotated declaration across them, and all diagnostics produced during
// loading, parsing, and validation.
type ScanResult struct {
	Packages     []*Package
	Declarations []*Declaration
	Diagnostics  []*annotation.Diagnostic
}

// HasErrors reports whether any diagnostic is error severity.
func (r *ScanResult) HasErrors() bool {
	for _, d := range r.Diagnostics {
		if d.Severity == annotation.SeverityError {
			return true
		}
	}
	return false
}

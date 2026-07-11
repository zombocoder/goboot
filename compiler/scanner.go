package compiler

import (
	"go/ast"
	"go/token"
	"go/types"
	"sort"

	"golang.org/x/tools/go/packages"

	"github.com/zombocoder/goboot/annotation"
)

// scanner walks package syntax trees and associates annotation comments with
// the declarations they document (§37.3). It accumulates diagnostics that the
// caller drains after each package with takeDiagnostics.
type scanner struct {
	registry *annotation.Registry
	diags    []*annotation.Diagnostic
}

func newScanner(registry *annotation.Registry) *scanner {
	return &scanner{registry: registry}
}

// takeDiagnostics returns the diagnostics accumulated since the last call and
// resets the buffer.
func (s *scanner) takeDiagnostics() []*annotation.Diagnostic {
	d := s.diags
	s.diags = nil
	return d
}

// scanPackage scans every file in pkg and returns its annotated declarations in
// deterministic source order.
func (s *scanner) scanPackage(pkg *packages.Package) *Package {
	out := &Package{Pkg: pkg}

	// Process files in filename order so results do not depend on the loader's
	// syntax ordering (§6.7).
	files := append([]*ast.File(nil), pkg.Syntax...)
	sort.Slice(files, func(i, j int) bool {
		return pkg.Fset.Position(files[i].Pos()).Filename <
			pkg.Fset.Position(files[j].Pos()).Filename
	})

	for _, file := range files {
		s.scanFile(pkg, file, out)
	}
	return out
}

func (s *scanner) scanFile(pkg *packages.Package, file *ast.File, out *Package) {
	// Package-level annotations live on the package clause's doc comment.
	if decl := s.declFromDoc(pkg, file.Doc, annotation.TargetPackage, pkg.Name,
		pkg.Fset.Position(file.Package), file); decl != nil {
		out.Declarations = append(out.Declarations, decl)
	}

	for _, d := range file.Decls {
		switch decl := d.(type) {
		case *ast.GenDecl:
			if decl.Tok == token.TYPE {
				s.scanTypeDecl(pkg, decl, out)
			}
		case *ast.FuncDecl:
			s.scanFuncDecl(pkg, decl, out)
		}
	}
}

// scanTypeDecl handles a `type (...)` declaration: each type spec, plus struct
// fields and interface methods within it.
func (s *scanner) scanTypeDecl(pkg *packages.Package, gen *ast.GenDecl, out *Package) {
	for _, spec := range gen.Specs {
		ts, ok := spec.(*ast.TypeSpec)
		if !ok {
			continue
		}
		// A single ungrouped spec carries its doc on the GenDecl.
		doc := ts.Doc
		if doc == nil && len(gen.Specs) == 1 {
			doc = gen.Doc
		}

		target := targetForType(ts.Type)
		obj, _ := pkg.TypesInfo.Defs[ts.Name].(*types.TypeName)
		if decl := s.declFromDoc(pkg, doc, target, ts.Name.Name,
			pkg.Fset.Position(ts.Name.Pos()), ts); decl != nil {
			decl.TypeName = obj
			out.Declarations = append(out.Declarations, decl)
		}

		switch t := ts.Type.(type) {
		case *ast.StructType:
			s.scanFields(pkg, t.Fields, annotation.TargetField, out)
		case *ast.InterfaceType:
			s.scanFields(pkg, t.Methods, annotation.TargetMethod, out)
		}
	}
}

// scanFields associates annotations with struct fields or interface methods.
func (s *scanner) scanFields(pkg *packages.Package, fields *ast.FieldList, target annotation.Target, out *Package) {
	if fields == nil {
		return
	}
	for _, field := range fields.List {
		if len(field.Names) == 0 {
			continue // embedded field or embedded interface: no name to bind
		}
		for _, name := range field.Names {
			decl := s.declFromDoc(pkg, field.Doc, target, name.Name,
				pkg.Fset.Position(name.Pos()), field)
			if decl == nil {
				continue
			}
			switch target {
			case annotation.TargetField:
				decl.Field, _ = pkg.TypesInfo.Defs[name].(*types.Var)
			case annotation.TargetMethod:
				decl.Func, _ = pkg.TypesInfo.Defs[name].(*types.Func)
			}
			out.Declarations = append(out.Declarations, decl)
		}
	}
}

// scanFuncDecl handles a function or method declaration.
func (s *scanner) scanFuncDecl(pkg *packages.Package, fn *ast.FuncDecl, out *Package) {
	target := annotation.TargetFunction
	if fn.Recv != nil {
		target = annotation.TargetMethod
	}
	decl := s.declFromDoc(pkg, fn.Doc, target, fn.Name.Name,
		pkg.Fset.Position(fn.Name.Pos()), fn)
	if decl == nil {
		return
	}
	decl.Func, _ = pkg.TypesInfo.Defs[fn.Name].(*types.Func)
	if decl.Func != nil {
		decl.Recv = receiverNamed(decl.Func)
	}
	out.Declarations = append(out.Declarations, decl)
}

// declFromDoc parses the doc comment, validates the annotations, and returns a
// Declaration when at least one annotation is present. It records parse and
// validation diagnostics regardless. It returns nil when there are no
// annotations.
func (s *scanner) declFromDoc(pkg *packages.Package, doc *ast.CommentGroup, target annotation.Target, name string, pos token.Position, node ast.Node) *Declaration {
	anns, diags := parseDoc(doc, pkg.Fset)
	s.diags = append(s.diags, diags...)
	if len(anns) == 0 {
		return nil
	}
	s.diags = append(s.diags, s.registry.ValidateGroup(anns, target)...)
	return &Declaration{
		Name:        name,
		PkgPath:     pkg.PkgPath,
		Target:      target,
		Annotations: anns,
		Pos:         pos,
		Node:        node,
		Pkg:         pkg,
	}
}

// targetForType maps a type spec's underlying syntax to an annotation target.
func targetForType(expr ast.Expr) annotation.Target {
	switch expr.(type) {
	case *ast.StructType:
		return annotation.TargetStruct
	case *ast.InterfaceType:
		return annotation.TargetInterface
	default:
		return annotation.TargetType
	}
}

// receiverNamed returns the named type of a method's receiver, unwrapping a
// pointer receiver, or nil if it cannot be determined.
func receiverNamed(fn *types.Func) *types.TypeName {
	sig, ok := fn.Type().(*types.Signature)
	if !ok || sig.Recv() == nil {
		return nil
	}
	t := sig.Recv().Type()
	if ptr, ok := t.(*types.Pointer); ok {
		t = ptr.Elem()
	}
	if named, ok := t.(*types.Named); ok {
		return named.Obj()
	}
	return nil
}

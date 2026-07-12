package model

import (
	"go/token"
	"go/types"

	"github.com/zombocoder/goboot/annotation"
)

// AnnotatedDecl is a scanned declaration together with the annotations attached
// to it (§46.5). The core model turns known annotations into structured fields
// (components, routes, ...) but otherwise discards them; AnnotatedDecl surfaces
// the raw annotations — including plugin-registered ones — so plugin Analyzers
// and Generators can drive their own behavior. Declarations that carry no
// annotations are omitted.
type AnnotatedDecl struct {
	// Package is the import path of the declaring package.
	Package string
	// Name is the declared identifier (type/function/method/field name; empty
	// for a package target).
	Name string
	// Receiver is the receiver type name for a method, empty otherwise.
	Receiver string
	// Target is the kind of declaration the annotations are attached to.
	Target annotation.Target
	// Annotations are the parsed annotations, in source order.
	Annotations []annotation.Annotation
	// Position is the source location of the declaration.
	Position token.Position
	// Signature is the function/method signature for method and function
	// targets, enabling plugins to inspect parameter and result types (e.g. a
	// message-handler payload); nil for other targets.
	Signature *types.Signature
}

// Find returns the first annotation with the given name and whether it exists.
func (d AnnotatedDecl) Find(name string) (annotation.Annotation, bool) {
	for _, a := range d.Annotations {
		if a.Name == name {
			return a, true
		}
	}
	return annotation.Annotation{}, false
}

// Has reports whether the declaration carries the named annotation.
func (d AnnotatedDecl) Has(name string) bool {
	_, ok := d.Find(name)
	return ok
}

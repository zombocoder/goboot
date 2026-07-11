package model

import (
	"go/token"
	"go/types"
)

// Dependency is a single requirement of a constructor or nut provider (§14.2).
// Before resolution only the syntactic fields are set; the resolver fills
// ResolvedTo.
type Dependency struct {
	// Name is the parameter name at the injection site.
	Name string
	// Type is the declared parameter type the resolver must satisfy.
	Type types.Type
	// Qualifier optionally narrows resolution to a named component (§14.6). MVP
	// leaves this empty; it is reserved for @Named/@Qualifier support.
	Qualifier string
	// Optional reports whether an unresolved dependency is tolerated. MVP
	// dependencies are all required (false).
	Optional bool
	// Position is the source location of the parameter.
	Position token.Position
	// ResolvedTo is the component chosen to satisfy this dependency, filled by
	// the resolver. Empty until resolved (or if resolution failed).
	ResolvedTo ComponentID
}

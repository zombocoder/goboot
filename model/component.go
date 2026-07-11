// Package model defines the intermediate application model produced by semantic
// analysis and consumed by the code generators (specification §38). It holds
// only data — components, constructors, dependencies, and the assembled
// application — and depends on nothing beyond the standard go/types and
// go/token packages so that it can be shared freely across the compiler, graph,
// and generator layers without import cycles.
package model

import (
	"go/token"
	"go/types"
)

// Scope is a component's lifecycle scope (§12.4). Only singleton and prototype
// exist in the MVP.
type Scope uint8

const (
	// ScopeSingleton is a single shared instance, the default.
	ScopeSingleton Scope = iota
	// ScopePrototype is a fresh instance per injection point.
	ScopePrototype
)

func (s Scope) String() string {
	switch s {
	case ScopeSingleton:
		return "singleton"
	case ScopePrototype:
		return "prototype"
	default:
		return "unknown"
	}
}

// ComponentKind classifies a component by the annotation that declared it
// (§12.1).
type ComponentKind uint8

const (
	// ComponentGeneric is a plain @Component.
	ComponentGeneric ComponentKind = iota
	// ComponentService is an @Service.
	ComponentService
	// ComponentRepository is a @Repository.
	ComponentRepository
	// ComponentController is a @RestController.
	ComponentController
	// ComponentConfiguration is a @Configuration.
	ComponentConfiguration
	// ComponentBean is produced by an @Bean provider function.
	ComponentBean
	// ComponentAdvice is a @ControllerAdvice.
	ComponentAdvice
)

func (k ComponentKind) String() string {
	switch k {
	case ComponentGeneric:
		return "component"
	case ComponentService:
		return "service"
	case ComponentRepository:
		return "repository"
	case ComponentController:
		return "controller"
	case ComponentConfiguration:
		return "configuration"
	case ComponentBean:
		return "bean"
	case ComponentAdvice:
		return "advice"
	default:
		return "unknown"
	}
}

// Component is a single injectable unit in the application graph (§12.2).
type Component struct {
	// ID uniquely and stably identifies the component (§12.3).
	ID ComponentID
	// Name is the component's declared name (its type or bean function name),
	// or an explicit name from the annotation.
	Name string
	// PackagePath is the import path of the package declaring the component.
	PackagePath string
	// ProvidedType is the type made available for injection: the constructor's
	// first return type (e.g. *UserService, or an interface for a bean).
	ProvidedType types.Type
	// Named is the underlying named type when ProvidedType resolves to one,
	// used for interface-satisfaction checks. May be nil (e.g. bean returning
	// an unnamed type).
	Named *types.Named
	// Kind is the declaring annotation's category.
	Kind ComponentKind
	// Scope is the lifecycle scope.
	Scope Scope
	// Primary marks the preferred candidate when a dependency is ambiguous
	// (§14.5).
	Primary bool
	// Constructor builds the component. Always set for MVP components.
	Constructor *Constructor
	// Dependencies mirrors Constructor.Params after resolution, in order.
	Dependencies []Dependency
	// Position is the source location of the component declaration.
	Position token.Position
}

// DependsOn returns the resolved component IDs this component requires, in
// constructor-parameter order. Unresolved dependencies are skipped.
func (c *Component) DependsOn() []ComponentID {
	out := make([]ComponentID, 0, len(c.Dependencies))
	for _, d := range c.Dependencies {
		if d.ResolvedTo != "" {
			out = append(out, d.ResolvedTo)
		}
	}
	return out
}

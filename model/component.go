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
	// ComponentNut is produced by an @Nut provider function.
	ComponentNut
	// ComponentAdvice is a @ControllerAdvice.
	ComponentAdvice
	// ComponentConfigProperties is an @ConfigurationProperties struct loaded
	// from configuration rather than constructed.
	ComponentConfigProperties
	// ComponentProxy is a generated interface proxy that wraps a target
	// component to apply method interception (§24).
	ComponentProxy
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
	case ComponentNut:
		return "nut"
	case ComponentAdvice:
		return "advice"
	case ComponentConfigProperties:
		return "configuration-properties"
	case ComponentProxy:
		return "proxy"
	default:
		return "unknown"
	}
}

// LifecycleMethod describes an @PostConstruct or @PreDestroy method (§30). The
// generator adapts the four supported signatures (§30.2) to a uniform hook.
type LifecycleMethod struct {
	// MethodName is the method to invoke on the component instance.
	MethodName string
	// TakesContext reports whether the method accepts a context.Context.
	TakesContext bool
	// ReturnsError reports whether the method returns an error.
	ReturnsError bool
}

// Component is a single injectable unit in the application graph (§12.2).
type Component struct {
	// ID uniquely and stably identifies the component (§12.3).
	ID ComponentID
	// Name is the component's declared name (its type or nut function name),
	// or an explicit name from the annotation.
	Name string
	// PackagePath is the import path of the package declaring the component.
	PackagePath string
	// ProvidedType is the type made available for injection: the constructor's
	// first return type (e.g. *UserService, or an interface for a nut).
	ProvidedType types.Type
	// Named is the underlying named type when ProvidedType resolves to one,
	// used for interface-satisfaction checks. May be nil (e.g. nut returning
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
	// ConfigPrefix is the @ConfigurationProperties prefix for a
	// ComponentConfigProperties component; empty otherwise.
	ConfigPrefix string
	// Interface is the interface a proxied service is exposed as (§24.2); nil
	// for non-proxied components. On a target it names the interface the proxy
	// implements; on a ComponentProxy it is the provided interface type.
	Interface types.Type
	// Proxied reports that this component's methods are intercepted and it is
	// therefore reached only through its generated proxy (§24.3).
	Proxied bool
	// Intercepted lists the target's methods that carry interception
	// annotations, in declaration order; empty for non-proxied components.
	Intercepted []InterceptedMethod
	// ProxyTarget is set on a ComponentProxy: the ID of the concrete component
	// it wraps.
	ProxyTarget ComponentID
	// Repository is set on a ComponentRepository whose implementation is
	// generated from @Query/@Exec methods (§27.2); nil for component-mode
	// repositories.
	Repository *RepositoryInfo
	// Conditions are the profile and conditional requirements that gate the
	// component's inclusion (§29).
	Conditions Conditions
	// PostConstruct is the component's @PostConstruct hook, or nil.
	PostConstruct *LifecycleMethod
	// PreDestroy is the component's @PreDestroy hook, or nil.
	PreDestroy *LifecycleMethod
	// Scheduled lists the component's @Scheduled methods, in declaration order.
	Scheduled []ScheduledMethod
	// Position is the source location of the component declaration.
	Position token.Position
}

// HasLifecycle reports whether the component declares any lifecycle hook.
func (c *Component) HasLifecycle() bool {
	return c.PostConstruct != nil || c.PreDestroy != nil
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

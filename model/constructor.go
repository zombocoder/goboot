package model

import (
	"go/token"
	"go/types"
)

// Constructor describes how a component is built (§13). It covers both the
// NewXxx naming convention and @Nut provider functions, which share the same
// signature rules (§16).
type Constructor struct {
	// PackagePath is the import path of the package declaring the function.
	PackagePath string
	// PackageName is the declaring package's name, used by the generator for
	// import aliasing (the function's package may differ from the provided
	// type's package, as with a nut returning another package's interface).
	PackageName string
	// FuncName is the constructor or nut function name, e.g. NewUserService or
	// ProvideDatabase.
	FuncName string
	// Params are the constructor's parameters, each an injection dependency, in
	// declaration order.
	Params []Dependency
	// ReturnType is the type of the constructor's first (value) return.
	ReturnType types.Type
	// ReturnsError reports whether the constructor's final return is an error
	// (§13.3), which the generated wiring must check.
	ReturnsError bool
	// IsNut reports whether the constructor is an @Nut provider rather than a
	// NewXxx constructor.
	IsNut bool
	// Constructorless reports that the component has no constructor function and
	// is built with a zero-value composite literal (§13.5). When true, FuncName
	// is empty and Params is nil.
	Constructorless bool
	// ConfigLoader reports that the component is loaded from configuration by a
	// generated loader rather than constructed (§28). When true, the generator
	// emits a Load<Type> function and constructs the component from the
	// application's config.Source.
	ConfigLoader bool
	// RepositoryImpl reports that the component is a generated repository
	// implementation (§27) constructed from a db.DBProvider.
	RepositoryImpl bool
	// Position is the source location of the function declaration.
	Position token.Position
}

// Qualified returns the package-qualified function name for diagnostics.
func (c *Constructor) Qualified() string {
	return c.PackagePath + "." + c.FuncName
}

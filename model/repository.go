package model

import "go/types"

// QueryKind distinguishes a read query from a write/exec (§27.3).
type QueryKind uint8

const (
	// QueryRead is an @Query returning rows.
	QueryRead QueryKind = iota
	// QueryExec is an @Exec that modifies data.
	QueryExec
)

// ReturnShape classifies a repository method's return so the generator can emit
// the right scan and result handling (§27.6).
type ReturnShape struct {
	// Multi reports a slice return ([]T or []*T).
	Multi bool
	// Pointer reports that the element type is a pointer (*T).
	Pointer bool
	// Scalar reports that the element is a scalar (int64/string/bool/...) rather
	// than a struct entity.
	Scalar bool
	// RowsAffected reports an exec returning (int64, error).
	RowsAffected bool
	// Elem is the element type: the entity struct or scalar, with any pointer
	// and slice stripped. Nil for an exec returning only error.
	Elem types.Type
}

// RepositoryMethod is a single generated query or exec method (§27.1).
type RepositoryMethod struct {
	// Name is the interface method name.
	Name string
	// RawSQL is the annotation's SQL with named parameters; the generator
	// compiles it with the configured dialect.
	RawSQL string
	// Kind is read (@Query) or exec (@Exec).
	Kind QueryKind
	// Return classifies the method's result.
	Return ReturnShape
	// Signature is the method's type signature, used to render parameters.
	Signature *types.Signature
}

// RepositoryInfo holds the generated methods of an @Repository(generate=true)
// interface.
type RepositoryInfo struct {
	Methods []RepositoryMethod
}

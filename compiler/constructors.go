package compiler

import (
	"fmt"
	"go/token"
	"go/types"

	"github.com/zombocoder/goboot/annotation"
	"github.com/zombocoder/goboot/model"
)

// Dependency-injection diagnostic codes (GOBDI family, §39.4).
const (
	// CodeMissingDependency is a constructor parameter no component satisfies.
	CodeMissingDependency = "GOBDI001"
	// CodeAmbiguousDependency is a parameter satisfied by more than one
	// component with no primary or qualifier to disambiguate.
	CodeAmbiguousDependency = "GOBDI002"
	// CodeMissingConstructor is a component with required fields but no
	// discoverable constructor.
	CodeMissingConstructor = "GOBDI003"
	// CodeInvalidConstructor is a constructor whose signature is unsupported.
	CodeInvalidConstructor = "GOBDI004"
	// CodeDependencyCycle is a cycle in the component graph.
	CodeDependencyCycle = "GOBDI005"
	// CodeApplicationRoot is a missing or duplicated @Application declaration.
	CodeApplicationRoot = "GOBDI006"
)

// namedOf unwraps a pointer and returns the underlying *types.Named, or nil.
func namedOf(t types.Type) *types.Named {
	if ptr, ok := t.(*types.Pointer); ok {
		t = ptr.Elem()
	}
	if named, ok := t.(*types.Named); ok {
		return named
	}
	return nil
}

// lookupConstructor returns the NewXxx constructor function for a named type
// declared in its own package, or nil if none exists.
func lookupConstructor(tn *types.TypeName) *types.Func {
	if tn == nil || tn.Pkg() == nil {
		return nil
	}
	obj := tn.Pkg().Scope().Lookup("New" + tn.Name())
	fn, _ := obj.(*types.Func)
	return fn
}

// buildConstructor validates a constructor or nut function signature and
// converts it into a model.Constructor. It returns diagnostics for unsupported
// signatures (§13.4); when a fatal signature error is found it returns a nil
// constructor.
func buildConstructor(fn *types.Func, isNut bool, fset *token.FileSet) (*model.Constructor, []*annotation.Diagnostic) {
	sig, ok := fn.Type().(*types.Signature)
	if !ok {
		return nil, nil
	}
	pos := fset.Position(fn.Pos())
	var diags []*annotation.Diagnostic

	if sig.Variadic() {
		diags = append(diags, diagErr(CodeInvalidConstructor, pos,
			"constructor %s must not be variadic", fn.Name()))
		return nil, diags
	}

	results := sig.Results()
	switch results.Len() {
	case 1:
		// value only
	case 2:
		if !isErrorType(results.At(1).Type()) {
			diags = append(diags, diagErr(CodeInvalidConstructor, pos,
				"constructor %s second return value must be error, found %s",
				fn.Name(), results.At(1).Type()))
			return nil, diags
		}
	default:
		diags = append(diags, diagErr(CodeInvalidConstructor, pos,
			"constructor %s must return (T) or (T, error), found %d return values",
			fn.Name(), results.Len()))
		return nil, diags
	}

	ctor := &model.Constructor{
		PackagePath:  pkgPathOf(fn),
		PackageName:  pkgNameOf(fn),
		FuncName:     fn.Name(),
		ReturnType:   results.At(0).Type(),
		ReturnsError: results.Len() == 2,
		IsNut:        isNut,
		Position:     pos,
	}
	ctor.Params = paramsToDeps(sig.Params(), fset)
	return ctor, diags
}

// constructorlessFor builds a synthetic constructor for a zero-field struct
// component (§13.5), or returns nil if the type is not a constructorless
// candidate.
func constructorlessFor(tn *types.TypeName, fset *token.FileSet) *model.Constructor {
	st, ok := tn.Type().Underlying().(*types.Struct)
	if !ok || st.NumFields() != 0 {
		return nil
	}
	return &model.Constructor{
		PackagePath:     tn.Pkg().Path(),
		PackageName:     tn.Pkg().Name(),
		ReturnType:      types.NewPointer(tn.Type()),
		Constructorless: true,
		Position:        fset.Position(tn.Pos()),
	}
}

// paramsToDeps converts a signature's parameter tuple into ordered
// dependencies.
func paramsToDeps(params *types.Tuple, fset *token.FileSet) []model.Dependency {
	deps := make([]model.Dependency, 0, params.Len())
	for i := 0; i < params.Len(); i++ {
		p := params.At(i)
		deps = append(deps, model.Dependency{
			Name:     p.Name(),
			Type:     p.Type(),
			Position: fset.Position(p.Pos()),
		})
	}
	return deps
}

// isErrorType reports whether t is the built-in error interface.
func isErrorType(t types.Type) bool {
	return types.Identical(t, types.Universe.Lookup("error").Type())
}

// pkgPathOf returns the import path of a function's package, or "" for builtins.
func pkgPathOf(fn *types.Func) string {
	if fn.Pkg() == nil {
		return ""
	}
	return fn.Pkg().Path()
}

// pkgNameOf returns the name of a function's package, or "" for builtins.
func pkgNameOf(fn *types.Func) string {
	if fn.Pkg() == nil {
		return ""
	}
	return fn.Pkg().Name()
}

// diagErr is a convenience for building an error-severity diagnostic.
func diagErr(code string, pos token.Position, format string, args ...any) *annotation.Diagnostic {
	return &annotation.Diagnostic{
		Severity: annotation.SeverityError,
		Code:     code,
		Message:  fmt.Sprintf(format, args...),
		Position: pos,
	}
}

package annotation

// Target identifies the kind of Go declaration an annotation may be attached to.
//
// See specification §9.4.
type Target uint8

const (
	// TargetPackage is a package declaration (package-level doc comment).
	TargetPackage Target = iota
	// TargetType is any named type declaration.
	TargetType
	// TargetStruct is a struct type declaration.
	TargetStruct
	// TargetInterface is an interface type declaration.
	TargetInterface
	// TargetFunction is a package-level function.
	TargetFunction
	// TargetMethod is a method with a receiver.
	TargetMethod
	// TargetField is a struct field.
	TargetField
	// TargetParameter is a function or method parameter.
	TargetParameter
)

// String returns the human-readable name of the target, used in diagnostics.
func (t Target) String() string {
	switch t {
	case TargetPackage:
		return "package"
	case TargetType:
		return "type"
	case TargetStruct:
		return "struct"
	case TargetInterface:
		return "interface"
	case TargetFunction:
		return "function"
	case TargetMethod:
		return "method"
	case TargetField:
		return "field"
	case TargetParameter:
		return "parameter"
	default:
		return "unknown"
	}
}

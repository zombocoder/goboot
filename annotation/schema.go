package annotation

import (
	"go/token"
	"slices"
	"sort"
	"strings"
	"time"
)

// ArgumentType constrains the kind of value a named or positional argument may
// hold. It is checked during schema validation (§9.5).
type ArgumentType uint8

const (
	// ArgString requires a quoted string value.
	ArgString ArgumentType = iota
	// ArgInteger requires an integer value.
	ArgInteger
	// ArgFloat requires a float; an integer literal is accepted and widened.
	ArgFloat
	// ArgBoolean requires true or false.
	ArgBoolean
	// ArgIdentifier requires an unquoted identifier.
	ArgIdentifier
	// ArgDuration requires a string parsable by time.ParseDuration.
	ArgDuration
	// ArgStringOrIdent accepts either a quoted string or an identifier, so that
	// enum-like arguments may be written as mode="any" or mode=any.
	ArgStringOrIdent
	// ArgStringArray requires an array whose every element is a string.
	ArgStringArray
	// ArgObject requires an object value.
	ArgObject
	// ArgAny accepts a value of any kind.
	ArgAny
)

func (t ArgumentType) String() string {
	switch t {
	case ArgString:
		return "string"
	case ArgInteger:
		return "integer"
	case ArgFloat:
		return "float"
	case ArgBoolean:
		return "boolean"
	case ArgIdentifier:
		return "identifier"
	case ArgDuration:
		return "duration"
	case ArgStringOrIdent:
		return "string or identifier"
	case ArgStringArray:
		return "array of strings"
	case ArgObject:
		return "object"
	case ArgAny:
		return "any"
	default:
		return "unknown"
	}
}

// ArgumentDefinition describes one argument of an annotation (§9.5).
type ArgumentDefinition struct {
	Type     ArgumentType
	Required bool
	// Default is applied by consumers when the argument is omitted; validation
	// does not require it. May be nil.
	Default Value
	// Allowed, when non-empty, restricts string/identifier values to this set.
	Allowed []string
}

// Validator is an optional cross-argument check attached to a Definition. It
// runs after per-argument validation succeeds and returns any diagnostics.
type Validator interface {
	Validate(ann Annotation) []*Diagnostic
}

// ValidatorFunc adapts a function to the Validator interface.
type ValidatorFunc func(ann Annotation) []*Diagnostic

func (f ValidatorFunc) Validate(ann Annotation) []*Diagnostic { return f(ann) }

// Definition is the registered schema for one annotation name (§9.5).
type Definition struct {
	Name    string
	Targets []Target
	// Arguments maps named-argument names to their definitions.
	Arguments map[string]ArgumentDefinition
	// Positional describes the single positional argument, e.g. @Timeout("2s").
	// Nil means positional arguments are not allowed.
	Positional *ArgumentDefinition
	// Repeatable reports whether the annotation may appear more than once on the
	// same declaration (e.g. @Response).
	Repeatable bool
	Validator  Validator
}

// AllowsTarget reports whether the annotation may be applied to target.
func (d *Definition) AllowsTarget(target Target) bool {
	return slices.Contains(d.Targets, target)
}

// argNames returns the declared named-argument names, for typo suggestions.
func (d *Definition) argNames() []string {
	out := make([]string, 0, len(d.Arguments))
	for n := range d.Arguments {
		out = append(out, n)
	}
	return out
}

// validate checks a single annotation instance against this definition,
// returning diagnostics for every problem found. target is the declaration the
// annotation was attached to.
func (d *Definition) validate(ann Annotation, target Target) []*Diagnostic {
	var diags []*Diagnostic

	if len(d.Targets) > 0 && !d.AllowsTarget(target) {
		diags = append(diags, newError(CodeInvalidTarget, ann.Position,
			"annotation @%s cannot be applied to a %s (allowed: %s)",
			d.Name, target, targetsString(d.Targets)))
	}

	for name, val := range ann.Arguments {
		if isPositionalKey(name) {
			if d.Positional == nil {
				diags = append(diags, newError(CodeUnknownArgument, ann.Position,
					"annotation @%s does not accept a positional argument", d.Name))
				continue
			}
			diags = appendArgDiags(diags, d.Name, name, *d.Positional, val, ann.Position)
			continue
		}
		def, ok := d.Arguments[name]
		if !ok {
			diags = append(diags, newError(CodeUnknownArgument, ann.Position,
				"unknown argument %q in annotation @%s%s", name, d.Name, didYouMean(name, "", d.argNames())))
			continue
		}
		diags = appendArgDiags(diags, d.Name, name, def, val, ann.Position)
	}

	for name, def := range d.Arguments {
		if def.Required {
			if _, ok := ann.Arguments[name]; !ok {
				diags = append(diags, newError(CodeMissingArgument, ann.Position,
					"annotation @%s requires argument %q", d.Name, name))
			}
		}
	}
	if d.Positional != nil && d.Positional.Required {
		if _, ok := ann.Positional(); !ok {
			diags = append(diags, newError(CodeMissingArgument, ann.Position,
				"annotation @%s requires a positional argument", d.Name))
		}
	}

	if d.Validator != nil && len(diags) == 0 {
		diags = append(diags, d.Validator.Validate(ann)...)
	}
	return diags
}

// appendArgDiags validates a single argument value against its definition.
func appendArgDiags(diags []*Diagnostic, annName, argName string, def ArgumentDefinition, val Value, pos token.Position) []*Diagnostic {
	if !typeMatches(def.Type, val) {
		return append(diags, newError(CodeArgumentType, pos,
			"argument %q of annotation @%s must be %s, found %s",
			argName, annName, def.Type, val.Kind()))
	}
	if len(def.Allowed) > 0 {
		if s, ok := AsString(val); ok && !slices.Contains(def.Allowed, s) {
			return append(diags, newError(CodeArgumentValue, pos,
				"argument %q of annotation @%s must be one of [%s], found %q",
				argName, annName, strings.Join(def.Allowed, ", "), s))
		}
	}
	return diags
}

// typeMatches reports whether val satisfies the expected argument type.
func typeMatches(t ArgumentType, val Value) bool {
	switch t {
	case ArgString:
		return val.Kind() == ValueString
	case ArgInteger:
		return val.Kind() == ValueInteger
	case ArgFloat:
		return val.Kind() == ValueFloat || val.Kind() == ValueInteger
	case ArgBoolean:
		return val.Kind() == ValueBoolean
	case ArgIdentifier:
		return val.Kind() == ValueIdentifier
	case ArgDuration:
		s, ok := val.(StringValue)
		if !ok {
			return false
		}
		_, err := time.ParseDuration(s.Val)
		return err == nil
	case ArgStringOrIdent:
		return val.Kind() == ValueString || val.Kind() == ValueIdentifier
	case ArgStringArray:
		arr, ok := val.(ArrayValue)
		if !ok {
			return false
		}
		for _, e := range arr.Elements {
			if e.Kind() != ValueString {
				return false
			}
		}
		return true
	case ArgObject:
		return val.Kind() == ValueObject
	case ArgAny:
		return true
	default:
		return false
	}
}

func isPositionalKey(name string) bool {
	return name == PositionalKey || strings.HasPrefix(name, PositionalKey) &&
		len(name) > len(PositionalKey) && name[len(PositionalKey)] >= '0' && name[len(PositionalKey)] <= '9'
}

func targetsString(targets []Target) string {
	names := make([]string, len(targets))
	for i, t := range targets {
		names[i] = t.String()
	}
	sort.Strings(names)
	return strings.Join(names, ", ")
}

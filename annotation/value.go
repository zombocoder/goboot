package annotation

import (
	"sort"
	"strconv"
	"strings"
)

// ValueKind enumerates the kinds of values an annotation argument may hold.
//
// See specification §9.3.
type ValueKind uint8

const (
	// ValueString is a quoted string, e.g. "userService" or a raw `...` string.
	ValueString ValueKind = iota
	// ValueInteger is a whole number, e.g. 3.
	ValueInteger
	// ValueFloat is a floating-point number, e.g. 2.0.
	ValueFloat
	// ValueBoolean is true or false.
	ValueBoolean
	// ValueArray is an ordered list of values, e.g. ["admin", "support"].
	ValueArray
	// ValueObject is a set of named values, e.g. {enabled=true, size=10}.
	ValueObject
	// ValueIdentifier is an unquoted bare word, e.g. singleton or any.
	ValueIdentifier
	// ValueNull is the null literal.
	ValueNull
)

// String returns the human-readable name of the kind, used in diagnostics.
func (k ValueKind) String() string {
	switch k {
	case ValueString:
		return "string"
	case ValueInteger:
		return "integer"
	case ValueFloat:
		return "float"
	case ValueBoolean:
		return "boolean"
	case ValueArray:
		return "array"
	case ValueObject:
		return "object"
	case ValueIdentifier:
		return "identifier"
	case ValueNull:
		return "null"
	default:
		return "unknown"
	}
}

// Value is a parsed annotation argument value.
//
// Implementations are immutable. GoString renders a value back into a
// deterministic, annotation-like textual form; it is used by tests and by
// diagnostics and must not depend on map iteration order.
type Value interface {
	Kind() ValueKind
	// GoString renders the value into deterministic source-like text.
	GoString() string
}

// StringValue holds a string literal. Raw reports whether the literal was
// written with backticks (a raw string), which matters for multi-line SQL in
// @Query/@Exec annotations.
type StringValue struct {
	Val string
	Raw bool
}

func (StringValue) Kind() ValueKind { return ValueString }

func (v StringValue) GoString() string {
	if v.Raw {
		return "`" + v.Val + "`"
	}
	return strconv.Quote(v.Val)
}

// IntValue holds an integer literal.
type IntValue struct{ Val int64 }

func (IntValue) Kind() ValueKind    { return ValueInteger }
func (v IntValue) GoString() string { return strconv.FormatInt(v.Val, 10) }

// FloatValue holds a floating-point literal.
type FloatValue struct{ Val float64 }

func (FloatValue) Kind() ValueKind { return ValueFloat }
func (v FloatValue) GoString() string {
	return strconv.FormatFloat(v.Val, 'g', -1, 64)
}

// BoolValue holds a boolean literal.
type BoolValue struct{ Val bool }

func (BoolValue) Kind() ValueKind    { return ValueBoolean }
func (v BoolValue) GoString() string { return strconv.FormatBool(v.Val) }

// IdentValue holds an unquoted identifier used as an enum-like value.
type IdentValue struct{ Name string }

func (IdentValue) Kind() ValueKind    { return ValueIdentifier }
func (v IdentValue) GoString() string { return v.Name }

// NullValue is the null literal.
type NullValue struct{}

func (NullValue) Kind() ValueKind  { return ValueNull }
func (NullValue) GoString() string { return "null" }

// ArrayValue holds an ordered list of values.
type ArrayValue struct{ Elements []Value }

func (ArrayValue) Kind() ValueKind { return ValueArray }

func (v ArrayValue) GoString() string {
	parts := make([]string, len(v.Elements))
	for i, e := range v.Elements {
		parts[i] = e.GoString()
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

// ObjectValue holds a set of named values. Fields are stored in a map; GoString
// sorts keys so that rendering is deterministic regardless of parse order.
type ObjectValue struct{ Fields map[string]Value }

func (ObjectValue) Kind() ValueKind { return ValueObject }

func (v ObjectValue) GoString() string {
	keys := make([]string, 0, len(v.Fields))
	for k := range v.Fields {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, len(keys))
	for i, k := range keys {
		parts[i] = k + "=" + v.Fields[k].GoString()
	}
	return "{" + strings.Join(parts, ", ") + "}"
}

// AsString returns the string content of a StringValue or IdentValue and true,
// or "" and false for any other kind. It is a convenience for schema consumers
// that accept either form (e.g. mode=any and mode="any").
func AsString(v Value) (string, bool) {
	switch t := v.(type) {
	case StringValue:
		return t.Val, true
	case IdentValue:
		return t.Name, true
	default:
		return "", false
	}
}

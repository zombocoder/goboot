package annotation

import "testing"

func TestValueGoString(t *testing.T) {
	tests := []struct {
		val  Value
		want string
	}{
		{StringValue{Val: "hi"}, `"hi"`},
		{StringValue{Val: "raw", Raw: true}, "`raw`"},
		{IntValue{Val: -5}, "-5"},
		{FloatValue{Val: 2.5}, "2.5"},
		{BoolValue{Val: true}, "true"},
		{BoolValue{Val: false}, "false"},
		{IdentValue{Name: "singleton"}, "singleton"},
		{NullValue{}, "null"},
		{ArrayValue{Elements: []Value{StringValue{Val: "a"}, IntValue{Val: 1}}}, `["a", 1]`},
		{ObjectValue{Fields: map[string]Value{"b": IntValue{Val: 2}, "a": IntValue{Val: 1}}}, "{a=1, b=2}"},
	}
	for _, tt := range tests {
		if got := tt.val.GoString(); got != tt.want {
			t.Errorf("GoString() = %q, want %q", got, tt.want)
		}
	}
}

func TestValueKindString(t *testing.T) {
	kinds := []ValueKind{
		ValueString, ValueInteger, ValueFloat, ValueBoolean,
		ValueArray, ValueObject, ValueIdentifier, ValueNull, ValueKind(99),
	}
	want := []string{"string", "integer", "float", "boolean", "array", "object", "identifier", "null", "unknown"}
	for i, k := range kinds {
		if got := k.String(); got != want[i] {
			t.Errorf("ValueKind(%d).String() = %q, want %q", k, got, want[i])
		}
	}
}

func TestTargetString(t *testing.T) {
	targets := []Target{
		TargetPackage, TargetType, TargetStruct, TargetInterface,
		TargetFunction, TargetMethod, TargetField, TargetParameter, Target(99),
	}
	want := []string{"package", "type", "struct", "interface", "function", "method", "field", "parameter", "unknown"}
	for i, tg := range targets {
		if got := tg.String(); got != want[i] {
			t.Errorf("Target(%d).String() = %q, want %q", tg, got, want[i])
		}
	}
}

func TestArgumentTypeAndSeverityString(t *testing.T) {
	if ArgDuration.String() != "duration" || ArgStringArray.String() != "array of strings" {
		t.Errorf("ArgumentType.String mismatch")
	}
	if ArgumentType(99).String() != "unknown" {
		t.Errorf("unknown ArgumentType string")
	}
	if SeverityInfo.String() != "info" || SeverityWarning.String() != "warning" || SeverityError.String() != "error" {
		t.Errorf("Severity.String mismatch")
	}
	if Severity(99).String() != "unknown" {
		t.Errorf("unknown Severity string")
	}
}

func TestAsString(t *testing.T) {
	if s, ok := AsString(StringValue{Val: "x"}); !ok || s != "x" {
		t.Errorf("AsString(string) = %q,%v", s, ok)
	}
	if s, ok := AsString(IdentValue{Name: "y"}); !ok || s != "y" {
		t.Errorf("AsString(ident) = %q,%v", s, ok)
	}
	if _, ok := AsString(IntValue{Val: 1}); ok {
		t.Errorf("AsString(int) should be false")
	}
}

func TestRegistryLookup(t *testing.T) {
	r := DefaultRegistry()
	if _, ok := r.Lookup("Service"); !ok {
		t.Errorf("expected Service to be registered")
	}
	if _, ok := r.Lookup("Nonexistent"); ok {
		t.Errorf("did not expect Nonexistent to be registered")
	}
}

func TestMustRegisterPanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Errorf("MustRegister should panic on duplicate")
		}
	}()
	r := NewRegistry()
	r.MustRegister(&Definition{Name: "Dup", Targets: []Target{TargetStruct}})
	r.MustRegister(&Definition{Name: "Dup", Targets: []Target{TargetStruct}})
}

func TestObjectStringArrayTypeMismatch(t *testing.T) {
	// An object where a string array is expected must fail typeMatches.
	if typeMatches(ArgStringArray, ObjectValue{Fields: map[string]Value{}}) {
		t.Errorf("object should not match string-array type")
	}
	if typeMatches(ArgObject, ObjectValue{Fields: map[string]Value{}}) == false {
		t.Errorf("object should match object type")
	}
	if !typeMatches(ArgAny, NullValue{}) {
		t.Errorf("ArgAny should accept anything")
	}
	if typeMatches(ArgDuration, IntValue{Val: 1}) {
		t.Errorf("non-string should not match duration")
	}
}

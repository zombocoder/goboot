package annotation

import (
	"go/token"
	"testing"
)

func basePos() token.Position {
	return token.Position{Filename: "test.go", Offset: 0, Line: 1, Column: 1}
}

// parseSingle parses text expected to contain exactly one well-formed
// annotation and returns it, failing the test on any diagnostic.
func parseSingle(t *testing.T, text string) Annotation {
	t.Helper()
	anns, diags := ParseComment(text, basePos())
	if len(diags) != 0 {
		t.Fatalf("unexpected diagnostics for %q: %v", text, diags)
	}
	if len(anns) != 1 {
		t.Fatalf("expected 1 annotation for %q, got %d", text, len(anns))
	}
	return anns[0]
}

func TestParseMarker(t *testing.T) {
	ann := parseSingle(t, "@Service")
	if ann.Name != "Service" {
		t.Fatalf("name = %q, want Service", ann.Name)
	}
	if len(ann.Arguments) != 0 {
		t.Fatalf("expected no arguments, got %v", ann.Arguments)
	}
}

func TestParseNamedArguments(t *testing.T) {
	ann := parseSingle(t, `@Service(name="userService", scope="singleton")`)
	if got, _ := AsString(ann.Arguments["name"]); got != "userService" {
		t.Fatalf("name = %q, want userService", got)
	}
	if got, _ := AsString(ann.Arguments["scope"]); got != "singleton" {
		t.Fatalf("scope = %q, want singleton", got)
	}
}

func TestParseValueKinds(t *testing.T) {
	tests := []struct {
		text string
		arg  string
		kind ValueKind
	}{
		{`@X(a="s")`, "a", ValueString},
		{`@X(a=3)`, "a", ValueInteger},
		{`@X(a=-7)`, "a", ValueInteger},
		{`@X(a=2.5)`, "a", ValueFloat},
		{`@X(a=1e3)`, "a", ValueFloat},
		{`@X(a=true)`, "a", ValueBoolean},
		{`@X(a=false)`, "a", ValueBoolean},
		{`@X(a=null)`, "a", ValueNull},
		{`@X(a=singleton)`, "a", ValueIdentifier},
		{`@X(a=["x","y"])`, "a", ValueArray},
		{`@X(a={k=1})`, "a", ValueObject},
	}
	for _, tt := range tests {
		ann := parseSingle(t, tt.text)
		v, ok := ann.Arguments[tt.arg]
		if !ok {
			t.Errorf("%s: missing arg %q", tt.text, tt.arg)
			continue
		}
		if v.Kind() != tt.kind {
			t.Errorf("%s: kind = %v, want %v", tt.text, v.Kind(), tt.kind)
		}
	}
}

func TestParseIntAndFloatValues(t *testing.T) {
	ann := parseSingle(t, `@Retry(maxAttempts=3, delay="100ms", multiplier=2.0)`)
	if iv, ok := ann.Arguments["maxAttempts"].(IntValue); !ok || iv.Val != 3 {
		t.Fatalf("maxAttempts = %v, want 3", ann.Arguments["maxAttempts"])
	}
	if fv, ok := ann.Arguments["multiplier"].(FloatValue); !ok || fv.Val != 2.0 {
		t.Fatalf("multiplier = %v, want 2.0", ann.Arguments["multiplier"])
	}
}

func TestParseArray(t *testing.T) {
	ann := parseSingle(t, `@Authorize(roles=["admin", "support"])`)
	arr, ok := ann.Arguments["roles"].(ArrayValue)
	if !ok {
		t.Fatalf("roles is %T, want ArrayValue", ann.Arguments["roles"])
	}
	if len(arr.Elements) != 2 {
		t.Fatalf("len(roles) = %d, want 2", len(arr.Elements))
	}
	if s, _ := AsString(arr.Elements[0]); s != "admin" {
		t.Fatalf("roles[0] = %q, want admin", s)
	}
}

func TestParseObject(t *testing.T) {
	ann := parseSingle(t, `@Custom(options={enabled=true, size=10})`)
	obj, ok := ann.Arguments["options"].(ObjectValue)
	if !ok {
		t.Fatalf("options is %T, want ObjectValue", ann.Arguments["options"])
	}
	if bv, ok := obj.Fields["enabled"].(BoolValue); !ok || !bv.Val {
		t.Fatalf("enabled = %v, want true", obj.Fields["enabled"])
	}
	if iv, ok := obj.Fields["size"].(IntValue); !ok || iv.Val != 10 {
		t.Fatalf("size = %v, want 10", obj.Fields["size"])
	}
	// GoString must be deterministic (sorted keys).
	if got := obj.GoString(); got != "{enabled=true, size=10}" {
		t.Fatalf("GoString = %q", got)
	}
}

func TestParsePositional(t *testing.T) {
	tests := []struct {
		text string
		kind ValueKind
	}{
		{`@Timeout("2s")`, ValueString},
		{`@ResponseStatus(404)`, ValueInteger},
		{`@Profile(["production", "staging"])`, ValueArray},
	}
	for _, tt := range tests {
		ann := parseSingle(t, tt.text)
		v, ok := ann.Positional()
		if !ok {
			t.Errorf("%s: no positional argument", tt.text)
			continue
		}
		if v.Kind() != tt.kind {
			t.Errorf("%s: positional kind = %v, want %v", tt.text, v.Kind(), tt.kind)
		}
	}
}

func TestParseRawStringMultiline(t *testing.T) {
	text := "@Query(`\n  SELECT id, name\n  FROM users\n  WHERE id = :id\n`)"
	ann := parseSingle(t, text)
	pv, _ := ann.Positional()
	sv, ok := pv.(StringValue)
	if !ok {
		t.Fatalf("positional is %T, want StringValue", pv)
	}
	if !sv.Raw {
		t.Fatalf("expected raw string")
	}
	if want := "\n  SELECT id, name\n  FROM users\n  WHERE id = :id\n"; sv.Val != want {
		t.Fatalf("raw content = %q, want %q", sv.Val, want)
	}
}

func TestParseMultilineAnnotation(t *testing.T) {
	text := "@Authorize(\n  roles=[\"admin\", \"support\"],\n  mode=\"any\"\n)"
	ann := parseSingle(t, text)
	if ann.Name != "Authorize" {
		t.Fatalf("name = %q", ann.Name)
	}
	if arr, ok := ann.Arguments["roles"].(ArrayValue); !ok || len(arr.Elements) != 2 {
		t.Fatalf("roles = %v", ann.Arguments["roles"])
	}
	if s, _ := AsString(ann.Arguments["mode"]); s != "any" {
		t.Fatalf("mode = %q, want any", s)
	}
}

func TestParseMultipleAnnotations(t *testing.T) {
	text := "@GetMapping(path=\"/{id}\")\n@Authorize(roles=[\"users.read\"])\n@Response(status=200)"
	anns, diags := ParseComment(text, basePos())
	if len(diags) != 0 {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if len(anns) != 3 {
		t.Fatalf("got %d annotations, want 3", len(anns))
	}
	names := []string{anns[0].Name, anns[1].Name, anns[2].Name}
	want := []string{"GetMapping", "Authorize", "Response"}
	for i := range want {
		if names[i] != want[i] {
			t.Fatalf("annotation[%d] = %q, want %q", i, names[i], want[i])
		}
	}
}

func TestProseIgnored(t *testing.T) {
	text := "UserService coordinates user use cases.\nContact team@example.com for details.\n@Service"
	anns, diags := ParseComment(text, basePos())
	if len(diags) != 0 {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if len(anns) != 1 || anns[0].Name != "Service" {
		t.Fatalf("got %v, want single @Service", anns)
	}
}

func TestPositionTracking(t *testing.T) {
	// Third line, indented by two spaces: '@' is at column 3.
	text := "doc line\n\n  @Service(name=\"x\")"
	anns, _ := ParseComment(text, basePos())
	if len(anns) != 1 {
		t.Fatalf("got %d annotations", len(anns))
	}
	pos := anns[0].Position
	if pos.Line != 3 {
		t.Errorf("line = %d, want 3", pos.Line)
	}
	if pos.Column != 3 {
		t.Errorf("column = %d, want 3", pos.Column)
	}
	if pos.Filename != "test.go" {
		t.Errorf("filename = %q, want test.go", pos.Filename)
	}
}

func TestTrailingCommaTolerated(t *testing.T) {
	ann := parseSingle(t, `@X(a=1, b=2,)`)
	if len(ann.Arguments) != 2 {
		t.Fatalf("args = %v", ann.Arguments)
	}
	arr := parseSingle(t, `@Y(items=["a", "b",])`)
	if a, ok := arr.Arguments["items"].(ArrayValue); !ok || len(a.Elements) != 2 {
		t.Fatalf("items = %v", arr.Arguments["items"])
	}
}

func TestParseErrors(t *testing.T) {
	tests := []struct {
		name string
		text string
		code string
	}{
		{"unterminated string", `@X(a="oops)`, CodeSyntax},
		{"missing close paren", `@X(a=1`, CodeSyntax},
		{"bad value token", `@X(a==)`, CodeSyntax},
		{"duplicate argument", `@X(a=1, a=2)`, CodeDuplicateArgument},
		{"bad separator", `@X(a=1 b=2)`, CodeSyntax},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, diags := ParseComment(tt.text, basePos())
			if len(diags) == 0 {
				t.Fatalf("expected a diagnostic for %q", tt.text)
			}
			found := false
			for _, d := range diags {
				if d.Code == tt.code {
					found = true
				}
			}
			if !found {
				t.Fatalf("expected code %s, got %v", tt.code, diags)
			}
		})
	}
}

func TestParseKeepsGoodSiblingAfterBadOne(t *testing.T) {
	text := "@X(a=)\n@Service"
	anns, diags := ParseComment(text, basePos())
	if len(diags) == 0 {
		t.Fatalf("expected diagnostic for first annotation")
	}
	found := false
	for _, a := range anns {
		if a.Name == "Service" {
			found = true
		}
	}
	if !found {
		t.Fatalf("well-formed @Service should still be returned; got %v", anns)
	}
}

func TestDiagnosticErrorFormat(t *testing.T) {
	_, diags := ParseComment(`@X(a=)`, basePos())
	if len(diags) == 0 {
		t.Fatal("expected a diagnostic")
	}
	msg := diags[0].Error()
	if want := "test.go:1:"; msg[:len(want)] != want {
		t.Fatalf("error format = %q, want prefix %q", msg, want)
	}
}

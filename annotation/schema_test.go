package annotation

import (
	"strings"
	"testing"
)

// validateText parses exactly one annotation and validates it against the
// default registry for the given target, returning the diagnostics.
func validateText(t *testing.T, text string, target Target) []*Diagnostic {
	t.Helper()
	ann := parseSingle(t, text)
	return DefaultRegistry().Validate(ann, target)
}

func hasCode(diags []*Diagnostic, code string) bool {
	for _, d := range diags {
		if d.Code == code {
			return true
		}
	}
	return false
}

func TestValidateWellFormed(t *testing.T) {
	cases := []struct {
		text   string
		target Target
	}{
		{`@Service(name="userService", scope="singleton")`, TargetStruct},
		{`@RestController`, TargetStruct},
		{`@RequestMapping(path="/api/v1/users")`, TargetStruct},
		{`@GetMapping(path="/{id}", timeout="2s")`, TargetMethod},
		{`@Response(status=200, error="user_not_found")`, TargetMethod},
		{`@ExceptionHandler(type="domain.UserNotFoundError")`, TargetMethod},
		{`@ConfigurationProperties(prefix="server")`, TargetStruct},
		{`@Application(name="users-service", scan=["./internal/..."])`, TargetStruct},
		{`@PostConstruct`, TargetMethod},
	}
	for _, c := range cases {
		if diags := validateText(t, c.text, c.target); len(diags) != 0 {
			t.Errorf("%s: unexpected diagnostics: %v", c.text, diags)
		}
	}
}

func TestValidateUnknownAnnotation(t *testing.T) {
	diags := validateText(t, `@Wat(x=1)`, TargetStruct)
	if !hasCode(diags, CodeUnknownAnnotation) {
		t.Fatalf("expected unknown-annotation diagnostic, got %v", diags)
	}
	if diags[0].Severity != SeverityWarning {
		t.Fatalf("unknown annotation should be a warning, got %v", diags[0].Severity)
	}
}

func TestValidateInvalidTarget(t *testing.T) {
	// @GetMapping is a method annotation; applying it to a struct is invalid.
	diags := validateText(t, `@GetMapping(path="/x")`, TargetStruct)
	if !hasCode(diags, CodeInvalidTarget) {
		t.Fatalf("expected invalid-target diagnostic, got %v", diags)
	}
}

func TestValidateUnknownArgument(t *testing.T) {
	diags := validateText(t, `@Service(bogus="x")`, TargetStruct)
	if !hasCode(diags, CodeUnknownArgument) {
		t.Fatalf("expected unknown-argument diagnostic, got %v", diags)
	}
}

func TestValidateMissingRequired(t *testing.T) {
	diags := validateText(t, `@Application(scan=["./..."])`, TargetStruct)
	if !hasCode(diags, CodeMissingArgument) {
		t.Fatalf("expected missing-argument diagnostic for name, got %v", diags)
	}
}

func TestValidateArgumentType(t *testing.T) {
	// path expects a string; supplying an integer is a type error.
	diags := validateText(t, `@RequestMapping(path=123)`, TargetStruct)
	if !hasCode(diags, CodeArgumentType) {
		t.Fatalf("expected argument-type diagnostic, got %v", diags)
	}
}

func TestValidateDurationArgument(t *testing.T) {
	if diags := validateText(t, `@GetMapping(timeout="150ms")`, TargetMethod); len(diags) != 0 {
		t.Fatalf("valid duration rejected: %v", diags)
	}
	diags := validateText(t, `@GetMapping(timeout="not-a-duration")`, TargetMethod)
	if !hasCode(diags, CodeArgumentType) {
		t.Fatalf("expected duration type error, got %v", diags)
	}
}

func TestValidateStringArrayArgument(t *testing.T) {
	// scan must be an array of strings; a mixed array fails.
	diags := validateText(t, `@Application(name="x", scan=["ok", 3])`, TargetStruct)
	if !hasCode(diags, CodeArgumentType) {
		t.Fatalf("expected type error for non-string array element, got %v", diags)
	}
}

func TestValidateAllowedSet(t *testing.T) {
	// scope allows only singleton/prototype.
	diags := validateText(t, `@Service(scope="request")`, TargetStruct)
	if !hasCode(diags, CodeArgumentValue) {
		t.Fatalf("expected allowed-value diagnostic, got %v", diags)
	}
	// Enum-like value may be written unquoted.
	if diags := validateText(t, `@Service(scope=singleton)`, TargetStruct); len(diags) != 0 {
		t.Fatalf("unquoted enum value rejected: %v", diags)
	}
}

func TestValidatePositionalRequired(t *testing.T) {
	if diags := validateText(t, `@ResponseStatus(404)`, TargetMethod); len(diags) != 0 {
		t.Fatalf("valid positional rejected: %v", diags)
	}
	diags := validateText(t, `@ResponseStatus`, TargetMethod)
	if !hasCode(diags, CodeMissingArgument) {
		t.Fatalf("expected missing positional diagnostic, got %v", diags)
	}
}

func TestValidateRepeatable(t *testing.T) {
	text := "@Response(status=200)\n@Response(status=404, error=\"x\")"
	anns, diags := ParseComment(text, basePos())
	if len(diags) != 0 {
		t.Fatalf("parse diagnostics: %v", diags)
	}
	if d := DefaultRegistry().ValidateGroup(anns, TargetMethod); len(d) != 0 {
		t.Fatalf("@Response is repeatable but ValidateGroup complained: %v", d)
	}

	// @Service is not repeatable.
	dupes := []Annotation{{Name: "Service", Position: basePos()}, {Name: "Service", Position: basePos()}}
	d := DefaultRegistry().ValidateGroup(dupes, TargetStruct)
	if !hasCode(d, CodeNotRepeatable) {
		t.Fatalf("expected not-repeatable diagnostic, got %v", d)
	}
}

func TestRegistryRegisterDuplicate(t *testing.T) {
	r := NewRegistry()
	def := &Definition{Name: "Foo", Targets: []Target{TargetStruct}}
	if err := r.Register(def); err != nil {
		t.Fatalf("first register failed: %v", err)
	}
	if err := r.Register(def); err == nil {
		t.Fatal("expected error registering duplicate name")
	}
	if err := r.Register(&Definition{Name: ""}); err == nil {
		t.Fatal("expected error registering empty name")
	}
}

func TestCustomValidator(t *testing.T) {
	r := NewRegistry()
	r.MustRegister(&Definition{
		Name:      "Even",
		Targets:   []Target{TargetMethod},
		Arguments: map[string]ArgumentDefinition{"n": required(ArgInteger)},
		Validator: ValidatorFunc(func(ann Annotation) []*Diagnostic {
			if iv, ok := ann.Arguments["n"].(IntValue); ok && iv.Val%2 != 0 {
				return []*Diagnostic{newError("GOBANN999", ann.Position, "n must be even")}
			}
			return nil
		}),
	})
	ann := parseSingle(t, `@Even(n=3)`)
	diags := r.Validate(ann, TargetMethod)
	if len(diags) != 1 || !strings.Contains(diags[0].Message, "even") {
		t.Fatalf("expected custom validator diagnostic, got %v", diags)
	}
}

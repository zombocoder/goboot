package validate

import (
	"go/token"
	"regexp"

	"github.com/zombocoder/goboot/annotation"
)

// diag builds a diagnostic at a position.
func diag(sev annotation.Severity, code string, pos token.Position, msg string) *annotation.Diagnostic {
	return &annotation.Diagnostic{Severity: sev, Code: code, Message: msg, Position: pos}
}

// typeErr builds a type-mismatch error anchored to an annotation.
func typeErr(a annotation.Annotation, msg string) *annotation.Diagnostic {
	return diag(annotation.SeverityError, codeTypeMismatch, a.Position, msg)
}

// intPositional extracts the single integer positional argument.
func intPositional(a annotation.Annotation) (int64, bool) {
	v, ok := a.Positional()
	if !ok {
		return 0, false
	}
	iv, ok := v.(annotation.IntValue)
	return iv.Val, ok
}

// stringPositional extracts the single string positional argument.
func stringPositional(a annotation.Annotation) (string, bool) {
	v, ok := a.Positional()
	if !ok {
		return "", false
	}
	sv, ok := v.(annotation.StringValue)
	return sv.Val, ok
}

// intArg extracts a named integer argument.
func intArg(a annotation.Annotation, name string) (int64, bool) {
	v, ok := a.Arg(name)
	if !ok {
		return 0, false
	}
	iv, ok := v.(annotation.IntValue)
	return iv.Val, ok
}

// compilePattern verifies a @Pattern regex compiles, returning a diagnostic on
// failure so the generated MustCompile can never panic at runtime.
func compilePattern(a annotation.Annotation, raw string) (*annotation.Diagnostic, bool) {
	if _, err := regexp.Compile(raw); err != nil {
		return diag(annotation.SeverityError, codeBadPattern, a.Position,
			"@Pattern regex does not compile: "+err.Error()), false
	}
	return nil, true
}

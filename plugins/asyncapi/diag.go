package asyncapi

import (
	"fmt"
	"go/token"

	"github.com/zombocoder/goboot/annotation"
)

// diag builds a diagnostic at a position.
func diag(sev annotation.Severity, code string, pos token.Position, format string, args ...any) *annotation.Diagnostic {
	return &annotation.Diagnostic{Severity: sev, Code: code, Message: fmt.Sprintf(format, args...), Position: pos}
}

// stringArg reads a named string argument, or "" when absent.
func stringArg(a annotation.Annotation, name string) string {
	if v, ok := a.Arg(name); ok {
		if s, ok := v.(annotation.StringValue); ok {
			return s.Val
		}
	}
	return ""
}

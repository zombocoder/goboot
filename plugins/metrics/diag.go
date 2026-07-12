package metrics

import (
	"fmt"
	"go/token"

	"github.com/zombocoder/goboot/annotation"
)

// diag builds an error-severity diagnostic at a position.
func diag(code string, pos token.Position, format string, args ...any) *annotation.Diagnostic {
	return &annotation.Diagnostic{
		Severity: annotation.SeverityError,
		Code:     code,
		Message:  fmt.Sprintf(format, args...),
		Position: pos,
	}
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

// stringArrayArg reads a named string-array argument, or nil when absent.
func stringArrayArg(a annotation.Annotation, name string) []string {
	v, ok := a.Arg(name)
	if !ok {
		return nil
	}
	arr, ok := v.(annotation.ArrayValue)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(arr.Elements))
	for _, e := range arr.Elements {
		if s, ok := e.(annotation.StringValue); ok {
			out = append(out, s.Val)
		}
	}
	return out
}

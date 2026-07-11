package annotation

import (
	"fmt"
	"go/token"
)

// Severity classifies a diagnostic. See specification §39.2.
type Severity uint8

const (
	// SeverityInfo is advisory and never fails a build.
	SeverityInfo Severity = iota
	// SeverityWarning may be promoted to an error under strict mode.
	SeverityWarning
	// SeverityError always fails a build.
	SeverityError
)

func (s Severity) String() string {
	switch s {
	case SeverityInfo:
		return "info"
	case SeverityWarning:
		return "warning"
	case SeverityError:
		return "error"
	default:
		return "unknown"
	}
}

// Diagnostic is a source-positioned message emitted while parsing or validating
// annotations. Code is a stable identifier from the GOBANN* family (§39.4).
//
// Diagnostic implements error so parse failures can flow through ordinary Go
// error handling while retaining position and code.
type Diagnostic struct {
	Severity Severity
	Code     string
	Message  string
	Position token.Position
}

// Error renders the diagnostic in the standard compiler format:
//
//	path/file.go:LINE:COL: CODE: message
//
// matching the layout described in §39.3.
func (d *Diagnostic) Error() string {
	pos := d.Position.String()
	if d.Code != "" {
		return fmt.Sprintf("%s: %s: %s", pos, d.Code, d.Message)
	}
	return fmt.Sprintf("%s: %s", pos, d.Message)
}

// newError builds an error-severity diagnostic at pos.
func newError(code string, pos token.Position, format string, args ...any) *Diagnostic {
	return &Diagnostic{
		Severity: SeverityError,
		Code:     code,
		Message:  fmt.Sprintf(format, args...),
		Position: pos,
	}
}

// Annotation diagnostic codes (GOBANN* family, §39.4).
const (
	// CodeSyntax is a malformed annotation: bad tokens, unbalanced delimiters,
	// unterminated strings, or an unexpected argument structure.
	CodeSyntax = "GOBANN001"
	// CodeUnknownAnnotation is an annotation with no registered schema.
	CodeUnknownAnnotation = "GOBANN002"
	// CodeInvalidTarget is an annotation applied to a declaration kind it does
	// not support.
	CodeInvalidTarget = "GOBANN003"
	// CodeUnknownArgument is a named argument not declared by the schema.
	CodeUnknownArgument = "GOBANN004"
	// CodeMissingArgument is a required argument that was not supplied.
	CodeMissingArgument = "GOBANN005"
	// CodeArgumentType is an argument whose value has the wrong type.
	CodeArgumentType = "GOBANN006"
	// CodeArgumentValue is an argument whose value is outside the allowed set.
	CodeArgumentValue = "GOBANN007"
	// CodeDuplicateArgument is the same named argument supplied twice.
	CodeDuplicateArgument = "GOBANN008"
	// CodeNotRepeatable is a non-repeatable annotation applied more than once.
	CodeNotRepeatable = "GOBANN009"
)

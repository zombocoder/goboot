package validate

import (
	"github.com/zombocoder/goboot/annotation"
	"github.com/zombocoder/goboot/model"
)

// Analyze validates the field-constraint annotations against their field types
// (§46.1): @Min/@Max on non-numeric fields, @Size on non-length types,
// @Pattern/@Email on non-strings, malformed @Size bounds, uncompilable @Pattern
// regexes, and constraints on non-request types (which won't be enforced). It
// never mutates the model.
func (*Plugin) Analyze(app *model.Application) []*annotation.Diagnostic {
	_, diags := resolve(app)
	return diags
}

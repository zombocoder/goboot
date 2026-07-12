// Package validate is a goboot plugin that generates request validation from
// field-constraint annotations (github.com/zombocoder/goboot issue #34). It
// registers a small set of Bean-Validation-style field annotations, checks them
// against their field types during analysis, and emits a runtime.Validator that
// the app wires onto goboot's existing HTTP bind→validate step:
//
//	// @Required
//	// @Size(max=200)
//	Title string `json:"title"`
//
// generates a validator that rejects a missing or over-long Title with a 400
// Problem listing every offending field.
//
// Register it in goboot.yaml:
//
//	plugins:
//	  - github.com/zombocoder/goboot/plugins/validate
//
// then wire the generated validator in the composition root:
//
//	httpDeps.Validator = generated.NewValidator()
//
// The plugin drives generation entirely from its own annotations via the deeper
// plugin API (model.Application.Declarations, §46.5) and the request types on
// the analyzed routes; it reads no struct tags of its own.
package validate

import (
	"github.com/zombocoder/goboot/annotation"
	"github.com/zombocoder/goboot/model"
)

// Annotation names this plugin owns.
const (
	annRequired = "Required"
	annMin      = "Min"
	annMax      = "Max"
	annSize     = "Size"
	annPattern  = "Pattern"
	annEmail    = "Email"
)

// outputFile is the generated validator's name; the zz_goboot_ prefix lets
// `goboot clean` remove it (§40).
const outputFile = "zz_goboot_validate.gen.go"

// Plugin implements the AnnotationProvider, Analyzer, and Generator capabilities.
type Plugin struct{}

// New constructs the validate plugin for injection into cli.Main.
func New() *Plugin { return &Plugin{} }

// Name identifies the plugin within a host.
func (*Plugin) Name() string { return "validate" }

// Version is the plugin's own version.
func (*Plugin) Version() string { return "0.1.0" }

// Annotations registers the field-constraint annotations so the compiler
// recognizes them instead of reporting them as unknown.
func (*Plugin) Annotations() []*annotation.Definition {
	field := []annotation.Target{annotation.TargetField}
	intArg := &annotation.ArgumentDefinition{Type: annotation.ArgInteger, Required: true}
	return []*annotation.Definition{
		{Name: annRequired, Targets: field},
		{Name: annEmail, Targets: field},
		{Name: annMin, Targets: field, Positional: intArg},
		{Name: annMax, Targets: field, Positional: intArg},
		{Name: annPattern, Targets: field, Positional: &annotation.ArgumentDefinition{
			Type: annotation.ArgString, Required: true}},
		{Name: annSize, Targets: field, Arguments: map[string]annotation.ArgumentDefinition{
			"min": {Type: annotation.ArgInteger},
			"max": {Type: annotation.ArgInteger},
		}},
	}
}

// annotationNames is the set of annotations this plugin consumes.
var annotationNames = []string{annRequired, annEmail, annMin, annMax, annPattern, annSize}

// hasOurAnnotation reports whether a declaration carries any constraint we own.
func hasOurAnnotation(d model.AnnotatedDecl) bool {
	for _, n := range annotationNames {
		if d.Has(n) {
			return true
		}
	}
	return false
}

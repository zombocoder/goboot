// Package plugin defines goboot's compile-time extension model (§46). Plugins
// are ordinary Go packages linked into a goboot host (the CLI, or a custom one)
// through normal imports — there is no dynamic .so loading (§46.3). A plugin
// contributes any combination of: annotation schemas, extra semantic analysis,
// additional generated files, and SQL dialects (database drivers). This is the
// payoff of the framework's adapter-seam design: a native pgx driver, an
// OpenAPI generator, or a custom annotation ships as a plugin without changing
// the core.
//
// A plugin declares its capabilities by implementing the optional interfaces
// below; the host invokes only the ones a plugin satisfies. Plugins must return
// diagnostics rather than panic and must produce deterministic output (§46.4);
// the host recovers panics defensively but a well-behaved plugin never relies on
// that.
package plugin

import (
	"github.com/zombocoder/goboot/annotation"
	"github.com/zombocoder/goboot/model"
	"github.com/zombocoder/goboot/sqlgen"
)

// Plugin is the base every plugin implements: a stable name and version (§46.1).
type Plugin interface {
	// Name uniquely identifies the plugin within a host.
	Name() string
	// Version is the plugin's own version, surfaced by `goboot version`.
	Version() string
}

// AnnotationProvider contributes annotation schemas that the host registers
// before scanning, so the compiler recognizes the plugin's annotations instead
// of reporting them as unknown.
type AnnotationProvider interface {
	Plugin
	Annotations() []*annotation.Definition
}

// Analyzer inspects the assembled application model and returns diagnostics
// (§46.1). Analyzers run after core analysis and must not mutate the model.
type Analyzer interface {
	Plugin
	Analyze(app *model.Application) []*annotation.Diagnostic
}

// Generator emits additional source files for the application (§46.1), for
// example an OpenAPI description or a metadata manifest. Output must be
// deterministic (§46.4).
type Generator interface {
	Plugin
	Generate(app *model.Application) ([]File, error)
}

// DialectProvider contributes SQL dialects (database drivers) by name, which the
// host makes available to repository generation (§27.4). This is how a native
// pgx or a MySQL driver registers its placeholder style.
type DialectProvider interface {
	Plugin
	Dialects() map[string]sqlgen.Dialect
}

// File is a source file produced by a Generator plugin. Name is a base file
// name (no directory) and should carry the generated-file prefix so `goboot
// clean` removes it.
type File struct {
	Name    string
	Content []byte
}

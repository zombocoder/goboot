package plugin

import (
	"fmt"
	"sort"

	"github.com/zombocoder/goboot/annotation"
	"github.com/zombocoder/goboot/model"
	"github.com/zombocoder/goboot/sqlgen"
)

// Plugin diagnostic codes (GOBPLG family, §39.4).
const (
	// CodeDuplicatePlugin is two plugins registered under the same name.
	CodeDuplicatePlugin = "GOBPLG001"
	// CodeAnnotationConflict is a plugin annotation whose name is already
	// registered by the core catalogue or another plugin.
	CodeAnnotationConflict = "GOBPLG002"
	// CodePluginPanic is a plugin that panicked; the host recovered it into a
	// diagnostic rather than crashing (§46.4).
	CodePluginPanic = "GOBPLG003"
)

// Registry is a host's set of plugins. It merges plugin annotations into the
// annotation registry, runs plugin analyzers and generators with panic
// recovery, and resolves SQL dialects. The zero value is not usable; call New.
type Registry struct {
	plugins []Plugin
	byName  map[string]Plugin
}

// New builds a registry from the given plugins. It panics if two plugins share a
// name, since that is a programming error in host wiring, not user input.
func New(plugins ...Plugin) *Registry {
	r := &Registry{byName: map[string]Plugin{}}
	for _, p := range plugins {
		if err := r.Register(p); err != nil {
			panic(err)
		}
	}
	return r
}

// Register adds a plugin, rejecting a duplicate name.
func (r *Registry) Register(p Plugin) error {
	if p == nil || p.Name() == "" {
		return fmt.Errorf("plugin: cannot register a nil or unnamed plugin")
	}
	if _, exists := r.byName[p.Name()]; exists {
		return fmt.Errorf("plugin: %q is already registered", p.Name())
	}
	r.byName[p.Name()] = p
	r.plugins = append(r.plugins, p)
	return nil
}

// Plugins returns the registered plugins in registration order.
func (r *Registry) Plugins() []Plugin {
	out := make([]Plugin, len(r.plugins))
	copy(out, r.plugins)
	return out
}

// AnnotationRegistry returns an annotation registry containing the v0.1 core
// catalogue plus every plugin's annotations. A plugin annotation whose name
// collides with an existing one yields a diagnostic and is skipped, so a bad
// plugin cannot silently shadow a core annotation.
func (r *Registry) AnnotationRegistry() (*annotation.Registry, []*annotation.Diagnostic) {
	reg := annotation.DefaultRegistry()
	var diags []*annotation.Diagnostic
	for _, p := range r.plugins {
		ap, ok := p.(AnnotationProvider)
		if !ok {
			continue
		}
		for _, def := range ap.Annotations() {
			if def == nil {
				continue
			}
			if err := reg.Register(def); err != nil {
				diags = append(diags, pluginDiag(CodeAnnotationConflict,
					"plugin %q: %v", p.Name(), err))
			}
		}
	}
	return reg, diags
}

// Analyze runs every Analyzer plugin against the application and returns the
// combined diagnostics. A panicking plugin is recovered into a diagnostic
// (§46.4).
func (r *Registry) Analyze(app *model.Application) []*annotation.Diagnostic {
	var diags []*annotation.Diagnostic
	for _, p := range r.plugins {
		a, ok := p.(Analyzer)
		if !ok {
			continue
		}
		d, err := safeAnalyze(a, app)
		if err != nil {
			diags = append(diags, pluginDiag(CodePluginPanic, "plugin %q analyze: %v", p.Name(), err))
			continue
		}
		diags = append(diags, d...)
	}
	return diags
}

// Generate runs every Generator plugin and returns the produced files sorted by
// name for deterministic output. A panicking plugin is recovered into a
// diagnostic and produces no files.
func (r *Registry) Generate(app *model.Application) ([]File, []*annotation.Diagnostic) {
	var (
		files []File
		diags []*annotation.Diagnostic
	)
	for _, p := range r.plugins {
		g, ok := p.(Generator)
		if !ok {
			continue
		}
		fs, err := safeGenerate(g, app)
		if err != nil {
			diags = append(diags, pluginDiag(CodePluginPanic, "plugin %q generate: %v", p.Name(), err))
			continue
		}
		files = append(files, fs...)
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Name < files[j].Name })
	return files, diags
}

// Dialect resolves a SQL dialect by name, consulting plugin DialectProviders
// first and falling back to the built-in dialects (§27.4).
func (r *Registry) Dialect(name string) (sqlgen.Dialect, bool) {
	for _, p := range r.plugins {
		dp, ok := p.(DialectProvider)
		if !ok {
			continue
		}
		if d, ok := dp.Dialects()[name]; ok {
			return d, true
		}
	}
	return sqlgen.DialectByName(name)
}

// safeAnalyze runs a plugin analyzer, converting a panic into an error (§46.4).
func safeAnalyze(a Analyzer, app *model.Application) (diags []*annotation.Diagnostic, err error) {
	defer func() {
		if rec := recover(); rec != nil {
			err = fmt.Errorf("panicked: %v", rec)
		}
	}()
	return a.Analyze(app), nil
}

// safeGenerate runs a plugin generator, converting a panic into an error.
func safeGenerate(g Generator, app *model.Application) (files []File, err error) {
	defer func() {
		if rec := recover(); rec != nil {
			err = fmt.Errorf("panicked: %v", rec)
		}
	}()
	return g.Generate(app)
}

// pluginDiag builds an error-severity plugin diagnostic.
func pluginDiag(code, format string, args ...any) *annotation.Diagnostic {
	return &annotation.Diagnostic{
		Severity: annotation.SeverityError,
		Code:     code,
		Message:  fmt.Sprintf(format, args...),
	}
}

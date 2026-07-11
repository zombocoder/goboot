package plugin

import (
	"testing"

	"github.com/zombocoder/goboot/annotation"
	"github.com/zombocoder/goboot/model"
	"github.com/zombocoder/goboot/sqlgen"
)

// basePlugin implements the minimal Plugin interface.
type basePlugin struct{ name string }

func (p basePlugin) Name() string    { return p.name }
func (p basePlugin) Version() string { return "1.0.0" }

func TestRegisterAndPlugins(t *testing.T) {
	r := New(basePlugin{name: "a"}, basePlugin{name: "b"})
	if len(r.Plugins()) != 2 {
		t.Fatalf("plugins = %d, want 2", len(r.Plugins()))
	}
	if err := r.Register(basePlugin{name: "a"}); err == nil {
		t.Error("duplicate plugin name should be rejected")
	}
	if err := r.Register(basePlugin{name: ""}); err == nil {
		t.Error("empty plugin name should be rejected")
	}
}

func TestNewPanicsOnDuplicate(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("New should panic on duplicate plugin names")
		}
	}()
	New(basePlugin{name: "x"}, basePlugin{name: "x"})
}

// annPlugin contributes an annotation.
type annPlugin struct {
	basePlugin
	def *annotation.Definition
}

func (p annPlugin) Annotations() []*annotation.Definition {
	return []*annotation.Definition{p.def}
}

func TestAnnotationRegistryMerges(t *testing.T) {
	r := New(annPlugin{
		basePlugin: basePlugin{name: "ann"},
		def:        &annotation.Definition{Name: "Custom", Targets: []annotation.Target{annotation.TargetStruct}},
	})
	reg, diags := r.AnnotationRegistry()
	if len(diags) != 0 {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if _, ok := reg.Lookup("Custom"); !ok {
		t.Error("plugin annotation should be registered")
	}
	// A core annotation is still present.
	if _, ok := reg.Lookup("Service"); !ok {
		t.Error("core annotations should remain")
	}
}

func TestAnnotationConflictReported(t *testing.T) {
	// Registering an annotation named like a core one conflicts.
	r := New(annPlugin{
		basePlugin: basePlugin{name: "bad"},
		def:        &annotation.Definition{Name: "Service", Targets: []annotation.Target{annotation.TargetStruct}},
	})
	_, diags := r.AnnotationRegistry()
	if len(diags) == 0 || diags[0].Code != CodeAnnotationConflict {
		t.Fatalf("expected an annotation-conflict diagnostic, got %v", diags)
	}
}

// analyzerPlugin returns a fixed diagnostic.
type analyzerPlugin struct {
	basePlugin
	panics bool
}

func (p analyzerPlugin) Analyze(*model.Application) []*annotation.Diagnostic {
	if p.panics {
		panic("boom")
	}
	return []*annotation.Diagnostic{{Severity: annotation.SeverityWarning, Code: "PLG", Message: "hi"}}
}

func TestAnalyzeRunsPlugins(t *testing.T) {
	r := New(analyzerPlugin{basePlugin: basePlugin{name: "an"}})
	diags := r.Analyze(&model.Application{})
	if len(diags) != 1 || diags[0].Code != "PLG" {
		t.Fatalf("analyzer diagnostics = %v", diags)
	}
}

func TestAnalyzeRecoversPanic(t *testing.T) {
	r := New(analyzerPlugin{basePlugin: basePlugin{name: "boom"}, panics: true})
	diags := r.Analyze(&model.Application{})
	if len(diags) != 1 || diags[0].Code != CodePluginPanic {
		t.Fatalf("expected a recovered-panic diagnostic, got %v", diags)
	}
}

// genPlugin emits files.
type genPlugin struct {
	basePlugin
	files  []File
	panics bool
}

func (p genPlugin) Generate(*model.Application) ([]File, error) {
	if p.panics {
		panic("kaboom")
	}
	return p.files, nil
}

func TestGenerateSortsAndRuns(t *testing.T) {
	r := New(genPlugin{
		basePlugin: basePlugin{name: "gen"},
		files: []File{
			{Name: "zz_b.txt", Content: []byte("b")},
			{Name: "zz_a.txt", Content: []byte("a")},
		},
	})
	files, diags := r.Generate(&model.Application{})
	if len(diags) != 0 {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if len(files) != 2 || files[0].Name != "zz_a.txt" {
		t.Errorf("files not sorted: %v", files)
	}
}

func TestGenerateRecoversPanic(t *testing.T) {
	r := New(genPlugin{basePlugin: basePlugin{name: "boom"}, panics: true})
	_, diags := r.Generate(&model.Application{})
	if len(diags) != 1 || diags[0].Code != CodePluginPanic {
		t.Fatalf("expected a recovered-panic diagnostic, got %v", diags)
	}
}

// dialectPlugin contributes a dialect.
type dialectPlugin struct{ basePlugin }

func (dialectPlugin) Dialects() map[string]sqlgen.Dialect {
	return map[string]sqlgen.Dialect{"custom": sqlgen.Question}
}

func TestDialectResolution(t *testing.T) {
	r := New(dialectPlugin{basePlugin: basePlugin{name: "d"}})
	// Plugin dialect.
	if d, ok := r.Dialect("custom"); !ok || d.Name() != "question" {
		t.Errorf("plugin dialect not resolved: %v %v", d, ok)
	}
	// Built-in fallback.
	if d, ok := r.Dialect("postgres"); !ok || d.Name() != "postgres" {
		t.Errorf("built-in dialect fallback failed: %v %v", d, ok)
	}
	// Unknown.
	if _, ok := r.Dialect("nope"); ok {
		t.Error("unknown dialect should not resolve")
	}
}

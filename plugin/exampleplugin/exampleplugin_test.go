package exampleplugin

import (
	"strings"
	"testing"

	"github.com/zombocoder/goboot/annotation"
	"github.com/zombocoder/goboot/compiler"
	"github.com/zombocoder/goboot/plugin"
	"github.com/zombocoder/goboot/sqlgen"
)

// host builds a plugin registry containing the example plugin.
func host() *plugin.Registry { return plugin.New(New()) }

// TestPluginAnnotationRecognized proves that, with the plugin registered, the
// compiler recognizes @Exposed instead of warning about an unknown annotation.
func TestPluginAnnotationRecognized(t *testing.T) {
	reg, diags := host().AnnotationRegistry()
	if len(diags) != 0 {
		t.Fatalf("registry diagnostics: %v", diags)
	}

	loader := &compiler.Loader{Registry: reg}
	scan, err := loader.Load("./testdata/app")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	for _, d := range scan.Diagnostics {
		if d.Code == annotation.CodeUnknownAnnotation && strings.Contains(d.Message, "Exposed") {
			t.Fatalf("@Exposed should be recognized once the plugin is registered: %s", d.Error())
		}
	}
}

// TestPluginAnalyzerAndGenerator drives the full host flow: analyze the app and
// generate the manifest artifact.
func TestPluginAnalyzerAndGenerator(t *testing.T) {
	h := host()
	reg, _ := h.AnnotationRegistry()
	loader := &compiler.Loader{Registry: reg}
	scan, err := loader.Load("./testdata/app")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	res := compiler.Analyze(scan)

	// Plugin analyzer contributes its diagnostic.
	pluginDiags := h.Analyze(res.App)
	found := false
	for _, d := range pluginDiags {
		if d.Code == "EXPL001" && strings.Contains(d.Message, "plugin-demo") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected the example analyzer diagnostic, got %v", pluginDiags)
	}

	// Plugin generator produces the manifest listing the discovered component.
	files, gdiags := h.Generate(res.App)
	if len(gdiags) != 0 {
		t.Fatalf("generate diagnostics: %v", gdiags)
	}
	if len(files) != 1 || files[0].Name != "zz_goboot_manifest.txt" {
		t.Fatalf("generated files = %v", files)
	}
	content := string(files[0].Content)
	if !strings.Contains(content, "application: plugin-demo") {
		t.Errorf("manifest missing application name:\n%s", content)
	}
	if !strings.Contains(content, ":Svc") {
		t.Errorf("manifest missing the Svc component:\n%s", content)
	}
	// The generator drives output from its own @Exposed annotation via the
	// model's surfaced declarations (§46.5).
	if !strings.Contains(content, "exposed:\n- Svc.Do") {
		t.Errorf("manifest should list the @Exposed method Svc.Do:\n%s", content)
	}
}

// TestPluginDialectDrivesSQLCompilation proves a plugin-provided dialect (a
// database driver's placeholder style) flows into SQL generation.
func TestPluginDialectDrivesSQLCompilation(t *testing.T) {
	d, ok := host().Dialect("sqlserver")
	if !ok {
		t.Fatal("sqlserver dialect should resolve from the plugin")
	}
	compiled := sqlgen.Compile("SELECT id FROM users WHERE id = :id AND org = :org", d)
	if compiled.SQL != "SELECT id FROM users WHERE id = @p1 AND org = @p2" {
		t.Errorf("sqlserver placeholders wrong: %q", compiled.SQL)
	}
}

// TestExamplePluginImplementsAllCapabilities is a compile-time assertion that
// the example plugin satisfies every optional interface.
func TestExamplePluginImplementsAllCapabilities(t *testing.T) {
	var _ plugin.Plugin = (*Plugin)(nil)
	var _ plugin.AnnotationProvider = (*Plugin)(nil)
	var _ plugin.Analyzer = (*Plugin)(nil)
	var _ plugin.Generator = (*Plugin)(nil)
	var _ plugin.DialectProvider = (*Plugin)(nil)
}

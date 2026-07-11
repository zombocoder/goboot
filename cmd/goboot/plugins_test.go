package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zombocoder/goboot/plugin"
	"github.com/zombocoder/goboot/plugin/exampleplugin"
)

// withExamplePlugin installs the example plugin into the CLI host for the
// duration of a test, restoring the default afterward.
func withExamplePlugin(t *testing.T) {
	t.Helper()
	prev := builtinPlugins
	builtinPlugins = func() []plugin.Plugin { return []plugin.Plugin{exampleplugin.New()} }
	t.Cleanup(func() { builtinPlugins = prev })
}

func TestVersionListsPlugins(t *testing.T) {
	withExamplePlugin(t)
	code, out, _ := runCLI("version")
	if code != 0 {
		t.Fatalf("version exit = %d", code)
	}
	if !strings.Contains(out, "example 0.1.0") {
		t.Errorf("version should list the example plugin: %q", out)
	}
}

// The example plugin's testdata app uses @Exposed and needs the plugin's
// annotation registered; validate must succeed with the plugin installed.
func TestValidateWithPluginAnnotation(t *testing.T) {
	withExamplePlugin(t)
	dir := filepath.Join("..", "..", "plugin", "exampleplugin")
	code, out, errOut := runCLI("validate", "-dir", dir, "./testdata/app")
	if code != 0 {
		t.Fatalf("validate exit = %d\nstdout=%s\nstderr=%s", code, out, errOut)
	}
	// The unknown-annotation warning for @Exposed must not appear.
	if strings.Contains(errOut, "Exposed") {
		t.Errorf("@Exposed should be recognized via the plugin, stderr=%s", errOut)
	}
}

func TestGenerateWritesPluginArtifact(t *testing.T) {
	withExamplePlugin(t)

	root, _ := filepath.Abs(filepath.Join("..", ".."))
	tmp, err := os.MkdirTemp(root, "genplugin")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmp)
	outRel, _ := filepath.Rel(root, tmp)

	code, _, errOut := runCLI("generate",
		"-dir", root, "-output", outRel, "-package", "wiring",
		"./plugin/exampleplugin/testdata/app")
	if code != 0 {
		t.Fatalf("generate exit = %d, stderr=%s", code, errOut)
	}

	// The plugin's manifest artifact was written alongside the wiring.
	manifest := filepath.Join(tmp, "zz_goboot_manifest.txt")
	data, err := os.ReadFile(manifest)
	if err != nil {
		t.Fatalf("plugin manifest not written: %v", err)
	}
	if !strings.Contains(string(data), "application: plugin-demo") {
		t.Errorf("manifest content = %q", data)
	}
}

func TestGenerateWithPluginDialect(t *testing.T) {
	withExamplePlugin(t)

	root, _ := filepath.Abs(filepath.Join("..", ".."))
	tmp, err := os.MkdirTemp(root, "gendialect")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmp)
	outRel, _ := filepath.Rel(root, tmp)

	// The plugin-provided "sqlserver" dialect must be accepted and applied.
	code, _, errOut := runCLI("generate",
		"-dir", root, "-output", outRel, "-package", "wiring", "-dialect", "sqlserver",
		"./compiler/testdata/repoapp")
	if code != 0 {
		t.Fatalf("generate exit = %d, stderr=%s", code, errOut)
	}
	data, err := os.ReadFile(filepath.Join(tmp, generatedFileName))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "WHERE id = @p1") {
		t.Errorf("plugin dialect not applied; generated SQL should use @p1 placeholders")
	}
}

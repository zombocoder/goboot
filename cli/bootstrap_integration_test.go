package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestSelfBootstrapEndToEnd proves the full plugin-loading loop: the stock
// binary reads goboot.yaml, builds a plugin-aware CLI, re-execs it, and the
// plugin becomes active (its @Exposed annotation is recognized and its Generator
// writes an artifact). It uses the in-repo example plugin via a replace
// directive so no network fetch is needed.
func TestSelfBootstrapEndToEnd(t *testing.T) {
	if testing.Short() {
		t.Skip("builds a plugin-aware CLI; skipped under -short")
	}
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go toolchain not available")
	}

	root, err := filepath.Abs("..")
	if err != nil {
		t.Fatal(err)
	}
	proj := t.TempDir()

	write(t, filepath.Join(proj, "go.mod"),
		"module bootstraptest\n\ngo 1.25\n\nrequire github.com/zombocoder/goboot v0.0.0\n\nreplace github.com/zombocoder/goboot => "+root+"\n")
	// Reuse the root go.sum so the -mod=mod build resolves offline from cache.
	if sum, err := os.ReadFile(filepath.Join(root, "go.sum")); err == nil {
		write(t, filepath.Join(proj, "go.sum"), string(sum))
	}
	write(t, filepath.Join(proj, "goboot.yaml"),
		"application:\n  name: smoke\nplugins:\n  - module: github.com/zombocoder/goboot\n    import: github.com/zombocoder/goboot/plugin/exampleplugin\n    new: New\n")
	write(t, filepath.Join(proj, "app", "app.go"), exposedApp)

	// Build the stock (plugin-free) binary.
	goboot := filepath.Join(proj, "goboot"+exeSuffix())
	build := exec.Command("go", "build", "-o", goboot, "./cmd/goboot")
	build.Dir = root
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("building stock binary: %v\n%s", err, out)
	}

	gen := func(extraEnv ...string) (string, error) {
		cmd := exec.Command(goboot, "generate", "-dir", proj,
			"-output", "internal/generated", "-package", "generated", "./app")
		cmd.Dir = proj
		cmd.Env = append(os.Environ(), extraEnv...)
		out, err := cmd.CombinedOutput()
		return string(out), err
	}
	manifest := filepath.Join(proj, "internal", "generated", "zz_goboot_manifest.txt")

	t.Run("bootstraps and activates the plugin", func(t *testing.T) {
		out, err := gen()
		if err != nil {
			t.Fatalf("generate: %v\n%s", err, out)
		}
		if !strings.Contains(out, "building plugin-aware CLI") {
			t.Errorf("expected a bootstrap build message, got:\n%s", out)
		}
		// The plugin's Generator wrote its artifact — proof it ran via bootstrap.
		if _, err := os.Stat(manifest); err != nil {
			t.Fatalf("plugin manifest not written (plugin did not run):\n%s", out)
		}
		// The plugin-aware binary was cached for reuse.
		bins, _ := filepath.Glob(filepath.Join(proj, ".goboot", "bin", "goboot-*"))
		if len(bins) == 0 {
			t.Error("no cached plugin-aware binary found")
		}
	})

	t.Run("second run reuses the cached binary", func(t *testing.T) {
		out, err := gen()
		if err != nil {
			t.Fatalf("generate: %v\n%s", err, out)
		}
		if strings.Contains(out, "building plugin-aware CLI") {
			t.Errorf("second run should reuse the cache, not rebuild:\n%s", out)
		}
	})

	t.Run("GOBOOT_BOOTSTRAP=off skips the plugin", func(t *testing.T) {
		os.Remove(manifest)
		out, err := gen("GOBOOT_BOOTSTRAP=off")
		if err != nil {
			t.Fatalf("generate: %v\n%s", err, out)
		}
		// Without the plugin, @Exposed is an unknown annotation and the plugin's
		// manifest is not written.
		if !strings.Contains(out, "Exposed") {
			t.Errorf("expected an unknown-annotation warning for @Exposed, got:\n%s", out)
		}
		if _, err := os.Stat(manifest); err == nil {
			t.Error("manifest should not be written when the plugin is disabled")
		}
	})
}

func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

const exposedApp = `// Package app is a bootstrap fixture using the example plugin's @Exposed.
package app

import "context"

// @Application(name="plugin-demo")
type Application struct{}

// @Service(name="svc")
type Svc struct{}

func NewSvc() *Svc { return &Svc{} }

// @Exposed
func (s *Svc) Do(ctx context.Context) error { return nil }
`

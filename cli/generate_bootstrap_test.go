package cli

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// TestGenerateToleratesUngeneratedOutput reproduces the fresh-project bootstrap
// case: the composition root imports the wiring package that does not exist yet
// (the output directory is empty). Generation must still succeed — otherwise a
// new project can never generate without first commenting out its own main.go.
func TestGenerateToleratesUngeneratedOutput(t *testing.T) {
	if testing.Short() {
		t.Skip("loads packages via the go tool; skipped under -short")
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
		"module fresh\n\ngo 1.25\n\nrequire github.com/zombocoder/goboot v0.0.0\n\nreplace github.com/zombocoder/goboot => "+root+"\n")
	if sum, err := os.ReadFile(filepath.Join(root, "go.sum")); err == nil {
		write(t, filepath.Join(proj, "go.sum"), string(sum))
	}
	write(t, filepath.Join(proj, "goboot.yaml"),
		"application:\n  name: fresh\n  packages:\n    - ./...\ngeneration:\n  output: internal/generated\n  package: generated\n  clean: true\n")
	write(t, filepath.Join(proj, "app", "app.go"),
		"package app\n\n// @Application(name=\"fresh\")\ntype Application struct{}\n\n// @Component(name=\"greeter\")\ntype Greeter struct{}\n\n// NewGreeter builds it.\nfunc NewGreeter() *Greeter { return &Greeter{} }\n")
	// The composition root imports the not-yet-generated wiring package.
	write(t, filepath.Join(proj, "main.go"),
		"package main\n\nimport \"fresh/internal/generated\"\n\nfunc main() { _, _ = generated.NewApplication() }\n")
	// An empty output directory — exactly the state that produced the
	// "invalid package name" load error.
	if err := os.MkdirAll(filepath.Join(proj, "internal", "generated"), 0o755); err != nil {
		t.Fatal(err)
	}

	// The temp module resolves goboot via a local replace; -mod=mod lets the go
	// tool complete its module graph from cache, and GOWORK=off keeps the repo
	// workspace (which does not include this temp dir) out of the way.
	t.Setenv("GOWORK", "off")
	t.Setenv("GOFLAGS", "-mod=mod")

	var stdout, stderr bytes.Buffer
	if code := cmdGenerate([]string{"-dir", proj}, &stdout, &stderr); code != 0 {
		t.Fatalf("generate failed (code %d) on a fresh project:\nstdout: %s\nstderr: %s",
			code, stdout.String(), stderr.String())
	}
	if _, err := os.Stat(filepath.Join(proj, "internal", "generated", generatedFileName)); err != nil {
		t.Fatalf("wiring file not written: %v\nstderr: %s", err, stderr.String())
	}
}

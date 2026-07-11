package di

import (
	"flag"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zombocoder/goboot/compiler"
)

var update = flag.Bool("update", false, "update golden files")

// analyzeDiapp loads and analyzes the multi-package example under the compiler
// package's testdata.
func analyzeDiapp(t *testing.T) *compiler.AnalysisResult {
	t.Helper()
	l := &compiler.Loader{Dir: filepath.Join("..", "..", "compiler")}
	scan, err := l.Load("./testdata/diapp/...")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	res := compiler.Analyze(scan)
	for _, d := range res.Diagnostics {
		if d.Severity == 2 { // SeverityError
			t.Fatalf("analysis error: %s", d.Error())
		}
	}
	return res
}

func generateDiapp(t *testing.T) string {
	t.Helper()
	res := analyzeDiapp(t)
	src, err := Generate(res.App, res.Graph, Options{Package: "wiring"})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	return src
}

func TestGenerateWiringGolden(t *testing.T) {
	src := generateDiapp(t)
	golden := filepath.Join("testdata", "golden", "diapp_wiring.gen.go")

	if *update {
		if err := os.MkdirAll(filepath.Dir(golden), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(golden, []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
		t.Logf("wrote golden %s", golden)
		return
	}

	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatalf("reading golden (run with -update to create): %v", err)
	}
	if src != string(want) {
		t.Errorf("generated output differs from golden.\n--- got ---\n%s", src)
	}
}

func TestGenerateWiringContent(t *testing.T) {
	src := generateDiapp(t)

	// Marker and package clause.
	if !strings.HasPrefix(src, GeneratedMarker) {
		t.Errorf("missing generated marker")
	}
	if !strings.Contains(src, "package wiring") {
		t.Errorf("missing package clause")
	}
	// Every component constructor should be called.
	for _, want := range []string{
		"repo.NewPostgresUserRepository()",
		"config.ProvideIDGenerator()",
		"service.NewUserService(",
		"controller.NewUserController(",
		"func buildComponents() (*Components, error)",
	} {
		if !strings.Contains(src, want) {
			t.Errorf("generated output missing %q", want)
		}
	}
	// The repository must be constructed before the service that consumes it.
	if idx(src, "NewPostgresUserRepository") > idx(src, "NewUserService") {
		t.Errorf("repository should be constructed before service")
	}
}

func TestGenerateDeterministic(t *testing.T) {
	first := generateDiapp(t)
	for i := 0; i < 5; i++ {
		if got := generateDiapp(t); got != first {
			t.Fatalf("generation is not deterministic")
		}
	}
}

// TestGeneratedWiringCompiles writes the generated file into a temporary
// package inside the module and compiles it, satisfying the Milestone 3
// acceptance criterion that a multi-package example compiles (§48.3).
func TestGeneratedWiringCompiles(t *testing.T) {
	src := generateDiapp(t)

	moduleRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	dir, err := os.MkdirTemp(moduleRoot, "genwire")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	if err := os.WriteFile(filepath.Join(dir, "wiring.gen.go"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("go", "build", ".")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("generated wiring did not compile: %v\n%s\n--- source ---\n%s", err, out, src)
	}
}

func TestGenerateRejectsCycle(t *testing.T) {
	l := &compiler.Loader{Dir: filepath.Join("..", "..", "compiler")}
	scan, err := l.Load("./testdata/cycle")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	res := compiler.Analyze(scan)
	if _, err := Generate(res.App, res.Graph, Options{Package: "wiring"}); err == nil {
		t.Fatal("expected an error generating wiring for a cyclic graph")
	}
}

func idx(s, sub string) int { return strings.Index(s, sub) }

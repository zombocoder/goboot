package validate

import (
	"flag"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/zombocoder/goboot/compiler"
	"github.com/zombocoder/goboot/model"
	"github.com/zombocoder/goboot/plugin"
)

var update = flag.Bool("update", false, "update golden files")

// analyzeFixture loads a testdata package with this plugin's annotations
// registered and returns the assembled application model.
func analyzeFixture(t *testing.T, pkg string) *model.Application {
	t.Helper()
	reg, diags := plugin.New(New()).AnnotationRegistry()
	for _, d := range diags {
		t.Fatalf("registry: %s", d.Error())
	}
	loader := &compiler.Loader{Dir: ".", Registry: reg}
	scan, err := loader.Load("./testdata/" + pkg)
	if err != nil {
		t.Fatalf("load %s: %v", pkg, err)
	}
	res := compiler.Analyze(scan)
	for _, d := range res.Diagnostics {
		if d.Severity == 2 { // SeverityError
			t.Fatalf("core analysis error: %s", d.Error())
		}
	}
	res.App.Package = "generated"
	return res.App
}

func TestAnnotationsRegistered(t *testing.T) {
	got := map[string]bool{}
	for _, d := range New().Annotations() {
		got[d.Name] = true
	}
	for _, want := range annotationNames {
		if !got[want] {
			t.Errorf("annotation %s not registered", want)
		}
	}
}

func TestGenerateGolden(t *testing.T) {
	app := analyzeFixture(t, "api")
	files, err := New().Generate(app)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if len(files) != 1 || files[0].Name != outputFile {
		t.Fatalf("unexpected files: %+v", files)
	}
	goldenPath := filepath.Join("testdata", "golden", outputFile)
	if *update {
		if err := os.WriteFile(goldenPath, files[0].Content, 0o644); err != nil {
			t.Fatalf("write golden: %v", err)
		}
	}
	want, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden (run -update): %v", err)
	}
	if string(files[0].Content) != string(want) {
		t.Errorf("generated output differs from golden.\n--- got ---\n%s", files[0].Content)
	}
}

// The generated source must be syntactically valid Go (the generator gofmt's it,
// so a parse failure means a genuine emission bug).
func TestGeneratedParses(t *testing.T) {
	app := analyzeFixture(t, "api")
	files, err := New().Generate(app)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if _, err := parser.ParseFile(token.NewFileSet(), outputFile, files[0].Content, parser.AllErrors); err != nil {
		t.Fatalf("generated source does not parse: %v\n%s", err, files[0].Content)
	}
}

// Determinism: generating twice yields byte-identical output.
func TestGenerateDeterministic(t *testing.T) {
	app := analyzeFixture(t, "api")
	a, _ := New().Generate(app)
	b, _ := New().Generate(app)
	if string(a[0].Content) != string(b[0].Content) {
		t.Error("generation is not deterministic")
	}
}

func TestAnalyzeValidFixtureIsClean(t *testing.T) {
	app := analyzeFixture(t, "api")
	if diags := New().Analyze(app); len(diags) != 0 {
		for _, d := range diags {
			t.Errorf("unexpected diagnostic: %s", d.Error())
		}
	}
}

func TestAnalyzeReportsMisapplied(t *testing.T) {
	app := analyzeFixture(t, "bad")
	codes := map[string]int{}
	for _, d := range New().Analyze(app) {
		codes[d.Code]++
	}
	for _, want := range []string{
		codeTypeMismatch, // @Min on string, @Pattern on int
		codeBadSize,      // @Size min > max
		codeBadPattern,   // @Pattern("[")
		codeUnenforced,   // Detached.Note is not a request type
	} {
		if codes[want] == 0 {
			t.Errorf("expected at least one %s diagnostic; got %v", want, codes)
		}
	}
}

// The generated validator must actually compile against the fixture's request
// types and the runtime — the real correctness gate for a generator (§48). It is
// built in a temp package under this module so go resolves the imports.
func TestGeneratedCompiles(t *testing.T) {
	app := analyzeFixture(t, "api")
	files, err := New().Generate(app)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	dir, err := os.MkdirTemp(".", "compilecheck")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	if err := os.WriteFile(filepath.Join(dir, "gen.go"), files[0].Content, 0o644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("go", "build", "./"+dir)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("generated code does not compile: %v\n%s", err, out)
	}
}

// A fixture with no constraints generates nothing.
func TestGenerateNoConstraintsEmitsNothing(t *testing.T) {
	app := &model.Application{Name: "empty", Package: "generated"}
	files, err := New().Generate(app)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if len(files) != 0 {
		t.Errorf("expected no files, got %d", len(files))
	}
}

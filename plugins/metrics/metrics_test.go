package metrics

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
	for _, want := range []string{annCounter, annGauge} {
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
	golden := filepath.Join("testdata", "golden", outputFile)
	if *update {
		if err := os.WriteFile(golden, files[0].Content, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatalf("read golden (run -update): %v", err)
	}
	if string(files[0].Content) != string(want) {
		t.Errorf("generated output differs from golden.\n--- got ---\n%s", files[0].Content)
	}
}

func TestGeneratedParses(t *testing.T) {
	app := analyzeFixture(t, "api")
	files, _ := New().Generate(app)
	if _, err := parser.ParseFile(token.NewFileSet(), outputFile, files[0].Content, parser.AllErrors); err != nil {
		t.Fatalf("generated source does not parse: %v\n%s", err, files[0].Content)
	}
}

// The generated collectors must compile against the Prometheus client.
func TestGeneratedCompiles(t *testing.T) {
	app := analyzeFixture(t, "api")
	files, _ := New().Generate(app)
	dir, err := os.MkdirTemp(".", "compilecheck")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	if err := os.WriteFile(filepath.Join(dir, "gen.go"), files[0].Content, 0o644); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command("go", "build", "./"+dir).CombinedOutput(); err != nil {
		t.Fatalf("generated code does not compile: %v\n%s", err, out)
	}
}

func TestGenerateDeterministic(t *testing.T) {
	app := analyzeFixture(t, "api")
	a, _ := New().Generate(app)
	b, _ := New().Generate(app)
	if string(a[0].Content) != string(b[0].Content) {
		t.Error("generation is not deterministic")
	}
}

func TestAnalyzeValidIsClean(t *testing.T) {
	if diags := New().Analyze(analyzeFixture(t, "api")); len(diags) != 0 {
		for _, d := range diags {
			t.Errorf("unexpected diagnostic: %s", d.Error())
		}
	}
}

func TestAnalyzeReportsBad(t *testing.T) {
	codes := map[string]int{}
	for _, d := range New().Analyze(analyzeFixture(t, "bad")) {
		codes[d.Code]++
	}
	for _, want := range []string{codeInvalidName, codeInvalidLabel, codeDuplicate} {
		if codes[want] == 0 {
			t.Errorf("expected a %s diagnostic; got %v", want, codes)
		}
	}
}

func TestGenerateNoMetricsEmitsNothing(t *testing.T) {
	files, err := New().Generate(&model.Application{Name: "empty", Package: "generated"})
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 0 {
		t.Errorf("expected no files, got %d", len(files))
	}
}

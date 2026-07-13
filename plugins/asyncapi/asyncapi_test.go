package asyncapi

import (
	"encoding/json"
	"flag"
	"os"
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
	return res.App
}

func TestAnnotationsRegistered(t *testing.T) {
	got := map[string]bool{}
	for _, d := range New().Annotations() {
		got[d.Name] = true
	}
	for _, want := range []string{annListener, annPublisher} {
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

// The emitted document must be a valid AsyncAPI 3.0 JSON object with channels
// and operations.
func TestGeneratedIsValidJSON(t *testing.T) {
	app := analyzeFixture(t, "api")
	files, _ := New().Generate(app)
	var doc map[string]any
	if err := json.Unmarshal(files[0].Content, &doc); err != nil {
		t.Fatalf("generated document is not valid JSON: %v\n%s", err, files[0].Content)
	}
	if doc["asyncapi"] != "3.0.0" {
		t.Errorf("asyncapi version = %v, want 3.0.0", doc["asyncapi"])
	}
	channels, ok := doc["channels"].(map[string]any)
	if !ok || len(channels) == 0 {
		t.Errorf("expected non-empty channels, got %v", doc["channels"])
	}
	ops, ok := doc["operations"].(map[string]any)
	if !ok || len(ops) == 0 {
		t.Errorf("expected non-empty operations, got %v", doc["operations"])
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

func TestAnalyzeReportsMissingPayload(t *testing.T) {
	codes := map[string]int{}
	for _, d := range New().Analyze(analyzeFixture(t, "bad")) {
		codes[d.Code]++
	}
	if codes[codeNoPayload] == 0 {
		t.Errorf("expected a %s diagnostic; got %v", codeNoPayload, codes)
	}
}

func TestGenerateNoHandlersEmitsNothing(t *testing.T) {
	files, err := New().Generate(&model.Application{Name: "empty"})
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 0 {
		t.Errorf("expected no files, got %d", len(files))
	}
}

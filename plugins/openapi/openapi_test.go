package openapi

import (
	"encoding/json"
	"testing"

	"github.com/zombocoder/goboot/compiler"
)

// generateSpec loads the fixture API through the compiler and runs the plugin,
// returning the parsed OpenAPI document.
func generateSpec(t *testing.T) map[string]any {
	t.Helper()
	loader := &compiler.Loader{Dir: "."}
	scan, err := loader.Load("./testdata/api")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	res := compiler.Analyze(scan)
	for _, d := range res.Diagnostics {
		if d.Severity == 2 {
			t.Fatalf("analysis error: %s", d.Error())
		}
	}
	files, err := New().Generate(res.App)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if len(files) != 1 || files[0].Name != "zz_goboot_openapi.json" {
		t.Fatalf("unexpected files: %+v", files)
	}
	var doc map[string]any
	if err := json.Unmarshal(files[0].Content, &doc); err != nil {
		t.Fatalf("spec is not valid JSON: %v\n%s", err, files[0].Content)
	}
	return doc
}

func TestDocumentShape(t *testing.T) {
	doc := generateSpec(t)
	if doc["openapi"] != "3.0.3" {
		t.Errorf("openapi version = %v", doc["openapi"])
	}
	if title := doc["info"].(map[string]any)["title"]; title != "widget-api" {
		t.Errorf("title = %v, want widget-api", title)
	}
}

func TestPathsAndParameters(t *testing.T) {
	doc := generateSpec(t)
	paths := doc["paths"].(map[string]any)

	get, ok := paths["/widgets/{id}"].(map[string]any)
	if !ok {
		t.Fatalf("missing GET path; paths = %v", keys(paths))
	}
	op := get["get"].(map[string]any)
	if op["operationId"] != "GetWidget" {
		t.Errorf("operationId = %v", op["operationId"])
	}
	params := op["parameters"].([]any)
	if len(params) != 2 {
		t.Fatalf("expected 2 parameters, got %d", len(params))
	}
	// Sorted by location: path before query.
	p0 := params[0].(map[string]any)
	if p0["name"] != "id" || p0["in"] != "path" || p0["required"] != true {
		t.Errorf("first param = %v, want required path 'id'", p0)
	}
	p1 := params[1].(map[string]any)
	if p1["name"] != "expand" || p1["in"] != "query" {
		t.Errorf("second param = %v, want query 'expand'", p1)
	}
	// The success response references the Widget schema.
	resp := op["responses"].(map[string]any)["200"].(map[string]any)
	schema := resp["content"].(map[string]any)["application/json"].(map[string]any)["schema"].(map[string]any)
	if schema["$ref"] != "#/components/schemas/Widget" {
		t.Errorf("response schema = %v", schema)
	}
}

func TestRequestBody(t *testing.T) {
	doc := generateSpec(t)
	post := doc["paths"].(map[string]any)["/widgets"].(map[string]any)["post"].(map[string]any)
	body, ok := post["requestBody"].(map[string]any)
	if !ok {
		t.Fatal("POST should have a request body")
	}
	schema := body["content"].(map[string]any)["application/json"].(map[string]any)["schema"].(map[string]any)
	props := schema["properties"].(map[string]any)
	if _, ok := props["name"]; !ok {
		t.Errorf("body should include the 'name' json field, got %v", keys(props))
	}
	if _, ok := props["quantity"]; !ok {
		t.Errorf("body should include the 'quantity' json field, got %v", keys(props))
	}
}

func TestSchemaTypes(t *testing.T) {
	doc := generateSpec(t)
	widget := doc["components"].(map[string]any)["schemas"].(map[string]any)["Widget"].(map[string]any)
	props := widget["properties"].(map[string]any)

	cases := map[string]map[string]any{
		"quantity":  {"type": "integer", "format": "int64"},
		"price":     {"type": "number", "format": "double"},
		"active":    {"type": "boolean"},
		"createdAt": {"type": "string", "format": "date-time"},
	}
	for field, want := range cases {
		got := props[field].(map[string]any)
		for k, v := range want {
			if got[k] != v {
				t.Errorf("%s.%s = %v, want %v", field, k, got[k], v)
			}
		}
	}
	// tags is a string array.
	tags := props["tags"].(map[string]any)
	if tags["type"] != "array" || tags["items"].(map[string]any)["type"] != "string" {
		t.Errorf("tags schema = %v", tags)
	}
}

func TestDeterministic(t *testing.T) {
	loader := &compiler.Loader{Dir: "."}
	scan, err := loader.Load("./testdata/api")
	if err != nil {
		t.Fatal(err)
	}
	res := compiler.Analyze(scan)
	first, _ := New().Generate(res.App)
	for i := 0; i < 3; i++ {
		again, _ := New().Generate(res.App)
		if string(again[0].Content) != string(first[0].Content) {
			t.Fatal("openapi output is not deterministic")
		}
	}
}

func keys(m map[string]any) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

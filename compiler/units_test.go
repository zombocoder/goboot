package compiler

import (
	"go/token"
	"testing"

	"github.com/zombocoder/goboot/annotation"
)

func TestSplitPos(t *testing.T) {
	cases := []struct {
		in     string
		line   int
		col    int
		file   string
		wantOK bool
	}{
		{"/a/b.go:12:5", 12, 5, "/a/b.go", true},
		{`C:\x\b.go:3:1`, 3, 1, `C:\x\b.go`, true},
		{"nocolons", 0, 0, "", false},
		{"file.go:notnum", 0, 0, "", false},
		{"file.go:10:notnum", 0, 0, "", false},
	}
	for _, c := range cases {
		line, col, file, ok := splitPos(c.in)
		if ok != c.wantOK || line != c.line || col != c.col || file != c.file {
			t.Errorf("splitPos(%q) = (%d,%d,%q,%v), want (%d,%d,%q,%v)",
				c.in, line, col, file, ok, c.line, c.col, c.file, c.wantOK)
		}
	}
}

func TestParsePackagesPos(t *testing.T) {
	if p := parsePackagesPos(""); p != (token.Position{}) {
		t.Errorf("empty pos = %v, want zero", p)
	}
	p := parsePackagesPos("/x/y.go:7:2")
	if p.Filename != "/x/y.go" || p.Line != 7 || p.Column != 2 {
		t.Errorf("parsePackagesPos = %+v", p)
	}
	// A string without the numeric suffix falls back to a bare filename.
	if p := parsePackagesPos("weird"); p.Filename != "weird" {
		t.Errorf("fallback filename = %q, want weird", p.Filename)
	}
}

func TestRemapPositionOutOfRange(t *testing.T) {
	table := []lineInfo{{filename: "a.go", srcLine: 5, srcColumn: 3, srcOffset: 40}}
	// Line within range: remaps.
	got := remapPosition(token.Position{Line: 1, Column: 2}, table)
	if got.Line != 5 || got.Column != 4 || got.Filename != "a.go" || got.Offset != 41 {
		t.Errorf("remap = %+v", got)
	}
	// Out-of-range line: returned unchanged.
	orig := token.Position{Line: 9, Column: 1}
	if remapPosition(orig, table) != orig {
		t.Errorf("out-of-range remap should be identity")
	}
}

func TestDeclarationHelpers(t *testing.T) {
	d := &Declaration{Annotations: []annotation.Annotation{
		{Name: "Service"},
		{Name: "Response"},
		{Name: "Response"},
	}}
	if !d.Has("Service") || d.Has("Missing") {
		t.Errorf("Has mismatch")
	}
	if _, ok := d.Find("Response"); !ok {
		t.Errorf("Find(Response) should succeed")
	}
	if _, ok := d.Find("Missing"); ok {
		t.Errorf("Find(Missing) should fail")
	}
	if got := len(d.FindAll("Response")); got != 2 {
		t.Errorf("FindAll(Response) = %d, want 2", got)
	}
}

func TestHasErrors(t *testing.T) {
	res := &ScanResult{Diagnostics: []*annotation.Diagnostic{
		{Severity: annotation.SeverityWarning},
	}}
	if res.HasErrors() {
		t.Errorf("warnings should not count as errors")
	}
	res.Diagnostics = append(res.Diagnostics, &annotation.Diagnostic{Severity: annotation.SeverityError})
	if !res.HasErrors() {
		t.Errorf("expected HasErrors true")
	}
}

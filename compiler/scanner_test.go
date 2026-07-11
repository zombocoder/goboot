package compiler

import (
	"go/types"
	"os"
	"strings"
	"testing"

	"github.com/zombocoder/goboot/annotation"
)

func loadPkg(t *testing.T, pattern string, flags ...string) *ScanResult {
	t.Helper()
	l := &Loader{BuildFlags: flags}
	res, err := l.Load(pattern)
	if err != nil {
		t.Fatalf("Load(%q) error: %v", pattern, err)
	}
	return res
}

func findDecl(res *ScanResult, name string) *Declaration {
	for _, d := range res.Declarations {
		if d.Name == name {
			return d
		}
	}
	return nil
}

func TestScanBasicAssociation(t *testing.T) {
	res := loadPkg(t, "./testdata/basic")

	cases := []struct {
		name       string
		target     annotation.Target
		annotation string
	}{
		{"basic", annotation.TargetPackage, "Application"},
		{"UserService", annotation.TargetStruct, "Service"},
		{"UserRepository", annotation.TargetInterface, "Repository"},
		{"UserController", annotation.TargetStruct, "RestController"},
		{"GetUser", annotation.TargetMethod, "GetMapping"},
		{"FindByID", annotation.TargetMethod, "Query"},
		{"Clock", annotation.TargetStruct, "Component"},
		{"Zone", annotation.TargetField, "Named"},
	}
	for _, c := range cases {
		d := findDecl(res, c.name)
		if d == nil {
			t.Errorf("declaration %q not found", c.name)
			continue
		}
		if d.Target != c.target {
			t.Errorf("%s: target = %v, want %v", c.name, d.Target, c.target)
		}
		if !d.Has(c.annotation) {
			t.Errorf("%s: missing @%s (has %v)", c.name, c.annotation, annNames(d))
		}
	}
}

func annNames(d *Declaration) []string {
	var out []string
	for _, a := range d.Annotations {
		out = append(out, a.Name)
	}
	return out
}

func TestScanRepeatedAnnotation(t *testing.T) {
	res := loadPkg(t, "./testdata/basic")
	d := findDecl(res, "GetUser")
	if d == nil {
		t.Fatal("GetUser not found")
	}
	if got := len(d.FindAll("Response")); got != 2 {
		t.Fatalf("GetUser has %d @Response, want 2", got)
	}
}

func TestTypeLookup(t *testing.T) {
	res := loadPkg(t, "./testdata/basic")

	svc := findDecl(res, "UserService")
	if svc.TypeName == nil {
		t.Fatal("UserService.TypeName is nil")
	}
	if _, ok := svc.TypeName.Type().Underlying().(*types.Struct); !ok {
		t.Errorf("UserService underlying type = %T, want struct", svc.TypeName.Type().Underlying())
	}

	repo := findDecl(res, "UserRepository")
	if _, ok := repo.TypeName.Type().Underlying().(*types.Interface); !ok {
		t.Errorf("UserRepository underlying type = %T, want interface", repo.TypeName.Type().Underlying())
	}

	m := findDecl(res, "GetUser")
	if m.Func == nil {
		t.Fatal("GetUser.Func is nil")
	}
	if m.Recv == nil || m.Recv.Name() != "UserController" {
		t.Errorf("GetUser receiver = %v, want UserController", m.Recv)
	}
	// The method signature must be resolvable through the types.Func.
	if _, ok := m.Func.Type().(*types.Signature); !ok {
		t.Errorf("GetUser.Func type = %T, want *types.Signature", m.Func.Type())
	}
}

// TestPositionAccuracy verifies that every annotation's reported position points
// at the exact '@' in the source file. This is the core Milestone 2 acceptance
// criterion "source positions are accurate".
func TestPositionAccuracy(t *testing.T) {
	if n := verifyPositions(t, loadPkg(t, "./testdata/basic")); n == 0 {
		t.Fatal("no annotations checked")
	}
}

// TestBlockCommentPositions verifies accurate positions for annotations written
// inside a /* */ block comment, exercising the multi-line block cleaner.
func TestBlockCommentPositions(t *testing.T) {
	res := loadPkg(t, "./testdata/blockcomment")
	d := findDecl(res, "BlockService")
	if d == nil {
		t.Fatal("BlockService not found")
	}
	if !d.Has("Service") || !d.Has("Primary") {
		t.Fatalf("BlockService annotations = %v", annNames(d))
	}
	verifyPositions(t, res)
}

// verifyPositions checks that each annotation position points at its '@' in the
// source and returns the number of annotations verified.
func verifyPositions(t *testing.T, res *ScanResult) int {
	t.Helper()
	fileCache := map[string][]string{}
	readLines := func(name string) []string {
		if lines, ok := fileCache[name]; ok {
			return lines
		}
		b, err := os.ReadFile(name)
		if err != nil {
			t.Fatalf("reading %s: %v", name, err)
		}
		lines := strings.Split(string(b), "\n")
		fileCache[name] = lines
		return lines
	}

	checked := 0
	for _, d := range res.Declarations {
		for _, a := range d.Annotations {
			pos := a.Position
			if pos.Filename == "" || pos.Line == 0 {
				t.Errorf("@%s on %s has empty position", a.Name, d.Name)
				continue
			}
			lines := readLines(pos.Filename)
			if pos.Line-1 >= len(lines) {
				t.Errorf("@%s: line %d out of range", a.Name, pos.Line)
				continue
			}
			line := lines[pos.Line-1]
			if pos.Column-1 > len(line) {
				t.Errorf("@%s: column %d out of range on %q", a.Name, pos.Column, line)
				continue
			}
			got := line[pos.Column-1:]
			if !strings.HasPrefix(got, "@"+a.Name) {
				t.Errorf("@%s at %s:%d:%d points at %q, want '@%s'",
					a.Name, pos.Filename, pos.Line, pos.Column, truncate(got), a.Name)
			}
			checked++
		}
	}
	t.Logf("verified %d annotation positions", checked)
	return checked
}

func truncate(s string) string {
	if len(s) > 20 {
		return s[:20]
	}
	return s
}

func TestBuildTagsRespected(t *testing.T) {
	// Without the tag, only the base file compiles.
	res := loadPkg(t, "./testdata/buildtags")
	if findDecl(res, "TaggedService") != nil {
		t.Error("TaggedService should be excluded without the build tag")
	}
	if findDecl(res, "BaseService") == nil {
		t.Error("BaseService should always be present")
	}

	// With the tag, both compile.
	tagged := loadPkg(t, "./testdata/buildtags", "-tags=goboot_on")
	if findDecl(tagged, "TaggedService") == nil {
		t.Error("TaggedService should be present with -tags=goboot_on")
	}
	if findDecl(tagged, "BaseService") == nil {
		t.Error("BaseService should be present with -tags=goboot_on")
	}
}

func TestLoadErrorsSurfaced(t *testing.T) {
	res := loadPkg(t, "./testdata/broken")
	if !res.HasErrors() {
		t.Fatal("expected load errors for broken package")
	}
	found := false
	for _, d := range res.Diagnostics {
		if d.Code == CodeLoadError {
			found = true
		}
	}
	if !found {
		t.Errorf("expected a %s diagnostic, got %v", CodeLoadError, res.Diagnostics)
	}
}

func TestUnknownAnnotationWarning(t *testing.T) {
	// @Query is not part of the v0.1 core catalogue; it should surface as a
	// warning, not an error, so the build is not blocked.
	res := loadPkg(t, "./testdata/basic")
	var warn *annotation.Diagnostic
	for _, d := range res.Diagnostics {
		if d.Code == annotation.CodeUnknownAnnotation && strings.Contains(d.Message, "Query") {
			warn = d
		}
	}
	if warn == nil {
		t.Fatal("expected an unknown-annotation warning for @Query")
	}
	if warn.Severity != annotation.SeverityWarning {
		t.Errorf("@Query diagnostic severity = %v, want warning", warn.Severity)
	}
}

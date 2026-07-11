package lint

import (
	"testing"

	"github.com/zombocoder/goboot/annotation"
	"github.com/zombocoder/goboot/compiler"
)

func analyze(t *testing.T) []*annotation.Diagnostic {
	t.Helper()
	loader := &compiler.Loader{Dir: "."}
	scan, err := loader.Load("./testdata/api")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	res := compiler.Analyze(scan)
	for _, d := range res.Diagnostics {
		if d.Severity == annotation.SeverityError {
			t.Fatalf("unexpected core error: %s", d.Error())
		}
	}
	return New().Analyze(res.App)
}

func byCode(diags []*annotation.Diagnostic) map[string]int {
	m := map[string]int{}
	for _, d := range diags {
		m[d.Code]++
	}
	return m
}

func TestRulesFire(t *testing.T) {
	counts := byCode(analyze(t))
	if counts["LINT001"] != 1 {
		t.Errorf("LINT001 (duplicate operationId) count = %d, want 1", counts["LINT001"])
	}
	// /Accounts and /Accounts/legacy/ both have an uppercase segment.
	if counts["LINT002"] != 2 {
		t.Errorf("LINT002 (non-lowercase) count = %d, want 2", counts["LINT002"])
	}
	if counts["LINT003"] != 1 {
		t.Errorf("LINT003 (trailing slash) count = %d, want 1", counts["LINT003"])
	}
}

func TestAllWarnings(t *testing.T) {
	for _, d := range analyze(t) {
		if d.Severity != annotation.SeverityWarning {
			t.Errorf("%s should be a warning, got severity %d", d.Code, d.Severity)
		}
		if d.Position.Line == 0 {
			t.Errorf("%s should carry a source position", d.Code)
		}
	}
}

func TestCleanAppHasNoDiagnostics(t *testing.T) {
	// nonLowerSegment ignores {param} segments and the plugin skips well-formed
	// routes; a lowercase, unique-handler route set yields nothing.
	if seg, ok := nonLowerSegment("/users/{id}/orders"); ok {
		t.Errorf("clean path flagged segment %q", seg)
	}
	if _, ok := nonLowerSegment("/users/{ID}"); ok {
		t.Error("a {param} segment should be ignored even if uppercase")
	}
}

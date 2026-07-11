package compiler

import "testing"

func TestExceptionHandlerDiscovery(t *testing.T) {
	res := analyzeApp(t, "./testdata/adviceapp")
	if errs := errorDiags(res.Diagnostics); len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	advice := componentByName(res.App, "Advice")
	if advice == nil {
		t.Fatal("advice component not found")
	}
	if len(advice.ExceptionHandlers) != 3 {
		t.Fatalf("expected 3 exception handlers, got %d", len(advice.ExceptionHandlers))
	}

	byName := map[string]int{}
	for i, h := range advice.ExceptionHandlers {
		byName[h.MethodName] = i
	}

	nf := advice.ExceptionHandlers[byName["HandleNotFound"]]
	if nf.CatchAll || nf.ResponseType == nil || nf.SuccessStatus != 404 {
		t.Errorf("HandleNotFound = %+v, want response form, status 404, not catch-all", nf)
	}

	cf := advice.ExceptionHandlers[byName["HandleConflict"]]
	if cf.CatchAll || cf.ResponseType != nil {
		t.Errorf("HandleConflict = %+v, want transform form (no response type)", cf)
	}

	any := advice.ExceptionHandlers[byName["HandleAny"]]
	if !any.CatchAll {
		t.Errorf("HandleAny should be a catch-all (err param), got %+v", any)
	}
}

func TestExceptionHandlerOrphanDiagnostic(t *testing.T) {
	res := analyzeApp(t, "./testdata/orphanadvice")
	found := false
	for _, d := range res.Diagnostics {
		if d.Code == CodeOrphanExceptionHandler {
			found = true
		}
	}
	if !found {
		t.Errorf("expected %s for @ExceptionHandler on a non-advice type", CodeOrphanExceptionHandler)
	}
}

package compiler

import "testing"

func TestObservabilityDiscovery(t *testing.T) {
	res := analyzeApp(t, "./testdata/obsapp")
	if errs := errorDiags(res.Diagnostics); len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	vault := componentByName(res.App, "vault")
	if vault == nil || !vault.Proxied {
		t.Fatal("vault service should be proxied")
	}
	byName := map[string]int{}
	for i, m := range vault.Intercepted {
		byName[m.Name] = i
	}

	store := vault.Intercepted[byName["Store"]]
	if !store.Logged || store.LogLevel != "debug" {
		t.Errorf("Store logged = %v level = %q, want true/debug", store.Logged, store.LogLevel)
	}
	if store.Audit == nil || store.Audit.Action != "store" || store.Audit.Resource != "secret" {
		t.Errorf("Store audit = %+v", store.Audit)
	}

	rotate := vault.Intercepted[byName["Rotate"]]
	if !rotate.Logged || rotate.LogLevel != "info" {
		t.Errorf("Rotate logged = %v level = %q, want true/info (default)", rotate.Logged, rotate.LogLevel)
	}
	if rotate.Audit != nil {
		t.Errorf("Rotate should have no audit, got %+v", rotate.Audit)
	}
}

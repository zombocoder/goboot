package compiler

import "testing"

func TestAnnotatedDeclarationsCollected(t *testing.T) {
	res := analyzeApp(t, "./testdata/plugindecls")

	// A plugin-style annotation (unknown to the core registry, only a warning)
	// is still surfaced so a plugin Generator/Analyzer can act on it.
	exposed := res.App.DeclarationsWith("Exposed")
	if len(exposed) != 1 {
		t.Fatalf("DeclarationsWith(Exposed) = %d, want 1", len(exposed))
	}
	d := exposed[0]
	if d.Name != "Do" || d.Receiver != "Svc" {
		t.Errorf("exposed decl = %s.%s, want Svc.Do", d.Receiver, d.Name)
	}
	if ann, ok := d.Find("Exposed"); !ok || ann.Name != "Exposed" {
		t.Errorf("Find(Exposed) = %+v, %v", ann, ok)
	}

	// Core annotations are surfaced too.
	if len(res.App.DeclarationsWith("Service")) != 1 {
		t.Errorf("expected one @Service declaration")
	}
	if len(res.App.DeclarationsWith("Application")) != 1 {
		t.Errorf("expected one @Application declaration")
	}

	// Declarations without annotations (e.g. NewSvc) are omitted.
	for _, decl := range res.App.Declarations {
		if decl.Name == "NewSvc" {
			t.Errorf("un-annotated NewSvc should not be surfaced")
		}
	}
}

func TestAnnotatedDeclarationsDeterministic(t *testing.T) {
	first := analyzeApp(t, "./testdata/plugindecls").App.Declarations
	for i := 0; i < 3; i++ {
		got := analyzeApp(t, "./testdata/plugindecls").App.Declarations
		if len(got) != len(first) {
			t.Fatalf("declaration count varies: %d vs %d", len(got), len(first))
		}
		for j := range got {
			if got[j].Name != first[j].Name || got[j].Package != first[j].Package {
				t.Fatalf("declaration order is not deterministic at %d", j)
			}
		}
	}
}

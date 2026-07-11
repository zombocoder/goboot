package compiler

import (
	"testing"

	"github.com/zombocoder/goboot/model"
)

func TestConstructorVariants(t *testing.T) {
	res := analyzeApp(t, "./testdata/ctorvariants")
	for _, d := range res.Diagnostics {
		if d.Code == CodeInvalidConstructor || d.Code == CodeMissingConstructor {
			t.Fatalf("unexpected constructor diagnostic: %s", d.Message)
		}
	}

	proto := componentByName(res.App, "Proto")
	if proto == nil {
		t.Fatal("Proto not discovered")
	}
	if proto.Scope != model.ScopePrototype {
		t.Errorf("Proto scope = %v, want prototype", proto.Scope)
	}
	if !proto.Constructor.ReturnsError {
		t.Errorf("Proto constructor should report ReturnsError")
	}

	clock := componentByName(res.App, "Clock")
	if clock == nil || clock.Constructor == nil || !clock.Constructor.Constructorless {
		t.Errorf("Clock should be a constructorless component: %+v", clock)
	}

	thing := componentByName(res.App, "ProvideThing")
	if thing == nil || thing.Kind != model.ComponentNut {
		t.Errorf("ProvideThing should be a nut component: %+v", thing)
	}
	if !thing.Constructor.ReturnsError {
		t.Errorf("ProvideThing nut should report ReturnsError")
	}
}

func TestInvalidSecondReturn(t *testing.T) {
	res := analyzeApp(t, "./testdata/badreturn")
	found := false
	for _, d := range res.Diagnostics {
		if d.Code == CodeInvalidConstructor {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected invalid-constructor diagnostic for non-error second return, got %v", res.Diagnostics)
	}
}

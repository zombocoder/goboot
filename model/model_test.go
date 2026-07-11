package model

import "testing"

func TestComponentIDConstructors(t *testing.T) {
	if got := NewComponentID("pkg/x", "Foo"); got != "pkg/x:Foo" {
		t.Errorf("NewComponentID = %q", got)
	}
	if got := NewBeanID("pkg/x", "Provide", "myBean"); got != "pkg/x:Provide#myBean" {
		t.Errorf("NewBeanID = %q", got)
	}
	if got := NewBeanID("pkg/x", "Provide", ""); got != "pkg/x:Provide" {
		t.Errorf("NewBeanID (no name) = %q", got)
	}
}

func TestScopeAndKindStrings(t *testing.T) {
	if ScopeSingleton.String() != "singleton" || ScopePrototype.String() != "prototype" {
		t.Errorf("scope strings mismatch")
	}
	if Scope(9).String() != "unknown" {
		t.Errorf("unknown scope")
	}
	kinds := map[ComponentKind]string{
		ComponentGeneric: "component", ComponentService: "service",
		ComponentRepository: "repository", ComponentController: "controller",
		ComponentConfiguration: "configuration", ComponentBean: "bean",
		ComponentAdvice: "advice", ComponentKind(99): "unknown",
	}
	for k, want := range kinds {
		if k.String() != want {
			t.Errorf("kind %d = %q, want %q", k, k.String(), want)
		}
	}
}

func TestSortComponentsAndLookup(t *testing.T) {
	app := &Application{Components: []*Component{
		{ID: "p:C"}, {ID: "p:A"}, {ID: "p:B"},
	}}
	app.SortComponents()
	if app.Components[0].ID != "p:A" || app.Components[2].ID != "p:C" {
		t.Errorf("components not sorted: %v", app.Components)
	}
	if app.ComponentByID("p:B") == nil {
		t.Errorf("ComponentByID(p:B) should find it")
	}
	if app.ComponentByID("p:missing") != nil {
		t.Errorf("ComponentByID(missing) should be nil")
	}
}

func TestDependsOn(t *testing.T) {
	c := &Component{Dependencies: []Dependency{
		{ResolvedTo: "p:A"},
		{ResolvedTo: ""}, // unresolved: skipped
		{ResolvedTo: "p:B"},
	}}
	got := c.DependsOn()
	if len(got) != 2 || got[0] != "p:A" || got[1] != "p:B" {
		t.Errorf("DependsOn = %v, want [p:A p:B]", got)
	}
}

func TestConstructorQualified(t *testing.T) {
	c := &Constructor{PackagePath: "pkg/x", FuncName: "NewFoo"}
	if c.Qualified() != "pkg/x.NewFoo" {
		t.Errorf("Qualified = %q", c.Qualified())
	}
}

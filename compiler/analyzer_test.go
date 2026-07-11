package compiler

import (
	"strings"
	"testing"

	"github.com/zombocoder/goboot/annotation"
	"github.com/zombocoder/goboot/model"
)

func analyzeApp(t *testing.T, pattern string) *AnalysisResult {
	t.Helper()
	res := loadPkg(t, pattern)
	return Analyze(res)
}

func componentByName(app *model.Application, name string) *model.Component {
	for _, c := range app.Components {
		if c.Name == name {
			return c
		}
	}
	return nil
}

func errorDiags(diags []*annotation.Diagnostic) []*annotation.Diagnostic {
	var out []*annotation.Diagnostic
	for _, d := range diags {
		if d.Severity == annotation.SeverityError {
			out = append(out, d)
		}
	}
	return out
}

func TestAnalyzeDiscoversComponents(t *testing.T) {
	res := analyzeApp(t, "./testdata/diapp/...")
	if errs := errorDiags(res.Diagnostics); len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if res.App.Name != "di-example" {
		t.Errorf("app name = %q, want di-example", res.App.Name)
	}

	want := map[string]model.ComponentKind{
		"userService":    model.ComponentService,
		"userRepository": model.ComponentRepository,
		"UserController": model.ComponentController,
		"Config":         model.ComponentConfiguration,
		"idGenerator":    model.ComponentBean,
	}
	for name, kind := range want {
		c := componentByName(res.App, name)
		if c == nil {
			t.Errorf("component %q not discovered", name)
			continue
		}
		if c.Kind != kind {
			t.Errorf("%s: kind = %v, want %v", name, c.Kind, kind)
		}
	}
}

func TestAnalyzeResolvesInterfaceDependencies(t *testing.T) {
	res := analyzeApp(t, "./testdata/diapp/...")

	svc := componentByName(res.App, "userService")
	if svc == nil {
		t.Fatal("userService not found")
	}
	if len(svc.Dependencies) != 2 {
		t.Fatalf("userService has %d deps, want 2", len(svc.Dependencies))
	}
	// Both dependencies must resolve.
	for _, d := range svc.Dependencies {
		if d.ResolvedTo == "" {
			t.Errorf("dependency %s (%s) did not resolve", d.Name, typeString(d.Type))
		}
	}
	// The repository dependency must resolve to the concrete repository.
	repo := componentByName(res.App, "userRepository")
	ctrl := componentByName(res.App, "UserController")
	resolved := map[model.ComponentID]bool{}
	for _, d := range svc.Dependencies {
		resolved[d.ResolvedTo] = true
	}
	if !resolved[repo.ID] {
		t.Errorf("userService should depend on %s; deps=%v", repo.ID, svc.DependsOn())
	}
	// The controller's UserUseCase dependency resolves to the service.
	if len(ctrl.Dependencies) != 1 || ctrl.Dependencies[0].ResolvedTo != svc.ID {
		t.Errorf("controller should depend on %s, got %v", svc.ID, ctrl.DependsOn())
	}
}

func TestAnalyzeConstructionOrder(t *testing.T) {
	res := analyzeApp(t, "./testdata/diapp/...")
	order, cyc := res.Graph.ConstructionOrder()
	if cyc != nil {
		t.Fatalf("unexpected cycle: %v", cyc.Path)
	}
	pos := map[model.ComponentID]int{}
	for i, id := range order {
		pos[id] = i
	}
	repo := componentByName(res.App, "userRepository")
	svc := componentByName(res.App, "userService")
	ctrl := componentByName(res.App, "UserController")
	if pos[repo.ID] >= pos[svc.ID] {
		t.Errorf("repository must precede service")
	}
	if pos[svc.ID] >= pos[ctrl.ID] {
		t.Errorf("service must precede controller")
	}
}

func TestAnalyzeMissingDependency(t *testing.T) {
	res := analyzeApp(t, "./testdata/missingdep")
	found := false
	for _, d := range res.Diagnostics {
		if d.Code == CodeMissingDependency {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected a missing-dependency diagnostic, got %v", res.Diagnostics)
	}
}

func TestAnalyzeAmbiguousDependency(t *testing.T) {
	res := analyzeApp(t, "./testdata/ambiguous")
	found := false
	for _, d := range res.Diagnostics {
		if d.Code == CodeAmbiguousDependency {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected an ambiguous-dependency diagnostic, got %v", res.Diagnostics)
	}
}

func TestAnalyzePrimaryResolvesAmbiguity(t *testing.T) {
	res := analyzeApp(t, "./testdata/primary")
	for _, d := range res.Diagnostics {
		if d.Code == CodeAmbiguousDependency {
			t.Fatalf("@Primary should resolve ambiguity, but got: %s", d.Message)
		}
	}
	consumer := componentByName(res.App, "consumer")
	if consumer == nil || len(consumer.Dependencies) != 1 {
		t.Fatalf("consumer/deps not as expected: %+v", consumer)
	}
	if consumer.Dependencies[0].ResolvedTo == "" {
		t.Errorf("consumer dependency should resolve to the primary component")
	}
}

func TestAnalyzeCycleDetected(t *testing.T) {
	res := analyzeApp(t, "./testdata/cycle")
	var cycleDiag *annotation.Diagnostic
	for _, d := range res.Diagnostics {
		if d.Code == CodeDependencyCycle {
			cycleDiag = d
		}
	}
	if cycleDiag == nil {
		t.Fatalf("expected a cycle diagnostic, got %v", res.Diagnostics)
	}
	if !strings.Contains(cycleDiag.Message, "cycle detected") {
		t.Errorf("cycle message = %q", cycleDiag.Message)
	}
}

func TestAnalyzeInvalidConstructor(t *testing.T) {
	res := analyzeApp(t, "./testdata/badctor")
	found := false
	for _, d := range res.Diagnostics {
		if d.Code == CodeInvalidConstructor {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected an invalid-constructor diagnostic, got %v", res.Diagnostics)
	}
}

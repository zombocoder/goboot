package compiler

import (
	"testing"

	"github.com/zombocoder/goboot/model"
)

func repoMethod(comp *model.Component, name string) *model.RepositoryMethod {
	if comp == nil || comp.Repository == nil {
		return nil
	}
	for i := range comp.Repository.Methods {
		if comp.Repository.Methods[i].Name == name {
			return &comp.Repository.Methods[i]
		}
	}
	return nil
}

func TestRepositoryDiscovery(t *testing.T) {
	res := analyzeApp(t, "./testdata/repoapp")
	if errs := errorDiags(res.Diagnostics); len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}

	repo := componentByName(res.App, "UserRepository")
	if repo == nil {
		t.Fatal("UserRepository component not discovered")
	}
	if repo.Kind != model.ComponentRepository || repo.Repository == nil {
		t.Fatalf("repo kind/info = %v / %v", repo.Kind, repo.Repository)
	}
	if repo.Constructor == nil || !repo.Constructor.RepositoryImpl {
		t.Error("repository should have a repository-impl constructor")
	}
	if len(repo.Repository.Methods) != 5 {
		t.Fatalf("methods = %d, want 5", len(repo.Repository.Methods))
	}
}

func TestRepositoryReturnShapes(t *testing.T) {
	res := analyzeApp(t, "./testdata/repoapp")
	repo := componentByName(res.App, "UserRepository")

	tests := []struct {
		method       string
		kind         model.QueryKind
		multi        bool
		pointer      bool
		scalar       bool
		rowsAffected bool
	}{
		{"FindByID", model.QueryRead, false, true, false, false},
		{"FindAll", model.QueryRead, true, true, false, false},
		{"Count", model.QueryRead, false, false, true, false},
		{"Create", model.QueryExec, false, false, false, false},
		{"Delete", model.QueryExec, false, false, false, true},
	}
	for _, tt := range tests {
		m := repoMethod(repo, tt.method)
		if m == nil {
			t.Errorf("%s not found", tt.method)
			continue
		}
		if m.Kind != tt.kind {
			t.Errorf("%s kind = %v, want %v", tt.method, m.Kind, tt.kind)
		}
		if m.Return.Multi != tt.multi || m.Return.Pointer != tt.pointer ||
			m.Return.Scalar != tt.scalar || m.Return.RowsAffected != tt.rowsAffected {
			t.Errorf("%s shape = %+v, want multi=%v ptr=%v scalar=%v rows=%v",
				tt.method, m.Return, tt.multi, tt.pointer, tt.scalar, tt.rowsAffected)
		}
	}
}

func TestRepositoryInjectedIntoService(t *testing.T) {
	res := analyzeApp(t, "./testdata/repoapp")
	repo := componentByName(res.App, "UserRepository")
	svc := componentByName(res.App, "userService")
	if svc == nil || repo == nil {
		t.Fatal("service or repo missing")
	}
	if len(svc.Dependencies) != 1 || svc.Dependencies[0].ResolvedTo != repo.ID {
		t.Errorf("service should depend on the repository %q, got %v", repo.ID, svc.DependsOn())
	}
}

func TestRepositoryBadParamRejected(t *testing.T) {
	res := analyzeApp(t, "./testdata/repobadparam")
	found := false
	for _, d := range res.Diagnostics {
		if d.Code == CodeUnknownQueryParam {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected unknown-query-param diagnostic, got %v", res.Diagnostics)
	}
}

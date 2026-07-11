package compiler

import (
	"testing"

	"github.com/zombocoder/goboot/model"
)

func TestBatchAndCallDiscovery(t *testing.T) {
	res := analyzeApp(t, "./testdata/batchapp")
	if errs := errorDiags(res.Diagnostics); len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	repo := componentByName(res.App, "UserRepository")
	if repo == nil || repo.Repository == nil {
		t.Fatal("UserRepository not discovered as a generated repository")
	}
	byName := map[string]model.RepositoryMethod{}
	for _, m := range repo.Repository.Methods {
		byName[m.Name] = m
	}

	insertAll := byName["InsertAll"]
	if insertAll.Kind != model.QueryBatch {
		t.Errorf("InsertAll kind = %v, want batch", insertAll.Kind)
	}
	if insertAll.Batch == nil || insertAll.Batch.ParamName != "users" {
		t.Errorf("InsertAll batch = %+v, want slice param 'users'", insertAll.Batch)
	}
	if !insertAll.Return.RowsAffected {
		t.Error("InsertAll should return rows-affected")
	}

	touchAll := byName["TouchAll"]
	if touchAll.Kind != model.QueryBatch || touchAll.Batch == nil || touchAll.Batch.ParamName != "ids" {
		t.Errorf("TouchAll batch = %+v (kind %v), want slice param 'ids'", touchAll.Batch, touchAll.Kind)
	}

	// @Call with no value result resolves to exec.
	if reindex := byName["Reindex"]; reindex.Kind != model.QueryExec {
		t.Errorf("Reindex (@Call, error-only) kind = %v, want exec", reindex.Kind)
	}
	// @Call returning a slice resolves to a read.
	top := byName["TopByScore"]
	if top.Kind != model.QueryRead || !top.Return.Multi {
		t.Errorf("TopByScore (@Call, []T) = kind %v multi %v, want read/multi", top.Kind, top.Return.Multi)
	}
}

func TestBatchWithoutSliceIsRejected(t *testing.T) {
	res := analyzeApp(t, "./testdata/batchnoslice")
	found := false
	for _, d := range res.Diagnostics {
		if d.Code == CodeInvalidQuerySignature {
			found = true
		}
	}
	if !found {
		t.Errorf("expected %s for an @Batch without a slice parameter", CodeInvalidQuerySignature)
	}
}

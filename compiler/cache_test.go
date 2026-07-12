package compiler

import (
	"strings"
	"testing"
)

func TestCacheableDiscovery(t *testing.T) {
	res := analyzeApp(t, "./testdata/cacheapp")
	if errs := errorDiags(res.Diagnostics); len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	store := componentByName(res.App, "store")
	if store == nil || !store.Proxied {
		t.Fatal("store service should be proxied")
	}
	byName := map[string]int{}
	for i, m := range store.Intercepted {
		byName[m.Name] = i
	}
	get := store.Intercepted[byName["Get"]]
	if get.Cacheable == nil {
		t.Fatalf("Get should be @Cacheable")
	}
	if get.Cacheable.TTL == 0 {
		t.Errorf("Get @Cacheable should carry a ttl")
	}
	// key "store:#{id}" → literal "store:" then argument index 1 (id).
	parts := get.Cacheable.Parts
	if len(parts) != 2 || parts[0].Literal != "store:" || !parts[1].IsArg || parts[1].ArgIndex != 1 {
		t.Errorf("Get cache key parts = %+v", parts)
	}
	if put := store.Intercepted[byName["Put"]]; put.CacheEvict == nil {
		t.Errorf("Put should be @CacheEvict")
	}
}

func TestCacheableValidationErrors(t *testing.T) {
	res := analyzeApp(t, "./testdata/badcache")
	var b strings.Builder
	for _, d := range res.Diagnostics {
		b.WriteString(d.Message)
		b.WriteByte('\n')
	}
	msgs := b.String()
	if !strings.Contains(msgs, "must return exactly one value and an error") {
		t.Errorf("expected a signature diagnostic for @Cacheable Save; got:\n%s", msgs)
	}
	if !strings.Contains(msgs, `references unknown parameter "missing"`) {
		t.Errorf("expected an unknown-parameter diagnostic for @Cacheable Fetch; got:\n%s", msgs)
	}
}

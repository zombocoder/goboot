// Package cachee2e drives the generated @Cacheable / @CacheEvict proxy to
// confirm that a cache hit skips the target and a write invalidates the entry.
// wiring.gen.go is produced by the goboot generator from the cacheapp example.
package cachee2e

import (
	"context"
	"testing"

	"github.com/zombocoder/goboot/runtime"
)

func newComps(t *testing.T) *Components {
	t.Helper()
	comps, err := buildComponents(runtime.DefaultProxyDependencies()) // Cache = MemoryCache
	if err != nil {
		t.Fatalf("buildComponents: %v", err)
	}
	return comps
}

func TestCacheableHitSkipsTarget(t *testing.T) {
	comps := newComps(t)
	ctx := context.Background()

	// Seed the backing data (Put also evicts, but nothing is cached yet).
	if err := comps.StoreServiceProxy.Put(ctx, "a", "hello"); err != nil {
		t.Fatal(err)
	}

	// First Get: cache miss → reaches the target and caches the result.
	if v, _ := comps.StoreServiceProxy.Get(ctx, "a"); v != "hello" {
		t.Errorf("Get = %q, want hello", v)
	}
	if comps.Store.Calls() != 1 {
		t.Fatalf("target calls = %d, want 1 after first Get", comps.Store.Calls())
	}

	// Second Get: cache hit → target is NOT called again.
	if v, _ := comps.StoreServiceProxy.Get(ctx, "a"); v != "hello" {
		t.Errorf("cached Get = %q, want hello", v)
	}
	if comps.Store.Calls() != 1 {
		t.Errorf("target calls = %d, want still 1 on a cache hit", comps.Store.Calls())
	}
}

func TestCacheEvictInvalidates(t *testing.T) {
	comps := newComps(t)
	ctx := context.Background()

	_ = comps.StoreServiceProxy.Put(ctx, "a", "hello")
	_, _ = comps.StoreServiceProxy.Get(ctx, "a") // caches "hello" (calls=1)

	// A write evicts the entry...
	if err := comps.StoreServiceProxy.Put(ctx, "a", "world"); err != nil {
		t.Fatal(err)
	}
	// ...so the next Get misses and reaches the target again with fresh data.
	if v, _ := comps.StoreServiceProxy.Get(ctx, "a"); v != "world" {
		t.Errorf("post-evict Get = %q, want world", v)
	}
	if comps.Store.Calls() != 2 {
		t.Errorf("target calls = %d, want 2 (miss, then miss after evict)", comps.Store.Calls())
	}
}

// Distinct arguments map to distinct cache keys.
func TestCacheKeyPerArgument(t *testing.T) {
	comps := newComps(t)
	ctx := context.Background()
	_ = comps.StoreServiceProxy.Put(ctx, "a", "AAA")
	_ = comps.StoreServiceProxy.Put(ctx, "b", "BBB")

	a, _ := comps.StoreServiceProxy.Get(ctx, "a")
	b, _ := comps.StoreServiceProxy.Get(ctx, "b")
	if a != "AAA" || b != "BBB" {
		t.Errorf("keys crossed: a=%q b=%q", a, b)
	}
	if comps.Store.Calls() != 2 {
		t.Errorf("target calls = %d, want 2 (one per distinct key)", comps.Store.Calls())
	}
}

package runtime

import (
	"context"
	"testing"
	"time"
)

func TestMemoryCacheSetGetDelete(t *testing.T) {
	c := NewMemoryCache()
	ctx := context.Background()

	if _, ok, _ := c.Get(ctx, "missing"); ok {
		t.Error("empty cache should miss")
	}
	if err := c.Set(ctx, "k", []byte("v"), 0); err != nil {
		t.Fatal(err)
	}
	got, ok, err := c.Get(ctx, "k")
	if err != nil || !ok || string(got) != "v" {
		t.Fatalf("Get = %q, %v, %v", got, ok, err)
	}
	if err := c.Delete(ctx, "k"); err != nil {
		t.Fatal(err)
	}
	if _, ok, _ := c.Get(ctx, "k"); ok {
		t.Error("deleted key should miss")
	}
	// Deleting an absent key is fine.
	if err := c.Delete(ctx, "absent"); err != nil {
		t.Errorf("delete absent: %v", err)
	}
}

func TestMemoryCacheTTLExpiry(t *testing.T) {
	c := NewMemoryCache()
	now := time.Unix(1000, 0)
	c.now = func() time.Time { return now }
	ctx := context.Background()

	if err := c.Set(ctx, "k", []byte("v"), time.Minute); err != nil {
		t.Fatal(err)
	}
	if _, ok, _ := c.Get(ctx, "k"); !ok {
		t.Error("entry should be live before TTL")
	}
	now = now.Add(2 * time.Minute) // past TTL
	if _, ok, _ := c.Get(ctx, "k"); ok {
		t.Error("entry should have expired after TTL")
	}
}

// Stored values are copied, so a caller mutating its buffer can't corrupt the
// cache, and a returned slice can't corrupt the stored entry.
func TestMemoryCacheCopies(t *testing.T) {
	c := NewMemoryCache()
	ctx := context.Background()
	src := []byte("value")
	_ = c.Set(ctx, "k", src, 0)
	src[0] = 'X' // mutate the caller's buffer after Set

	got, _, _ := c.Get(ctx, "k")
	if string(got) != "value" {
		t.Errorf("stored value was corrupted by caller mutation: %q", got)
	}
	got[0] = 'Y' // mutate the returned slice
	again, _, _ := c.Get(ctx, "k")
	if string(again) != "value" {
		t.Errorf("stored value was corrupted by returned-slice mutation: %q", again)
	}
}

func TestDefaultProxyDependenciesHasCache(t *testing.T) {
	if DefaultProxyDependencies().Cache == nil {
		t.Error("default proxy dependencies must provide a Cache")
	}
}

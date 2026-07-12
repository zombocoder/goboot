package redis

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"
)

func newCache(t *testing.T, opts ...Option) (*Cache, *miniredis.Miniredis) {
	t.Helper()
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis: %v", err)
	}
	t.Cleanup(mr.Close)
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	return New(client, opts...), mr
}

func TestSetGetDelete(t *testing.T) {
	c, _ := newCache(t)
	ctx := context.Background()

	if _, ok, err := c.Get(ctx, "missing"); ok || err != nil {
		t.Errorf("miss = ok:%v err:%v, want false/nil", ok, err)
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
}

func TestTTLExpiry(t *testing.T) {
	c, mr := newCache(t)
	ctx := context.Background()
	if err := c.Set(ctx, "k", []byte("v"), time.Minute); err != nil {
		t.Fatal(err)
	}
	if _, ok, _ := c.Get(ctx, "k"); !ok {
		t.Fatal("entry should be live before TTL")
	}
	mr.FastForward(2 * time.Minute) // advance miniredis' clock past the TTL
	if _, ok, _ := c.Get(ctx, "k"); ok {
		t.Error("entry should have expired")
	}
}

func TestPrefixNamespacesKeys(t *testing.T) {
	c, mr := newCache(t, WithPrefix("app:"))
	ctx := context.Background()
	if err := c.Set(ctx, "k", []byte("v"), 0); err != nil {
		t.Fatal(err)
	}
	if !mr.Exists("app:k") {
		t.Errorf("stored key should be namespaced as app:k; keys = %v", mr.Keys())
	}
	// The prefix is transparent to the caller.
	if got, ok, _ := c.Get(ctx, "k"); !ok || string(got) != "v" {
		t.Errorf("prefixed Get = %q, %v", got, ok)
	}
}

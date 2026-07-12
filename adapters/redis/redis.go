// Package redis implements goboot's runtime.Cache over Redis, backing
// @Cacheable / @CacheEvict with a shared, out-of-process cache (§32). Wire it
// into the composition root:
//
//	rdb := goredis.NewClient(&goredis.Options{Addr: "localhost:6379"})
//	proxyDeps.Cache = redis.New(rdb, redis.WithPrefix("todo:"))
//
// It is a separate module so the go-redis dependency stays out of the core.
package redis

import (
	"context"
	"errors"
	"time"

	goredis "github.com/redis/go-redis/v9"
	"github.com/zombocoder/goboot/runtime"
)

// Cache is a runtime.Cache backed by a go-redis client.
type Cache struct {
	client goredis.UniversalClient
	prefix string
}

// compile-time proof that Cache satisfies the runtime seam.
var _ runtime.Cache = (*Cache)(nil)

// Option configures a Cache.
type Option func(*Cache)

// WithPrefix namespaces every key with prefix, so multiple apps can share a
// Redis instance without colliding.
func WithPrefix(prefix string) Option { return func(c *Cache) { c.prefix = prefix } }

// New wraps a go-redis client (any of *Client, *ClusterClient, *Ring) as a
// runtime.Cache.
func New(client goredis.UniversalClient, opts ...Option) *Cache {
	c := &Cache{client: client}
	for _, o := range opts {
		o(c)
	}
	return c
}

func (c *Cache) key(k string) string { return c.prefix + k }

// Get returns the value for key; a missing key is (nil, false, nil).
func (c *Cache) Get(ctx context.Context, key string) ([]byte, bool, error) {
	b, err := c.client.Get(ctx, c.key(key)).Bytes()
	if errors.Is(err, goredis.Nil) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return b, true, nil
}

// Set stores value under key with an optional ttl (0 means no expiry, matching
// Redis semantics).
func (c *Cache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	return c.client.Set(ctx, c.key(key), value, ttl).Err()
}

// Delete removes key. Deleting an absent key is not an error.
func (c *Cache) Delete(ctx context.Context, key string) error {
	return c.client.Del(ctx, c.key(key)).Err()
}

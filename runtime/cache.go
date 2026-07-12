package runtime

import (
	"context"
	"sync"
	"time"
)

// Cache is a key/value cache with per-entry TTL, consulted by @Cacheable /
// @CacheEvict generated proxies and available for injection (§32). Values are
// opaque bytes — the generated code serializes method results (JSON) — so an
// adapter (e.g. Redis) only moves bytes.
type Cache interface {
	// Get returns the cached value for key and whether it was present. A cache
	// miss is (nil, false, nil); an error is reserved for backend failures.
	Get(ctx context.Context, key string) (value []byte, found bool, err error)
	// Set stores value under key. A ttl of 0 means no expiry.
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	// Delete removes key. Deleting an absent key is not an error.
	Delete(ctx context.Context, key string) error
}

// MemoryCache is an in-process Cache with lazy TTL expiry. It is the default so
// @Cacheable works out of the box; swap in the Redis adapter for a shared cache.
// It is safe for concurrent use.
type MemoryCache struct {
	mu      sync.RWMutex
	entries map[string]memoryEntry
	now     func() time.Time
}

type memoryEntry struct {
	value   []byte
	expires time.Time // zero means no expiry
}

// NewMemoryCache returns an empty in-memory cache.
func NewMemoryCache() *MemoryCache {
	return &MemoryCache{entries: make(map[string]memoryEntry), now: time.Now}
}

// Get returns the value for key, treating an expired entry as a miss.
func (c *MemoryCache) Get(_ context.Context, key string) ([]byte, bool, error) {
	c.mu.RLock()
	e, ok := c.entries[key]
	c.mu.RUnlock()
	if !ok {
		return nil, false, nil
	}
	if !e.expires.IsZero() && !c.now().Before(e.expires) {
		c.mu.Lock()
		// Re-check under the write lock in case a concurrent Set refreshed it.
		if cur, ok := c.entries[key]; ok && cur.expires.Equal(e.expires) {
			delete(c.entries, key)
		}
		c.mu.Unlock()
		return nil, false, nil
	}
	out := make([]byte, len(e.value))
	copy(out, e.value)
	return out, true, nil
}

// Set stores a copy of value under key with an optional ttl.
func (c *MemoryCache) Set(_ context.Context, key string, value []byte, ttl time.Duration) error {
	stored := make([]byte, len(value))
	copy(stored, value)
	var expires time.Time
	if ttl > 0 {
		expires = c.now().Add(ttl)
	}
	c.mu.Lock()
	c.entries[key] = memoryEntry{value: stored, expires: expires}
	c.mu.Unlock()
	return nil
}

// Delete removes key.
func (c *MemoryCache) Delete(_ context.Context, key string) error {
	c.mu.Lock()
	delete(c.entries, key)
	c.mu.Unlock()
	return nil
}

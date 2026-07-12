// Package cacheapp exercises @Cacheable / @CacheEvict service caching (§32): the
// generated proxy returns a cached result on a hit and invalidates on a write.
package cacheapp

import "context"

// @Application(name="cache-app")
type Application struct{}

// Store is the cached service interface.
type Store interface {
	Get(ctx context.Context, id string) (string, error)
	Put(ctx context.Context, id string, value string) error
}

// StoreService counts target calls so a test can prove caching short-circuits.
//
// @Service(name="store", implements="Store")
type StoreService struct {
	calls int
	data  map[string]string
}

// NewStoreService constructs the service.
func NewStoreService() *StoreService { return &StoreService{data: map[string]string{}} }

// Calls reports how many times Get reached the target (test helper).
func (s *StoreService) Calls() int { return s.calls }

// Get returns the stored value; with caching the target runs only on a miss.
//
// @Cacheable(key="store:#{id}", ttl="1m")
func (s *StoreService) Get(ctx context.Context, id string) (string, error) {
	s.calls++
	return s.data[id], nil
}

// Put writes a value and evicts the cached entry for that id.
//
// @CacheEvict(key="store:#{id}")
func (s *StoreService) Put(ctx context.Context, id string, value string) error {
	s.data[id] = value
	return nil
}

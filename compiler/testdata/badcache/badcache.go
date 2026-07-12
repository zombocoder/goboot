// Package badcache exercises the @Cacheable validation diagnostics.
package badcache

import "context"

// @Application(name="bad-cache")
type Application struct{}

// Store is the service interface.
type Store interface {
	Save(ctx context.Context, id string) error
	Fetch(ctx context.Context, id string) (string, error)
}

// @Service(name="store", implements="Store")
type StoreService struct{}

// NewStoreService constructs it.
func NewStoreService() *StoreService { return &StoreService{} }

// Save is @Cacheable but returns only an error (no value to cache) → invalid.
//
// @Cacheable(key="k:#{id}")
func (s *StoreService) Save(ctx context.Context, id string) error { return nil }

// Fetch has a valid signature but its key references an unknown parameter.
//
// @Cacheable(key="k:#{missing}")
func (s *StoreService) Fetch(ctx context.Context, id string) (string, error) { return "", nil }

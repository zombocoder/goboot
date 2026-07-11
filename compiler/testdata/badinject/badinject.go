// Package badinject injects a proxied service by its concrete type, which is
// forbidden because it would bypass interception (§24.3).
package badinject

import "context"

type UseCase interface {
	Do(ctx context.Context) error
}

// @Service(implements="UseCase")
type Svc struct{}

func NewSvc() *Svc { return &Svc{} }

// @Transactional
func (s *Svc) Do(ctx context.Context) error { return nil }

// Consumer wrongly depends on the concrete *Svc rather than the UseCase
// interface.
//
// @Service
type Consumer struct{}

func NewConsumer(s *Svc) *Consumer { return &Consumer{} }

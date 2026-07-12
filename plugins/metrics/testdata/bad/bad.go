// Package bad is a fixture of invalid metric declarations, exercising the
// analyzer's diagnostics.
package bad

import "context"

// @Application(name="bad-metrics")
type Application struct{}

// @Service(name="svc")
type Service struct{}

// NewService constructs it.
func NewService() *Service { return &Service{} }

// A metric name with a dash is not a valid Prometheus name.
//
// @Counter(name="bad-name")
func (s *Service) A(ctx context.Context) error { return nil }

// A label with a dash is not a valid Prometheus label.
//
// @Counter(name="ok_total", labels=["bad-label"])
func (s *Service) B(ctx context.Context) error { return nil }

// A duplicate of the same metric name.
//
// @Gauge(name="dupe")
func (s *Service) C(ctx context.Context) error { return nil }

// @Gauge(name="dupe")
func (s *Service) D(ctx context.Context) error { return nil }

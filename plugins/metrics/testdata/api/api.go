// Package api is a fixture exercising the metrics generator: counters and gauges,
// with and without labels and a namespace, on service methods.
package api

import "context"

// @Application(name="metrics-api")
type Application struct{}

// OrderService carries the metric annotations.
//
// @Service(name="orders")
type OrderService struct{}

// NewOrderService constructs the service.
func NewOrderService() *OrderService { return &OrderService{} }

// Process handles an order.
//
// @Counter(name="orders_processed_total", help="Orders processed, by status", labels=["status"])
func (s *OrderService) Process(ctx context.Context) error { return nil }

// Lookup reads an order.
//
// @Counter(name="cache_hits_total", help="Order cache hits")
func (s *OrderService) Lookup(ctx context.Context) error { return nil }

// Enqueue queues work.
//
// @Gauge(name="queue_depth", help="Pending items in the queue", namespace="orders")
func (s *OrderService) Enqueue(ctx context.Context) error { return nil }

// Requests declares a metric on a plain marker struct (not a component),
// exercising the @Counter/@Gauge struct target.
//
// @Counter(name="http_requests_total", help="HTTP requests, by method", labels=["method"])
type Requests struct{}

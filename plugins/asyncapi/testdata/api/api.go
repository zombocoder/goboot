// Package api is a fixture exercising the AsyncAPI generator: a @Listener
// (receive) and a @Publisher (send) with struct payloads.
package api

import (
	"context"
	"time"
)

// @Application(name="orders-events")
type Application struct{}

// OrderCreated is the payload of the orders.created event.
type OrderCreated struct {
	ID        string    `json:"id"`
	Total     float64   `json:"total"`
	CreatedAt time.Time `json:"createdAt"`
}

// OrderShipped is the payload of the orders.shipped event.
type OrderShipped struct {
	ID       string `json:"id"`
	Tracking string `json:"tracking"`
}

// OrderHandler consumes and produces order events.
//
// @Service(name="orderHandler")
type OrderHandler struct{}

// NewOrderHandler constructs the handler.
func NewOrderHandler() *OrderHandler { return &OrderHandler{} }

// OnOrderCreated handles an order-created event (the app receives).
//
// @Listener(channel="orders.created")
func (h *OrderHandler) OnOrderCreated(ctx context.Context, evt OrderCreated) error { return nil }

// ShipOrder publishes an order-shipped event (the app sends).
//
// @Publisher(channel="orders.shipped", summary="Emitted when an order ships")
func (h *OrderHandler) ShipOrder(ctx context.Context, evt OrderShipped) error { return nil }

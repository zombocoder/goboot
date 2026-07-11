// Package proxyapp exercises interface service proxies and interception.
package proxyapp

import "context"

// @Application(name="proxy-app")
type Application struct{}

// OrderUseCase is the interface the service is exposed as.
type OrderUseCase interface {
	CreateOrder(ctx context.Context, name string) (string, error)
	GetOrder(ctx context.Context, id string) (string, error)
}

// OrderService implements OrderUseCase and has intercepted methods.
//
// @Service(name="orderService", implements="OrderUseCase")
type OrderService struct{}

// NewOrderService constructs the service.
func NewOrderService() *OrderService { return &OrderService{} }

// CreateOrder is fully intercepted: traced, timed, and transactional.
//
// @Transactional
// @Traced(name="orders.create")
// @Timed(name="orders.create")
func (s *OrderService) CreateOrder(ctx context.Context, name string) (string, error) {
	if name == "boom" {
		return "", errBoom
	}
	return "order-" + name, nil
}

// errBoom is returned by CreateOrder to exercise the rollback path.
var errBoom = errBoomError("order creation failed")

type errBoomError string

func (e errBoomError) Error() string { return string(e) }

// GetOrder is not intercepted; the proxy delegates to it directly.
func (s *OrderService) GetOrder(ctx context.Context, id string) (string, error) {
	return "order:" + id, nil
}

// OrderController depends on the OrderUseCase interface, which resolves to the
// generated proxy.
//
// @RestController
// @RequestMapping(path="/orders")
type OrderController struct {
	orders OrderUseCase
}

// NewOrderController injects the interface.
func NewOrderController(orders OrderUseCase) *OrderController {
	return &OrderController{orders: orders}
}

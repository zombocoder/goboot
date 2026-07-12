// Package api is a fixture exercising the OpenAPI generator: path/query
// parameters, a JSON request body, and a response schema with varied field
// types.
package api

import (
	"context"
	"time"
)

// @Application(name="widget-api")
type Application struct{}

// Widget is the response entity.
type Widget struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Quantity  int64     `json:"quantity"`
	Price     float64   `json:"price"`
	Active    bool      `json:"active"`
	Tags      []string  `json:"tags"`
	CreatedAt time.Time `json:"createdAt"`
}

// WidgetController serves widgets.
//
// @RestController
// @RequestMapping(path="/widgets")
type WidgetController struct{}

// NewWidgetController constructs the controller.
func NewWidgetController() *WidgetController { return &WidgetController{} }

// GetWidgetRequest binds a path id and a query flag.
type GetWidgetRequest struct {
	ID     string `path:"id"`
	Expand bool   `query:"expand"`
}

// GetWidget returns a single widget.
//
// @GetMapping(path="/{id}")
func (c *WidgetController) GetWidget(ctx context.Context, req GetWidgetRequest) (*Widget, error) {
	return &Widget{ID: req.ID}, nil
}

// CreateWidgetRequest is the JSON body for creating a widget.
type CreateWidgetRequest struct {
	Name     string `json:"name"`
	Quantity int64  `json:"quantity"`
}

// CreateWidget creates a widget.
//
// @PostMapping(path="", produces=["application/json"])
func (c *WidgetController) CreateWidget(ctx context.Context, req CreateWidgetRequest) (*Widget, error) {
	return &Widget{Name: req.Name, Quantity: req.Quantity}, nil
}

// DeleteWidget removes a widget; it requires the "admin" role.
//
// @DeleteMapping(path="/{id}")
// @Authorize(roles=["admin"])
func (c *WidgetController) DeleteWidget(ctx context.Context, req GetWidgetRequest) error {
	return nil
}

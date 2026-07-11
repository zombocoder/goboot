// Package verbapp exercises the HTTP verb mappings @PutMapping, @PatchMapping,
// and @DeleteMapping alongside GET/POST (§17).
package verbapp

import "context"

// @Application(name="verb-app")
type Application struct{}

// Widget is a trivial resource returned by the controller.
type Widget struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// WidgetController maps every supported HTTP verb.
//
// @RestController
// @RequestMapping(path="/widgets")
type WidgetController struct{}

// NewWidgetController constructs a WidgetController.
func NewWidgetController() *WidgetController { return &WidgetController{} }

// GetWidget handles GET /widgets/{id}.
//
// @GetMapping(path="/{id}")
func (c *WidgetController) GetWidget(ctx context.Context, req IDRequest) (*Widget, error) {
	return &Widget{ID: req.ID, Name: "gadget"}, nil
}

// CreateWidget handles POST /widgets.
//
// @PostMapping(path="")
func (c *WidgetController) CreateWidget(ctx context.Context, req NameRequest) (*Widget, error) {
	return &Widget{ID: "new", Name: req.Name}, nil
}

// ReplaceWidget handles PUT /widgets/{id} (default 200).
//
// @PutMapping(path="/{id}")
func (c *WidgetController) ReplaceWidget(ctx context.Context, req UpdateRequest) (*Widget, error) {
	return &Widget{ID: req.ID, Name: req.Name}, nil
}

// PatchWidget handles PATCH /widgets/{id} (default 200).
//
// @PatchMapping(path="/{id}")
func (c *WidgetController) PatchWidget(ctx context.Context, req UpdateRequest) (*Widget, error) {
	return &Widget{ID: req.ID, Name: req.Name}, nil
}

// DeleteWidget handles DELETE /widgets/{id} (default 204, no body).
//
// @DeleteMapping(path="/{id}")
func (c *WidgetController) DeleteWidget(ctx context.Context, req IDRequest) error {
	return nil
}

// IDRequest binds the path id.
type IDRequest struct {
	ID string `path:"id"`
}

// NameRequest binds a JSON name.
type NameRequest struct {
	Name string `json:"name"`
}

// UpdateRequest binds a path id and a JSON name.
type UpdateRequest struct {
	ID   string `path:"id"`
	Name string `json:"name"`
}

// Package negapp exercises @Consumes / @Produces content negotiation (§19),
// declared both as mapping arguments and as standalone annotations.
package negapp

import "context"

// @Application(name="neg-app")
type Application struct{}

// Doc is a trivial resource.
type Doc struct {
	Body string `json:"body"`
}

// DocController negotiates content types.
//
// @RestController
// @RequestMapping(path="/docs")
type DocController struct{}

// NewDocController constructs a DocController.
func NewDocController() *DocController { return &DocController{} }

// Create consumes and produces JSON, declared via mapping arguments.
//
// @PostMapping(path="", consumes=["application/json"], produces=["application/json"])
func (c *DocController) Create(ctx context.Context, req Doc) (*Doc, error) {
	return &Doc{Body: req.Body}, nil
}

// Render only produces JSON, declared via the standalone @Produces annotation.
//
// @GetMapping(path="/{id}")
// @Produces(["application/json"])
func (c *DocController) Render(ctx context.Context, req IDRequest) (*Doc, error) {
	return &Doc{Body: req.ID}, nil
}

// IDRequest binds the path id.
type IDRequest struct {
	ID string `path:"id"`
}

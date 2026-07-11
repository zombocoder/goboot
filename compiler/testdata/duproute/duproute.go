package duproute

import "context"

// @RestController
// @RequestMapping(path="/x")
type C struct{}

func NewC() *C { return &C{} }

// @GetMapping(path="/a")
func (c *C) A(ctx context.Context) error { return nil }

// B collides with A on GET /x/a.
//
// @GetMapping(path="/a")
func (c *C) B(ctx context.Context) error { return nil }

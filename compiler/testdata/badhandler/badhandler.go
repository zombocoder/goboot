package badhandler

// @RestController
type C struct{}

func NewC() *C { return &C{} }

// Bad has an unsupported handler signature: its first parameter is not
// context.Context.
//
// @GetMapping(path="/x")
func (c *C) Bad(x int) error { return nil }

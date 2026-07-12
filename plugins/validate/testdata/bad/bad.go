// Package bad is a fixture of misapplied constraints, exercising the analyzer's
// diagnostics. It is never expected to generate a validator.
package bad

import "context"

// @Application(name="bad-api")
type Application struct{}

// BadRequest misapplies constraints to incompatible field types.
type BadRequest struct {
	// @Min(3) on a string is a type error.
	// @Min(3)
	Name string `json:"name"`
	// @Pattern on a numeric field is a type error.
	// @Pattern("^x$")
	Count int `json:"count"`
	// @Size with min > max is invalid.
	// @Size(min=10, max=2)
	Tags []string `json:"tags"`
	// @Pattern with an uncompilable regex is rejected.
	// @Pattern("[")
	Code string `json:"code"`
}

// Detached is not a request type, so its constraints are unenforced.
type Detached struct {
	// @Required
	Note string `json:"note"`
}

// BadController serves the bad request.
//
// @RestController
// @RequestMapping(path="/bad")
type BadController struct{}

// NewBadController constructs the controller.
func NewBadController() *BadController { return &BadController{} }

// Do handles the bad request.
//
// @PostMapping(path="")
func (c *BadController) Do(ctx context.Context, req BadRequest) error { return nil }

// Package api is a fixture exercising the validate generator: every constraint
// annotation across string, numeric, and slice fields on HTTP request types.
package api

import "context"

// @Application(name="valid-api")
type Application struct{}

// UserResponse is the response entity.
type UserResponse struct {
	ID string `json:"id"`
}

// CreateUserRequest is the JSON body for creating a user.
type CreateUserRequest struct {
	// @Required
	// @Size(min=3, max=40)
	Name string `json:"name"`
	// @Required
	// @Email
	Email string `json:"email"`
	// @Min(0)
	// @Max(150)
	Age int `json:"age"`
	// @Pattern("^[a-z][a-z0-9-]*$")
	Slug string `json:"slug"`
	// @Size(max=5)
	Tags []string `json:"tags"`
}

// GetUserRequest binds a path id.
type GetUserRequest struct {
	// @Required
	ID string `path:"id"`
}

// UserController serves users.
//
// @RestController
// @RequestMapping(path="/users")
type UserController struct{}

// NewUserController constructs the controller.
func NewUserController() *UserController { return &UserController{} }

// Create creates a user.
//
// @PostMapping(path="")
func (c *UserController) Create(ctx context.Context, req CreateUserRequest) (*UserResponse, error) {
	return &UserResponse{}, nil
}

// Get returns a user.
//
// @GetMapping(path="/{id}")
func (c *UserController) Get(ctx context.Context, req GetUserRequest) (*UserResponse, error) {
	return &UserResponse{ID: req.ID}, nil
}

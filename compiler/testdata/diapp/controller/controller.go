package controller

import (
	"context"

	"github.com/zombocoder/goboot/compiler/testdata/diapp/domain"
)

// UserController depends on the domain.UserUseCase interface, which the service
// satisfies.
//
// @RestController
// @RequestMapping(path="/api/v1/users")
type UserController struct {
	service domain.UserUseCase
}

// NewUserController constructs a UserController.
func NewUserController(service domain.UserUseCase) *UserController {
	return &UserController{service: service}
}

// GetUserRequest is the bound request for GetUser.
type GetUserRequest struct {
	ID string `path:"id"`
}

// GetUser handles GET /{id}.
//
// @GetMapping(path="/{id}")
// @Response(status=200)
func (c *UserController) GetUser(ctx context.Context, req GetUserRequest) (*domain.User, error) {
	return c.service.GetUser(req.ID)
}

// CreateUserRequest is the bound request body for CreateUser.
type CreateUserRequest struct {
	Name string `json:"name"`
}

// CreateUser handles POST /.
//
// @PostMapping(path="")
// @Response(status=201)
func (c *UserController) CreateUser(ctx context.Context, req CreateUserRequest) (*domain.User, error) {
	return &domain.User{Name: req.Name}, nil
}

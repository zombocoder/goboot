package controller

import "github.com/zombocoder/goboot/compiler/testdata/diapp/domain"

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

// GetUser handles GET /{id}.
//
// @GetMapping(path="/{id}")
func (c *UserController) GetUser(id string) (*domain.User, error) {
	return c.service.GetUser(id)
}

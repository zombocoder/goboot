// Package basic is a fixture exercising the scanner's declaration association.
//
// @Application(name="basic-service", scan=["./..."])
package basic

import "context"

// UserService coordinates user use cases.
//
// @Service(name="userService", scope="singleton")
type UserService struct {
	// repository is injected by constructor; not annotated.
	repository UserRepository
}

// NewUserService constructs a UserService.
func NewUserService(repository UserRepository) *UserService {
	return &UserService{repository: repository}
}

// GetName is an ordinary method with no annotations.
func (s *UserService) GetName() string { return "user" }

// UserRepository reads users.
//
// @Repository(entity="User", table="users", generate=true)
type UserRepository interface {
	// FindByID loads a user.
	//
	// @Query(`SELECT id FROM users WHERE id = :id`)
	FindByID(ctx context.Context, id string) (string, error)
}

// UserController serves the user API.
//
// @RestController
// @RequestMapping(path="/api/v1/users")
type UserController struct {
	service *UserService
}

// GetUser handles GET /{id}.
//
// @GetMapping(path="/{id}")
// @Response(status=200)
// @Response(status=404, error="user_not_found")
func (c *UserController) GetUser(ctx context.Context, id string) (string, error) {
	return c.service.repository.FindByID(ctx, id)
}

// Clock is a marker component with a field annotation.
//
// @Component
type Clock struct {
	// Zone selects the timezone.
	//
	// @Named("zone")
	Zone string
}

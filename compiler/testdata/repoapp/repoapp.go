// Package repoapp exercises generated repositories from @Query/@Exec.
package repoapp

import "context"

// @Application(name="repo-app")
type Application struct{}

// User is the entity mapped by the repository.
type User struct {
	ID    string
	Name  string
	Email string
}

// UserRepository is a generated repository.
//
// @Repository(generate=true, entity="User", table="users")
type UserRepository interface {
	// FindByID returns a single entity or a not-found error.
	//
	// @Query(`SELECT id, name, email FROM users WHERE id = :id`)
	FindByID(ctx context.Context, id string) (*User, error)

	// FindAll returns all entities.
	//
	// @Query(`SELECT id, name, email FROM users`)
	FindAll(ctx context.Context) ([]*User, error)

	// Count returns a scalar.
	//
	// @Query(`SELECT count(*) FROM users`)
	Count(ctx context.Context) (int64, error)

	// Create inserts a row and returns only an error.
	//
	// @Exec(`INSERT INTO users (id, name, email) VALUES (:id, :name, :email)`)
	Create(ctx context.Context, id string, name string, email string) error

	// Delete removes a row and returns the affected count.
	//
	// @Exec(`DELETE FROM users WHERE id = :id`)
	Delete(ctx context.Context, id string) (int64, error)
}

// UserService depends on the repository interface, which resolves to the
// generated implementation.
//
// @Service(name="userService")
type UserService struct {
	repo UserRepository
}

// NewUserService injects the repository.
func NewUserService(repo UserRepository) *UserService {
	return &UserService{repo: repo}
}

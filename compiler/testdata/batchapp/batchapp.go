// Package batchapp exercises the @Batch and @Call repository annotations
// (§27.3): batch execution over a slice and stored-procedure invocation.
package batchapp

import "context"

// @Application(name="batch-app")
type Application struct{}

// User is the entity mapped by the repository.
type User struct {
	ID   string
	Name string
}

// UserRepository is a generated repository.
//
// @Repository(generate=true, entity="User", table="users")
type UserRepository interface {
	// InsertAll batches an insert over every user, returning the total rows
	// affected (element fields are bound via the slice parameter name).
	//
	// @Batch(`INSERT INTO users (id, name) VALUES (:users.ID, :users.Name)`)
	InsertAll(ctx context.Context, users []User) (int64, error)

	// TouchAll batches an update scoped by a shared org id, returning only an
	// error (mixing a scalar parameter with the iterated slice).
	//
	// @Batch(`UPDATE users SET org = :org WHERE id = :ids`)
	TouchAll(ctx context.Context, org string, ids []string) error

	// Reindex calls a procedure that returns nothing (exec form).
	//
	// @Call(`CALL reindex_users()`)
	Reindex(ctx context.Context) error

	// TopByScore calls a set-returning function scanned like a query.
	//
	// @Call(`SELECT id, name FROM top_users(:limit)`)
	TopByScore(ctx context.Context, limit int64) ([]*User, error)
}

// UserService depends on the repository interface.
//
// @Service(name="userService")
type UserService struct {
	repo UserRepository
}

// NewUserService injects the repository.
func NewUserService(repo UserRepository) *UserService { return &UserService{repo: repo} }

// Repo exposes the repository for tests.
func (s *UserService) Repo() UserRepository { return s.repo }

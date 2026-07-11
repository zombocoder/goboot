package repo

import "github.com/zombocoder/goboot/compiler/testdata/diapp/domain"

// PostgresUserRepository is a component-mode repository implementing
// domain.UserRepository.
//
// @Repository(name="userRepository")
type PostgresUserRepository struct{}

// NewPostgresUserRepository constructs the repository.
func NewPostgresUserRepository() *PostgresUserRepository {
	return &PostgresUserRepository{}
}

// FindByID satisfies domain.UserRepository.
func (r *PostgresUserRepository) FindByID(id string) (*domain.User, error) {
	return &domain.User{ID: id, Name: "example"}, nil
}

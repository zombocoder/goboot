package service

import "github.com/zombocoder/goboot/compiler/testdata/diapp/domain"

// UserService implements domain.UserUseCase and depends on a repository and an
// ID generator bean.
//
// @Service(name="userService")
type UserService struct {
	repo domain.UserRepository
	ids  domain.IDGenerator
}

// NewUserService constructs a UserService.
func NewUserService(repo domain.UserRepository, ids domain.IDGenerator) *UserService {
	return &UserService{repo: repo, ids: ids}
}

// GetUser satisfies domain.UserUseCase.
func (s *UserService) GetUser(id string) (*domain.User, error) {
	return s.repo.FindByID(id)
}

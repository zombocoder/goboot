// Package authapp exercises method-level @Authorize / @RolesAllowed (§34).
package authapp

import "context"

// @Application(name="auth-app")
type Application struct{}

// Admin is the interface the service is exposed as.
type Admin interface {
	DeleteAll(ctx context.Context) error
	Read(ctx context.Context) (string, error)
}

// AdminService has authorization-gated methods.
//
// @Service(name="admin", implements="Admin")
type AdminService struct {
	deleted bool
}

func NewAdminService() *AdminService { return &AdminService{} }

// Deleted reports whether DeleteAll reached the target (test helper).
func (s *AdminService) Deleted() bool { return s.deleted }

// DeleteAll requires the "admin" role.
//
// @Authorize(roles=["admin"], mode="all")
func (s *AdminService) DeleteAll(ctx context.Context) error {
	s.deleted = true
	return nil
}

// Read requires the "reader" role via the shorthand.
//
// @RolesAllowed(["reader"])
func (s *AdminService) Read(ctx context.Context) (string, error) {
	return "data", nil
}

// Package obsapp exercises method-level @Logged / @Audit observability
// interceptors (§35.3, §35.4).
package obsapp

import "context"

// @Application(name="obs-app")
type Application struct{}

// Vault is the interface the service is exposed as.
type Vault interface {
	Store(ctx context.Context, key string) error
	Rotate(ctx context.Context) (string, error)
}

// VaultService has logged and audited methods.
//
// @Service(name="vault", implements="Vault")
type VaultService struct {
	stored bool
	fail   bool
}

func NewVaultService() *VaultService { return &VaultService{} }

// Stored reports whether Store reached the target (test helper).
func (s *VaultService) Stored() bool { return s.stored }

// SetFail makes the next Store return an error (test helper).
func (s *VaultService) SetFail(v bool) { s.fail = v }

// Store logs at debug and records an audit event for the write.
//
// @Logged(level="debug")
// @Audit(action="store", resource="secret")
func (s *VaultService) Store(ctx context.Context, key string) error {
	if s.fail {
		return context.Canceled
	}
	s.stored = true
	return nil
}

// Rotate is logged at the default level.
//
// @Logged
func (s *VaultService) Rotate(ctx context.Context) (string, error) {
	return "rotated", nil
}

// Package authhttpapp exercises route-level @Authorize on an HTTP controller:
// the generated handler authenticates the caller, places the principal on the
// context, and authorizes before invoking the controller (§34).
package authhttpapp

import "context"

// @Application(name="auth-http-app")
type Application struct{}

// Secret is the protected response payload.
type Secret struct {
	Value string `json:"value"`
}

// SecretController serves a role-protected resource.
//
// @RestController
// @RequestMapping(path="/secret")
type SecretController struct{}

// NewSecretController constructs the controller.
func NewSecretController() *SecretController { return &SecretController{} }

// Get returns the secret; it requires the "user" role.
//
// @GetMapping(path="")
// @Authorize(roles=["user"])
func (c *SecretController) Get(ctx context.Context) (*Secret, error) {
	return &Secret{Value: "42"}, nil
}

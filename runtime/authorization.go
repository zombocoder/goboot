package runtime

import "context"

// AuthorizationMode selects whether all or any of the required roles must be
// satisfied (§34.1).
type AuthorizationMode int

const (
	// AuthorizationModeAny grants access when any required role matches.
	AuthorizationModeAny AuthorizationMode = iota
	// AuthorizationModeAll requires every listed role.
	AuthorizationModeAll
)

// AuthorizationRequest describes an access check for a route or method (§34.2).
type AuthorizationRequest struct {
	Roles       []string
	Permissions []string
	Mode        AuthorizationMode
	Resource    string
	Action      string
}

// Authorizer decides whether the current principal may proceed (§34.2). It
// returns an error (typically an HTTPStatusError with 401/403) to deny.
type Authorizer interface {
	Authorize(ctx context.Context, req AuthorizationRequest) error
}

// PermitAllAuthorizer allows every request. It is the default so generated
// handlers run before an authorization adapter is configured; a full
// authorization implementation is out of v0.1 scope (§54.2).
type PermitAllAuthorizer struct{}

// Authorize always permits.
func (PermitAllAuthorizer) Authorize(context.Context, AuthorizationRequest) error { return nil }

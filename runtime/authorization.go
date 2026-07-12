package runtime

import (
	"context"
	"slices"
)

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

// RoleAuthorizer enforces an AuthorizationRequest against the Principal on the
// context (established by an Authenticator). A request that requires no roles or
// permissions is always allowed. Otherwise an authenticated principal is
// required — its absence is a 401 (Unauthenticated) — and it must hold the
// required roles (and scopes, if any) per Mode, else a 403 (Forbidden). Select
// it as the Authorizer once authentication is wired.
type RoleAuthorizer struct{}

// Authorize checks the context Principal against req.
func (RoleAuthorizer) Authorize(ctx context.Context, req AuthorizationRequest) error {
	if len(req.Roles) == 0 && len(req.Permissions) == 0 {
		return nil // the route imposes no access restriction
	}
	p, ok := PrincipalFrom(ctx)
	if !ok || !p.IsAuthenticated() {
		return Unauthenticated("authentication required")
	}
	if !satisfies(p.Roles, req.Roles, req.Mode) || !satisfies(p.Scopes, req.Permissions, req.Mode) {
		return Forbidden("insufficient permissions")
	}
	return nil
}

// satisfies reports whether the granted set covers the required set under mode.
// An empty required set is always satisfied. AuthorizationModeAll needs every
// required value; AuthorizationModeAny needs at least one.
func satisfies(granted, required []string, mode AuthorizationMode) bool {
	if len(required) == 0 {
		return true
	}
	if mode == AuthorizationModeAll {
		for _, r := range required {
			if !slices.Contains(granted, r) {
				return false
			}
		}
		return true
	}
	for _, r := range required {
		if slices.Contains(granted, r) {
			return true
		}
	}
	return false
}

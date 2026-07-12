package runtime

import (
	"context"
	"slices"
)

// Principal is the caller identity established by an Authenticator and carried on
// the request context (§34). It is populated from a validated credential — for
// example an OIDC bearer token — by an authentication adapter; the framework
// core depends only on this shape, never on a concrete token format.
//
// The zero value is the anonymous principal: no subject, no roles.
type Principal struct {
	// Subject is the stable identifier of the caller (the token `sub` claim). It
	// is empty for an anonymous principal.
	Subject string
	// Username is a human-readable name when available (e.g. `preferred_username`).
	Username string
	// Roles are the caller's granted roles, matched against @Authorize /
	// @RolesAllowed by an authorizer.
	Roles []string
	// Scopes are the granted OAuth2 scopes, when present.
	Scopes []string
	// Claims are the remaining raw claims from the credential, for adapters and
	// application code that need more than the normalized fields.
	Claims map[string]any
}

// IsAuthenticated reports whether a principal was established from a credential.
func (p Principal) IsAuthenticated() bool { return p.Subject != "" }

// HasRole reports whether the principal was granted role.
func (p Principal) HasRole(role string) bool { return slices.Contains(p.Roles, role) }

// HasScope reports whether the principal was granted scope.
func (p Principal) HasScope(scope string) bool { return slices.Contains(p.Scopes, scope) }

// principalKey is the private context key for the current Principal.
type principalKey struct{}

// WithPrincipal returns a context carrying p. The authentication step stores the
// authenticated principal here before authorization and the controller run.
func WithPrincipal(ctx context.Context, p Principal) context.Context {
	return context.WithValue(ctx, principalKey{}, p)
}

// PrincipalFrom returns the Principal on ctx and whether one was set. When no
// principal has been established it returns the zero (anonymous) principal and
// false.
func PrincipalFrom(ctx context.Context) (Principal, bool) {
	p, ok := ctx.Value(principalKey{}).(Principal)
	return p, ok
}

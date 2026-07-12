package runtime

import (
	"context"
	"net/http"
)

// Authenticator establishes the caller's Principal from an incoming request —
// typically by validating an `Authorization: Bearer` access token (§34). It runs
// before authorization; the generated handler stores the returned Principal on
// the request context with WithPrincipal so the authorizer and the controller
// can read it via PrincipalFrom.
//
// A present-but-invalid credential (bad signature, expired, wrong audience)
// should return an error that maps to 401 — see Unauthenticated. A request that
// simply carries no credential should return the anonymous Principal and a nil
// error, letting the authorizer decide whether the route requires identity.
type Authenticator interface {
	Authenticate(ctx context.Context, r *http.Request) (Principal, error)
}

// AnonymousAuthenticator establishes no identity: every request is anonymous. It
// is the default until an authentication adapter (e.g. OIDC) is configured, so
// generated handlers run out of the box — unsecured routes work, and a secured
// route is denied with 401 by a role-checking authorizer because no principal is
// present.
type AnonymousAuthenticator struct{}

// Authenticate always returns the anonymous principal.
func (AnonymousAuthenticator) Authenticate(context.Context, *http.Request) (Principal, error) {
	return Principal{}, nil
}

// Unauthenticated builds a 401 error for a missing or invalid credential. It
// satisfies CodedError and HTTPStatusError, so the error handler renders it as a
// 401 Problem with code "unauthenticated".
func Unauthenticated(format string, args ...any) *Error {
	return Errorf(http.StatusUnauthorized, "unauthenticated", format, args...)
}

// Forbidden builds a 403 error for an authenticated caller that lacks the
// required roles or scopes. It renders as a 403 Problem with code "forbidden".
func Forbidden(format string, args ...any) *Error {
	return Errorf(http.StatusForbidden, "forbidden", format, args...)
}

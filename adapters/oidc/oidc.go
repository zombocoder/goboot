// Package oidc authenticates goboot HTTP requests against an OpenID Connect
// provider — for example Keycloak — by validating the request's Bearer access
// token. It implements runtime.Authenticator: the provider's metadata and
// signing keys (JWKS) are discovered from the issuer URL and cached, each token
// is verified (signature, issuer, expiry, and optionally audience), and the
// claims are mapped to a runtime.Principal (subject, username, roles, scopes).
//
// Wire it into the composition root:
//
//	authn, err := oidc.New(ctx, oidc.Config{
//	    IssuerURL: "https://keycloak.example.com/realms/todo",
//	    Audience:  "todo-api",
//	    ClientID:  "todo-api", // also read Keycloak client roles
//	})
//	httpDeps.Authenticator = authn
//
// It is a separate module so the go-oidc dependency stays out of the core.
package oidc

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	coreoidc "github.com/coreos/go-oidc/v3/oidc"
	"github.com/zombocoder/goboot/runtime"
)

// Config configures the OIDC authenticator.
type Config struct {
	// IssuerURL is the token issuer, e.g.
	// https://keycloak.example.com/realms/todo. Provider metadata and the JWKS
	// are discovered from it.
	IssuerURL string
	// Audience, when set, is the expected `aud` claim (the API's client id or
	// resource identifier). Empty skips the audience check.
	Audience string
	// ClientID, when set, additionally reads Keycloak client roles from
	// resource_access[ClientID].roles into the principal.
	ClientID string
	// HTTPClient overrides the client used for discovery and JWKS fetches.
	HTTPClient *http.Client
	// SkipExpiryCheck disables the token-expiry check. For tests only.
	SkipExpiryCheck bool
}

// Authenticator is a runtime.Authenticator backed by an OIDC provider.
type Authenticator struct {
	verifier *coreoidc.IDTokenVerifier
	clientID string
}

// New builds an Authenticator, discovering the provider's metadata and JWKS from
// cfg.IssuerURL. It returns an error when discovery fails.
func New(ctx context.Context, cfg Config) (*Authenticator, error) {
	if cfg.IssuerURL == "" {
		return nil, fmt.Errorf("oidc: IssuerURL is required")
	}
	if cfg.HTTPClient != nil {
		ctx = coreoidc.ClientContext(ctx, cfg.HTTPClient)
	}
	provider, err := coreoidc.NewProvider(ctx, cfg.IssuerURL)
	if err != nil {
		return nil, fmt.Errorf("oidc: discovering provider %q: %w", cfg.IssuerURL, err)
	}
	vc := &coreoidc.Config{ClientID: cfg.Audience, SkipExpiryCheck: cfg.SkipExpiryCheck}
	if cfg.Audience == "" {
		vc.SkipClientIDCheck = true
	}
	return &Authenticator{verifier: provider.Verifier(vc), clientID: cfg.ClientID}, nil
}

// Authenticate validates the request's Bearer token and returns the caller
// principal. A request with no bearer token is anonymous (nil error), letting
// the authorizer decide; an invalid token is a 401.
func (a *Authenticator) Authenticate(ctx context.Context, r *http.Request) (runtime.Principal, error) {
	raw := bearerToken(r)
	if raw == "" {
		return runtime.Principal{}, nil
	}
	tok, err := a.verifier.Verify(ctx, raw)
	if err != nil {
		return runtime.Principal{}, runtime.Unauthenticated("invalid bearer token: %v", err)
	}
	var claims claimSet
	if err := tok.Claims(&claims); err != nil {
		return runtime.Principal{}, runtime.Unauthenticated("cannot parse token claims: %v", err)
	}
	_ = tok.Claims(&claims.raw) // best-effort: preserve all claims for the app
	return claims.principal(tok.Subject, a.clientID), nil
}

// bearerToken extracts the token from an `Authorization: Bearer <token>` header,
// or "" when absent.
func bearerToken(r *http.Request) string {
	const prefix = "Bearer "
	h := r.Header.Get("Authorization")
	if len(h) >= len(prefix) && strings.EqualFold(h[:len(prefix)], prefix) {
		return strings.TrimSpace(h[len(prefix):])
	}
	return ""
}

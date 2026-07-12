package oidc

import (
	"strings"

	"github.com/zombocoder/goboot/runtime"
)

// claimSet is the subset of OIDC / Keycloak claims mapped to a Principal. The
// raw map preserves every claim for application code that needs more.
type claimSet struct {
	Subject           string `json:"sub"`
	PreferredUsername string `json:"preferred_username"`
	Scope             string `json:"scope"`
	// RealmAccess holds Keycloak realm-wide roles.
	RealmAccess struct {
		Roles []string `json:"roles"`
	} `json:"realm_access"`
	// ResourceAccess holds Keycloak per-client roles, keyed by client id.
	ResourceAccess map[string]struct {
		Roles []string `json:"roles"`
	} `json:"resource_access"`

	raw map[string]any
}

// principal maps the claims to a runtime.Principal. subject is the verified
// token subject; clientID, when non-empty, pulls that client's roles from
// resource_access in addition to the realm roles.
func (c claimSet) principal(subject, clientID string) runtime.Principal {
	sub := subject
	if sub == "" {
		sub = c.Subject
	}
	roles := append([]string(nil), c.RealmAccess.Roles...)
	if clientID != "" {
		if ra, ok := c.ResourceAccess[clientID]; ok {
			roles = append(roles, ra.Roles...)
		}
	}
	return runtime.Principal{
		Subject:  sub,
		Username: c.PreferredUsername,
		Roles:    roles,
		Scopes:   strings.Fields(c.Scope),
		Claims:   c.raw,
	}
}

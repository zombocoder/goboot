package oidc

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
	"time"

	jose "github.com/go-jose/go-jose/v4"
	"github.com/zombocoder/goboot/runtime"
)

func TestBearerToken(t *testing.T) {
	cases := map[string]string{
		"":                   "",
		"Bearer abc.def.ghi": "abc.def.ghi",
		"bearer abc":         "abc", // case-insensitive scheme
		"Basic abc":          "",
		"Bearer   spaced":    "spaced",
	}
	for header, want := range cases {
		r := httptest.NewRequest("GET", "/", nil)
		if header != "" {
			r.Header.Set("Authorization", header)
		}
		if got := bearerToken(r); got != want {
			t.Errorf("bearerToken(%q) = %q, want %q", header, got, want)
		}
	}
}

func TestClaimSetPrincipal(t *testing.T) {
	c := claimSet{
		PreferredUsername: "alice",
		Scope:             "openid todos:read todos:write",
		raw:               map[string]any{"sub": "u1"},
	}
	c.RealmAccess.Roles = []string{"user"}
	c.ResourceAccess = map[string]struct {
		Roles []string `json:"roles"`
	}{"todo-api": {Roles: []string{"editor"}}}

	p := c.principal("u1", "todo-api")
	if p.Subject != "u1" || p.Username != "alice" {
		t.Errorf("principal identity = %+v", p)
	}
	if !p.HasRole("user") || !p.HasRole("editor") {
		t.Errorf("roles = %v, want realm + client roles", p.Roles)
	}
	if !p.HasScope("todos:read") || !p.HasScope("todos:write") {
		t.Errorf("scopes = %v", p.Scopes)
	}
	// Without a clientID, only realm roles are mapped.
	if p2 := c.principal("u1", ""); p2.HasRole("editor") {
		t.Errorf("client roles should not appear without a clientID: %v", p2.Roles)
	}
}

// mockProvider serves an OIDC discovery document and JWKS for priv's public key.
func mockProvider(t *testing.T, priv *rsa.PrivateKey) *httptest.Server {
	t.Helper()
	var issuer string
	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"issuer":                 issuer,
			"jwks_uri":               issuer + "/jwks",
			"authorization_endpoint": issuer + "/auth",
			"token_endpoint":         issuer + "/token",
		})
	})
	mux.HandleFunc("/jwks", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(jose.JSONWebKeySet{Keys: []jose.JSONWebKey{
			{Key: &priv.PublicKey, KeyID: "test-key", Algorithm: "RS256", Use: "sig"},
		}})
	})
	srv := httptest.NewServer(mux)
	issuer = srv.URL
	t.Cleanup(srv.Close)
	return srv
}

func signToken(t *testing.T, priv *rsa.PrivateKey, claims map[string]any) string {
	t.Helper()
	signer, err := jose.NewSigner(
		jose.SigningKey{Algorithm: jose.RS256, Key: priv},
		(&jose.SignerOptions{}).WithType("JWT").WithHeader("kid", "test-key"),
	)
	if err != nil {
		t.Fatal(err)
	}
	payload, err := json.Marshal(claims)
	if err != nil {
		t.Fatal(err)
	}
	jws, err := signer.Sign(payload)
	if err != nil {
		t.Fatal(err)
	}
	tok, err := jws.CompactSerialize()
	if err != nil {
		t.Fatal(err)
	}
	return tok
}

func authenticate(t *testing.T, srv *httptest.Server, cfg Config, token string) (runtime.Principal, error) {
	t.Helper()
	cfg.IssuerURL = srv.URL
	cfg.HTTPClient = srv.Client()
	authn, err := New(context.Background(), cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	r := httptest.NewRequest("GET", "/secret", nil)
	if token != "" {
		r.Header.Set("Authorization", "Bearer "+token)
	}
	return authn.Authenticate(context.Background(), r)
}

func TestAuthenticateValidToken(t *testing.T) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	srv := mockProvider(t, priv)
	now := time.Now()
	token := signToken(t, priv, map[string]any{
		"iss":                srv.URL,
		"sub":                "user-123",
		"aud":                []string{"todo-api"},
		"exp":                now.Add(time.Hour).Unix(),
		"iat":                now.Unix(),
		"preferred_username": "alice",
		"scope":              "openid todos:read",
		"realm_access":       map[string]any{"roles": []string{"user"}},
		"resource_access":    map[string]any{"todo-api": map[string]any{"roles": []string{"editor"}}},
	})

	p, err := authenticate(t, srv, Config{Audience: "todo-api", ClientID: "todo-api"}, token)
	if err != nil {
		t.Fatalf("Authenticate: %v", err)
	}
	if p.Subject != "user-123" || p.Username != "alice" {
		t.Errorf("principal = %+v", p)
	}
	if !p.HasRole("user") || !p.HasRole("editor") {
		t.Errorf("roles = %v", p.Roles)
	}
	if !slices.Contains(p.Scopes, "todos:read") {
		t.Errorf("scopes = %v", p.Scopes)
	}
	if p.Claims["preferred_username"] != "alice" {
		t.Errorf("raw claims not preserved: %v", p.Claims)
	}
}

func TestAuthenticateNoTokenIsAnonymous(t *testing.T) {
	priv, _ := rsa.GenerateKey(rand.Reader, 2048)
	srv := mockProvider(t, priv)
	p, err := authenticate(t, srv, Config{Audience: "todo-api"}, "")
	if err != nil {
		t.Fatalf("no-token should be nil error, got %v", err)
	}
	if p.IsAuthenticated() {
		t.Errorf("no token should yield an anonymous principal, got %+v", p)
	}
}

func TestAuthenticateRejectsBadTokens(t *testing.T) {
	priv, _ := rsa.GenerateKey(rand.Reader, 2048)
	srv := mockProvider(t, priv)
	now := time.Now()

	base := func() map[string]any {
		return map[string]any{
			"iss": srv.URL, "sub": "u1", "aud": []string{"todo-api"},
			"exp": now.Add(time.Hour).Unix(), "iat": now.Unix(),
		}
	}
	expired := base()
	expired["exp"] = now.Add(-time.Hour).Unix()
	wrongAud := base()
	wrongAud["aud"] = []string{"other-api"}

	cases := map[string]string{
		"garbage":        "not-a-jwt",
		"expired token":  signToken(t, priv, expired),
		"wrong audience": signToken(t, priv, wrongAud),
	}
	for name, token := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := authenticate(t, srv, Config{Audience: "todo-api"}, token)
			if err == nil {
				t.Fatal("expected an error for an invalid token")
			}
			if got := runtime.StatusOf(err); got != 401 {
				t.Errorf("status = %d, want 401 (err=%v)", got, err)
			}
		})
	}
}

// A token signed by a different key must be rejected (signature check).
func TestAuthenticateRejectsForeignSignature(t *testing.T) {
	priv, _ := rsa.GenerateKey(rand.Reader, 2048)
	other, _ := rsa.GenerateKey(rand.Reader, 2048)
	srv := mockProvider(t, priv) // JWKS advertises priv's public key
	now := time.Now()
	token := signToken(t, other, map[string]any{ // but signed with `other`
		"iss": srv.URL, "sub": "u1", "aud": []string{"todo-api"},
		"exp": now.Add(time.Hour).Unix(), "iat": now.Unix(),
	})
	if _, err := authenticate(t, srv, Config{Audience: "todo-api"}, token); runtime.StatusOf(err) != 401 {
		t.Errorf("foreign-signed token should 401, got %v", err)
	}
}

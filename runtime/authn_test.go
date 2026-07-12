package runtime

import (
	"context"
	"net/http/httptest"
	"testing"
)

func TestPrincipalContextRoundTrip(t *testing.T) {
	if _, ok := PrincipalFrom(context.Background()); ok {
		t.Error("empty context should carry no principal")
	}
	p := Principal{Subject: "u1", Username: "alice", Roles: []string{"user"}, Scopes: []string{"read"}}
	ctx := WithPrincipal(context.Background(), p)
	got, ok := PrincipalFrom(ctx)
	if !ok {
		t.Fatal("principal not found on context")
	}
	if got.Subject != "u1" || got.Username != "alice" {
		t.Errorf("round-trip principal = %+v", got)
	}
}

func TestPrincipalPredicates(t *testing.T) {
	if (Principal{}).IsAuthenticated() {
		t.Error("zero principal should be anonymous")
	}
	p := Principal{Subject: "u1", Roles: []string{"admin", "user"}, Scopes: []string{"read"}}
	if !p.IsAuthenticated() {
		t.Error("principal with subject should be authenticated")
	}
	if !p.HasRole("admin") || p.HasRole("root") {
		t.Error("HasRole mismatch")
	}
	if !p.HasScope("read") || p.HasScope("write") {
		t.Error("HasScope mismatch")
	}
}

func TestAnonymousAuthenticator(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	p, err := AnonymousAuthenticator{}.Authenticate(context.Background(), r)
	if err != nil {
		t.Fatalf("anonymous authenticate: %v", err)
	}
	if p.IsAuthenticated() {
		t.Error("anonymous authenticator should not authenticate")
	}
}

func TestAuthErrorHelpers(t *testing.T) {
	if got := StatusOf(Unauthenticated("nope")); got != 401 {
		t.Errorf("Unauthenticated status = %d, want 401", got)
	}
	if got := Unauthenticated("nope").Code(); got != "unauthenticated" {
		t.Errorf("Unauthenticated code = %q", got)
	}
	if got := StatusOf(Forbidden("nope")); got != 403 {
		t.Errorf("Forbidden status = %d, want 403", got)
	}
	if got := Forbidden("nope").Code(); got != "forbidden" {
		t.Errorf("Forbidden code = %q", got)
	}
}

func TestRoleAuthorizer(t *testing.T) {
	auth := RoleAuthorizer{}
	authed := WithPrincipal(context.Background(),
		Principal{Subject: "u1", Roles: []string{"user", "editor"}, Scopes: []string{"todos:read"}})
	anon := context.Background()

	tests := []struct {
		name   string
		ctx    context.Context
		req    AuthorizationRequest
		status int // 0 = allowed
	}{
		{"no restriction, anonymous", anon, AuthorizationRequest{}, 0},
		{"roles required, no principal", anon, AuthorizationRequest{Roles: []string{"user"}}, 401},
		{"roles required, anonymous principal", WithPrincipal(context.Background(), Principal{}),
			AuthorizationRequest{Roles: []string{"user"}}, 401},
		{"any role satisfied", authed, AuthorizationRequest{Roles: []string{"user", "admin"}}, 0},
		{"any role unsatisfied", authed, AuthorizationRequest{Roles: []string{"admin"}}, 403},
		{"all roles satisfied", authed,
			AuthorizationRequest{Roles: []string{"user", "editor"}, Mode: AuthorizationModeAll}, 0},
		{"all roles missing one", authed,
			AuthorizationRequest{Roles: []string{"user", "admin"}, Mode: AuthorizationModeAll}, 403},
		{"scope permission satisfied", authed, AuthorizationRequest{Permissions: []string{"todos:read"}}, 0},
		{"scope permission missing", authed, AuthorizationRequest{Permissions: []string{"todos:write"}}, 403},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := auth.Authorize(tc.ctx, tc.req)
			if tc.status == 0 {
				if err != nil {
					t.Errorf("expected allow, got %v", err)
				}
				return
			}
			if got := StatusOf(err); got != tc.status {
				t.Errorf("status = %d, want %d (err=%v)", got, tc.status, err)
			}
		})
	}
}

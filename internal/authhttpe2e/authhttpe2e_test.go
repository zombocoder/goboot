// Package authhttpe2e drives the generated HTTP handler for a route-level
// @Authorize to confirm the authenticate → authorize → controller pipeline:
// anonymous callers are denied 401, authenticated callers lacking the role are
// denied 403, and authenticated callers with the role reach the controller.
// wiring.gen.go is produced by the goboot generator from the authhttpapp example.
package authhttpe2e

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/zombocoder/goboot/runtime"
)

// stubAuthenticator returns a fixed principal (or error), standing in for an
// OIDC adapter.
type stubAuthenticator struct {
	principal runtime.Principal
	err       error
}

func (s stubAuthenticator) Authenticate(context.Context, *http.Request) (runtime.Principal, error) {
	return s.principal, s.err
}

func serve(t *testing.T, deps runtime.HTTPHandlerDependencies) *httptest.Server {
	t.Helper()
	comps, err := buildComponents()
	if err != nil {
		t.Fatalf("buildComponents: %v", err)
	}
	mux := http.NewServeMux()
	RegisterRoutes(mux, comps, deps)
	return httptest.NewServer(mux)
}

func TestSecuredRoute(t *testing.T) {
	tests := []struct {
		name string
		deps func() runtime.HTTPHandlerDependencies
		want int
	}{
		{
			name: "anonymous is unauthorized",
			deps: runtime.DefaultHTTPHandlerDependencies, // AnonymousAuthenticator + RoleAuthorizer
			want: http.StatusUnauthorized,                // 401
		},
		{
			name: "invalid credential is unauthorized",
			deps: func() runtime.HTTPHandlerDependencies {
				d := runtime.DefaultHTTPHandlerDependencies()
				d.Authenticator = stubAuthenticator{err: runtime.Unauthenticated("bad token")}
				return d
			},
			want: http.StatusUnauthorized, // 401
		},
		{
			name: "authenticated without role is forbidden",
			deps: func() runtime.HTTPHandlerDependencies {
				d := runtime.DefaultHTTPHandlerDependencies()
				d.Authenticator = stubAuthenticator{principal: runtime.Principal{Subject: "u1", Roles: []string{"guest"}}}
				return d
			},
			want: http.StatusForbidden, // 403
		},
		{
			name: "authenticated with role reaches controller",
			deps: func() runtime.HTTPHandlerDependencies {
				d := runtime.DefaultHTTPHandlerDependencies()
				d.Authenticator = stubAuthenticator{principal: runtime.Principal{Subject: "u1", Roles: []string{"user"}}}
				return d
			},
			want: http.StatusOK, // 200
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			srv := serve(t, tc.deps())
			defer srv.Close()
			resp, err := http.Get(srv.URL + "/secret")
			if err != nil {
				t.Fatalf("GET: %v", err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != tc.want {
				t.Errorf("status = %d, want %d", resp.StatusCode, tc.want)
			}
		})
	}
}

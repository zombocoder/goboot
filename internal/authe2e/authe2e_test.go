// Package authe2e drives generated @Authorize / @RolesAllowed proxies to confirm
// the target runs only when authorization succeeds. wiring.gen.go is produced by
// the goboot generator from the authapp example.
package authe2e

import (
	"context"
	"errors"
	"testing"

	goruntime "github.com/zombocoder/goboot/runtime"
)

// recordingAuthorizer allows or denies and records the requests it saw.
type recordingAuthorizer struct {
	deny     bool
	requests []goruntime.AuthorizationRequest
}

func (a *recordingAuthorizer) Authorize(_ context.Context, req goruntime.AuthorizationRequest) error {
	a.requests = append(a.requests, req)
	if a.deny {
		return goruntime.NewError(403, "forbidden", "denied")
	}
	return nil
}

func newComps(t *testing.T, authz goruntime.Authorizer) *Components {
	t.Helper()
	deps := goruntime.DefaultProxyDependencies()
	deps.Authorizer = authz
	comps, err := buildComponents(deps)
	if err != nil {
		t.Fatalf("buildComponents: %v", err)
	}
	return comps
}

func TestAuthorizedCallReachesTarget(t *testing.T) {
	authz := &recordingAuthorizer{deny: false}
	comps := newComps(t, authz)

	if err := comps.AdminServiceProxy.DeleteAll(context.Background()); err != nil {
		t.Fatalf("DeleteAll: %v", err)
	}
	if !comps.Admin.Deleted() {
		t.Error("target DeleteAll should have run when authorized")
	}
	// The proxy passed the declared role and mode.
	if len(authz.requests) != 1 || len(authz.requests[0].Roles) != 1 || authz.requests[0].Roles[0] != "admin" {
		t.Errorf("authorization request = %+v", authz.requests)
	}
	if authz.requests[0].Mode != goruntime.AuthorizationModeAll {
		t.Errorf("mode = %v, want all", authz.requests[0].Mode)
	}
}

func TestDeniedCallSkipsTarget(t *testing.T) {
	authz := &recordingAuthorizer{deny: true}
	comps := newComps(t, authz)

	err := comps.AdminServiceProxy.DeleteAll(context.Background())
	if err == nil {
		t.Fatal("DeleteAll should fail when authorization is denied")
	}
	if goruntime.StatusOf(err) != 403 {
		t.Errorf("denied status = %d, want 403", goruntime.StatusOf(err))
	}
	if comps.Admin.Deleted() {
		t.Error("target must NOT run when authorization is denied")
	}
}

func TestRolesAllowedShorthand(t *testing.T) {
	authz := &recordingAuthorizer{deny: false}
	comps := newComps(t, authz)

	got, err := comps.AdminServiceProxy.Read(context.Background())
	if err != nil || got != "data" {
		t.Fatalf("Read = %q, %v", got, err)
	}
	if len(authz.requests) != 1 || authz.requests[0].Roles[0] != "reader" {
		t.Errorf("@RolesAllowed should pass the reader role, got %+v", authz.requests)
	}
}

func TestDefaultAuthorizerPermits(t *testing.T) {
	// The default proxy dependencies permit all, so calls succeed.
	comps, err := buildComponents(goruntime.DefaultProxyDependencies())
	if err != nil {
		t.Fatal(err)
	}
	if err := comps.AdminServiceProxy.DeleteAll(context.Background()); err != nil {
		t.Errorf("default authorizer should permit: %v", err)
	}
	if !errors.Is(err, error(nil)) && err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

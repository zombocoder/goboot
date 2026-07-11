package compiler

import (
	"testing"

	"github.com/zombocoder/goboot/model"
)

func routeByPattern(app *model.Application, method, pattern string) *model.Route {
	for _, r := range app.Routes {
		if r.Method == method && r.Pattern == pattern {
			return r
		}
	}
	return nil
}

func TestDiscoverRoutes(t *testing.T) {
	res := analyzeApp(t, "./testdata/diapp/...")
	if errs := errorDiags(res.Diagnostics); len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if len(res.App.Controllers) != 1 {
		t.Fatalf("controllers = %d, want 1", len(res.App.Controllers))
	}
	if len(res.App.Routes) != 2 {
		t.Fatalf("routes = %d, want 2", len(res.App.Routes))
	}

	get := routeByPattern(res.App, "GET", "/api/v1/users/{id}")
	if get == nil {
		t.Fatal("GET /api/v1/users/{id} not found")
	}
	if get.HandlerName != "GetUser" || get.SuccessStatus != 200 {
		t.Errorf("GET route = %+v", get)
	}
	if !get.HasRequest() || !get.HasResponse() {
		t.Errorf("GET route should have request and response")
	}

	post := routeByPattern(res.App, "POST", "/api/v1/users")
	if post == nil {
		t.Fatal("POST /api/v1/users not found")
	}
	if post.SuccessStatus != 201 {
		t.Errorf("POST success status = %d, want 201", post.SuccessStatus)
	}
}

func TestDuplicateRouteRejected(t *testing.T) {
	res := analyzeApp(t, "./testdata/duproute")
	found := false
	for _, d := range res.Diagnostics {
		if d.Code == CodeDuplicateRoute {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected a duplicate-route diagnostic, got %v", res.Diagnostics)
	}
}

func TestInvalidHandlerRejected(t *testing.T) {
	res := analyzeApp(t, "./testdata/badhandler")
	found := false
	for _, d := range res.Diagnostics {
		if d.Code == CodeInvalidHandler {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected an invalid-handler diagnostic, got %v", res.Diagnostics)
	}
}

func TestJoinPath(t *testing.T) {
	cases := []struct{ base, sub, want string }{
		{"/api/v1/users", "/{id}", "/api/v1/users/{id}"},
		{"/api/v1/users", "", "/api/v1/users"},
		{"/api/v1/users", "{id}", "/api/v1/users/{id}"},
		{"", "/{id}", "/{id}"},
		{"", "", "/"},
		{"api", "x", "/api/x"},
	}
	for _, c := range cases {
		if got := joinPath(c.base, c.sub); got != c.want {
			t.Errorf("joinPath(%q,%q) = %q, want %q", c.base, c.sub, got, c.want)
		}
	}
}

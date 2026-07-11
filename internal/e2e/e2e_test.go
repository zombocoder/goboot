// Package e2e exercises the generated wiring end to end: it constructs the
// application's components, registers the generated HTTP handlers on a mux, and
// drives real requests through them, asserting on the responses. The wiring in
// wiring.gen.go is produced by the goboot generator from the diapp example.
package e2e

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/zombocoder/goboot/runtime"
)

// newServer builds the components, registers routes with the given
// dependencies, and returns a test server.
func newServer(t *testing.T, deps runtime.HTTPHandlerDependencies) *httptest.Server {
	t.Helper()
	components, err := buildComponents()
	if err != nil {
		t.Fatalf("buildComponents: %v", err)
	}
	mux := http.NewServeMux()
	RegisterRoutes(mux, components, deps)
	return httptest.NewServer(mux)
}

func TestGetEndpoint(t *testing.T) {
	srv := newServer(t, runtime.DefaultHTTPHandlerDependencies())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/v1/users/42")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var body struct {
		ID   string `json:"ID"`
		Name string `json:"Name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	// The path parameter must have been bound and flowed to the repository.
	if body.ID != "42" {
		t.Errorf("response ID = %q, want 42 (path binding failed)", body.ID)
	}
}

func TestPostEndpoint(t *testing.T) {
	srv := newServer(t, runtime.DefaultHTTPHandlerDependencies())
	defer srv.Close()

	resp, err := http.Post(srv.URL+"/api/v1/users", "application/json",
		strings.NewReader(`{"name":"Ada"}`))
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d, want 201", resp.StatusCode)
	}
	var body struct {
		Name string `json:"Name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Name != "Ada" {
		t.Errorf("response Name = %q, want Ada (body binding failed)", body.Name)
	}
}

// failingValidator rejects every request with a field error.
type failingValidator struct{}

func (failingValidator) Validate(context.Context, any) error {
	return runtime.NewValidationError(runtime.FieldError{
		Field: "id", Code: "required", Message: "id is required",
	})
}

func TestValidationFailureReturns400Problem(t *testing.T) {
	deps := runtime.DefaultHTTPHandlerDependencies()
	deps.Validator = failingValidator{}
	srv := newServer(t, deps)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/v1/users/42")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}
	var problem runtime.Problem
	if err := json.NewDecoder(resp.Body).Decode(&problem); err != nil {
		t.Fatalf("decode problem: %v", err)
	}
	if problem.Type != "validation_error" || len(problem.Errors) != 1 {
		t.Errorf("problem = %+v", problem)
	}
	if problem.Errors[0].Field != "id" {
		t.Errorf("field error = %+v", problem.Errors[0])
	}
}

// codedBinder fails binding with a mapped domain error, exercising the
// centralized error handler through the generated handler.
type codedBinder struct{}

func (codedBinder) Bind(context.Context, *http.Request, any) error {
	return runtime.NewError(http.StatusNotFound, "user_not_found", "no such user")
}

func TestMappedErrorReturnsProblem(t *testing.T) {
	deps := runtime.DefaultHTTPHandlerDependencies()
	deps.Binder = codedBinder{}
	srv := newServer(t, deps)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/v1/users/42")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", resp.StatusCode)
	}
	var problem runtime.Problem
	if err := json.NewDecoder(resp.Body).Decode(&problem); err != nil {
		t.Fatalf("decode problem: %v", err)
	}
	if problem.Code != "user_not_found" || problem.Status != http.StatusNotFound {
		t.Errorf("problem = %+v", problem)
	}
}

func TestUnknownRouteIs404(t *testing.T) {
	srv := newServer(t, runtime.DefaultHTTPHandlerDependencies())
	defer srv.Close()
	resp, err := http.Get(srv.URL + "/nope")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("unknown route status = %d, want 404", resp.StatusCode)
	}
}

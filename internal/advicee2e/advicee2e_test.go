// Package advicee2e drives the generated @ControllerAdvice / @ExceptionHandler
// dispatch end to end: a controller raises typed domain errors and the wiring
// routes each to the matching advice method. wiring.gen.go is produced by the
// goboot generator from the adviceapp example.
package advicee2e

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/zombocoder/goboot/runtime"
)

func newServer(t *testing.T) *httptest.Server {
	t.Helper()
	components, err := buildComponents()
	if err != nil {
		t.Fatalf("buildComponents: %v", err)
	}
	mux := http.NewServeMux()
	RegisterRoutes(mux, components, runtime.DefaultHTTPHandlerDependencies())
	return httptest.NewServer(mux)
}

func get(t *testing.T, url string) (int, string) {
	t.Helper()
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(b)
}

func TestResponseFormHandler(t *testing.T) {
	srv := newServer(t)
	defer srv.Close()

	// NotFoundError → HandleNotFound writes a 404 body (response form).
	status, body := get(t, srv.URL+"/things/widget?kind=missing")
	if status != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", status)
	}
	var eb struct {
		Message  string `json:"message"`
		Resource string `json:"resource"`
	}
	if err := json.Unmarshal([]byte(body), &eb); err != nil {
		t.Fatalf("decode: %v (%s)", err, body)
	}
	if eb.Resource != "widget" || eb.Message == "" {
		t.Errorf("body = %+v", eb)
	}
}

func TestTransformFormHandler(t *testing.T) {
	srv := newServer(t)
	defer srv.Close()

	// ConflictError → HandleConflict returns a coded 409 error the delegate
	// renders as a Problem.
	status, body := get(t, srv.URL+"/things/x?kind=conflict")
	if status != http.StatusConflict {
		t.Fatalf("status = %d, want 409 (%s)", status, body)
	}
	var problem struct {
		Code   string `json:"code"`
		Status int    `json:"status"`
	}
	if err := json.Unmarshal([]byte(body), &problem); err != nil {
		t.Fatalf("decode: %v (%s)", err, body)
	}
	if problem.Code != "conflict" || problem.Status != http.StatusConflict {
		t.Errorf("problem = %+v", problem)
	}
}

func TestCatchAllHandler(t *testing.T) {
	srv := newServer(t)
	defer srv.Close()

	// An unmapped error falls through every concrete handler to HandleAny, which
	// passes it to the delegate → a generic 500 Problem.
	status, _ := get(t, srv.URL+"/things/y?kind=other")
	if status != http.StatusInternalServerError {
		t.Fatalf("catch-all status = %d, want 500", status)
	}
}

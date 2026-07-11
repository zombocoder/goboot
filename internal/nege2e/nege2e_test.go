// Package nege2e drives the generated @Consumes / @Produces content negotiation
// end to end. wiring.gen.go is produced by the goboot generator from the negapp
// example.
package nege2e

import (
	"net/http"
	"net/http/httptest"
	"strings"
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

// req builds and sends a request with optional Content-Type / Accept headers.
func req(t *testing.T, method, url, contentType, accept, body string) int {
	t.Helper()
	var r *http.Request
	var err error
	if body != "" {
		r, err = http.NewRequest(method, url, strings.NewReader(body))
	} else {
		r, err = http.NewRequest(method, url, nil)
	}
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	if contentType != "" {
		r.Header.Set("Content-Type", contentType)
	}
	if accept != "" {
		r.Header.Set("Accept", accept)
	}
	resp, err := http.DefaultClient.Do(r)
	if err != nil {
		t.Fatalf("%s %s: %v", method, url, err)
	}
	defer resp.Body.Close()
	return resp.StatusCode
}

func TestConsumesRejectsWrongContentType(t *testing.T) {
	srv := newServer(t)
	defer srv.Close()

	// POST declares consumes=application/json; text/xml → 415.
	if s := req(t, http.MethodPost, srv.URL+"/docs", "text/xml", "", `<x/>`); s != http.StatusUnsupportedMediaType {
		t.Errorf("wrong Content-Type status = %d, want 415", s)
	}
	// Correct Content-Type + Accept → 201.
	if s := req(t, http.MethodPost, srv.URL+"/docs", "application/json", "application/json", `{"body":"hi"}`); s != http.StatusCreated {
		t.Errorf("valid create status = %d, want 201", s)
	}
}

func TestProducesRejectsUnacceptableAccept(t *testing.T) {
	srv := newServer(t)
	defer srv.Close()

	// GET declares produces=application/json; Accept text/html → 406.
	if s := req(t, http.MethodGet, srv.URL+"/docs/7", "", "text/html", ""); s != http.StatusNotAcceptable {
		t.Errorf("unacceptable Accept status = %d, want 406", s)
	}
	// A compatible Accept → 200.
	if s := req(t, http.MethodGet, srv.URL+"/docs/7", "", "application/json", ""); s != http.StatusOK {
		t.Errorf("acceptable Accept status = %d, want 200", s)
	}
	// No Accept header (client accepts anything) → 200.
	if s := req(t, http.MethodGet, srv.URL+"/docs/7", "", "", ""); s != http.StatusOK {
		t.Errorf("missing Accept status = %d, want 200", s)
	}
}

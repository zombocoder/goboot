// Package verbe2e drives the generated HTTP wiring for every supported verb
// (GET/POST/PUT/PATCH/DELETE), asserting the method routing and default status
// codes. wiring.gen.go is produced by the goboot generator from the verbapp
// example.
package verbe2e

import (
	"encoding/json"
	"io"
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

// do performs a request with the given method and returns status + body.
func do(t *testing.T, method, url, body string) (int, string) {
	t.Helper()
	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}
	req, err := http.NewRequest(method, url, reader)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, url, err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(b)
}

func TestPutReplaces(t *testing.T) {
	srv := newServer(t)
	defer srv.Close()

	status, body := do(t, http.MethodPut, srv.URL+"/widgets/7", `{"name":"updated"}`)
	if status != http.StatusOK {
		t.Fatalf("PUT status = %d, want 200", status)
	}
	var w struct{ ID, Name string }
	if err := json.Unmarshal([]byte(body), &w); err != nil {
		t.Fatalf("decode: %v (%s)", err, body)
	}
	if w.ID != "7" || w.Name != "updated" {
		t.Errorf("PUT body = %+v, want id=7 name=updated", w)
	}
}

func TestPatchUpdates(t *testing.T) {
	srv := newServer(t)
	defer srv.Close()

	status, body := do(t, http.MethodPatch, srv.URL+"/widgets/9", `{"name":"patched"}`)
	if status != http.StatusOK {
		t.Fatalf("PATCH status = %d, want 200", status)
	}
	if !strings.Contains(body, `"patched"`) || !strings.Contains(body, `"9"`) {
		t.Errorf("PATCH body = %s", body)
	}
}

func TestDeleteReturns204NoBody(t *testing.T) {
	srv := newServer(t)
	defer srv.Close()

	status, body := do(t, http.MethodDelete, srv.URL+"/widgets/3", "")
	if status != http.StatusNoContent {
		t.Fatalf("DELETE status = %d, want 204", status)
	}
	if body != "" {
		t.Errorf("DELETE should have no body, got %q", body)
	}
}

func TestMethodRoutingIsDistinct(t *testing.T) {
	srv := newServer(t)
	defer srv.Close()

	// GET and POST still work and are routed independently of the new verbs.
	if status, _ := do(t, http.MethodGet, srv.URL+"/widgets/1", ""); status != http.StatusOK {
		t.Errorf("GET status = %d, want 200", status)
	}
	if status, _ := do(t, http.MethodPost, srv.URL+"/widgets", `{"name":"x"}`); status != http.StatusCreated {
		t.Errorf("POST status = %d, want 201", status)
	}
	// A verb with no route on the collection path is rejected by the mux.
	if status, _ := do(t, http.MethodDelete, srv.URL+"/widgets", ""); status == http.StatusNoContent {
		t.Error("DELETE on the collection path should not match the /{id} route")
	}
}

package runtime

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDefaultBinderPathQueryHeader(t *testing.T) {
	type Req struct {
		ID     string   `path:"id"`
		Expand []string `query:"expand"`
		Limit  int      `query:"limit"`
		Locale string   `header:"Accept-Language"`
	}
	r := httptest.NewRequest(http.MethodGet, "/users/42?expand=a&expand=b&limit=10", nil)
	r.SetPathValue("id", "42")
	r.Header.Set("Accept-Language", "en-US")

	var req Req
	if err := (DefaultBinder{}).Bind(context.Background(), r, &req); err != nil {
		t.Fatalf("bind: %v", err)
	}
	if req.ID != "42" || req.Limit != 10 || req.Locale != "en-US" {
		t.Errorf("scalar binding wrong: %+v", req)
	}
	if len(req.Expand) != 2 || req.Expand[0] != "a" || req.Expand[1] != "b" {
		t.Errorf("slice binding wrong: %+v", req.Expand)
	}
}

func TestDefaultBinderJSONBody(t *testing.T) {
	type Req struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	}
	body := strings.NewReader(`{"name":"Ada","email":"ada@example.com"}`)
	r := httptest.NewRequest(http.MethodPost, "/users", body)
	r.Header.Set("Content-Type", "application/json")

	var req Req
	if err := (DefaultBinder{}).Bind(context.Background(), r, &req); err != nil {
		t.Fatalf("bind: %v", err)
	}
	if req.Name != "Ada" || req.Email != "ada@example.com" {
		t.Errorf("body binding wrong: %+v", req)
	}
}

func TestDefaultBinderInvalidJSON(t *testing.T) {
	type Req struct {
		Name string `json:"name"`
	}
	r := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader("{bad"))
	var req Req
	err := (DefaultBinder{}).Bind(context.Background(), r, &req)
	if err == nil {
		t.Fatal("expected an error for invalid JSON")
	}
	if StatusOf(err) != http.StatusBadRequest {
		t.Errorf("invalid JSON should map to 400, got %d", StatusOf(err))
	}
}

func TestDefaultBinderInvalidInteger(t *testing.T) {
	type Req struct {
		Limit int `query:"limit"`
	}
	r := httptest.NewRequest(http.MethodGet, "/x?limit=notanumber", nil)
	var req Req
	if err := (DefaultBinder{}).Bind(context.Background(), r, &req); err == nil {
		t.Fatal("expected error for invalid integer")
	}
}

func TestDefaultBinderRejectsNonPointer(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	if err := (DefaultBinder{}).Bind(context.Background(), r, struct{}{}); err == nil {
		t.Fatal("expected error binding to a non-pointer")
	}
}

func TestErrorHelpers(t *testing.T) {
	err := NewError(http.StatusNotFound, "user_not_found", "no such user")
	if StatusOf(err) != http.StatusNotFound {
		t.Errorf("StatusOf = %d", StatusOf(err))
	}
	if CodeOf(err) != "user_not_found" {
		t.Errorf("CodeOf = %q", CodeOf(err))
	}
	// A plain error maps to 500 with no code.
	if StatusOf(errPlain()) != http.StatusInternalServerError {
		t.Errorf("plain error should map to 500")
	}
	if CodeOf(errPlain()) != "" {
		t.Errorf("plain error should have no code")
	}
	// Wrapping preserves cause and status via errors.As.
	wrapped := NewError(http.StatusConflict, "conflict", "dup").Wrap(errPlain())
	if StatusOf(wrapped) != http.StatusConflict {
		t.Errorf("wrapped status = %d", StatusOf(wrapped))
	}
	if !strings.Contains(wrapped.Error(), "plain") {
		t.Errorf("wrapped error should include cause: %q", wrapped.Error())
	}
}

func errPlain() error { return &plainErr{} }

type plainErr struct{}

func (*plainErr) Error() string { return "plain failure" }

func TestToProblemValidation(t *testing.T) {
	ve := NewValidationError(FieldError{Field: "email", Code: "email", Message: "invalid"})
	p := ToProblem(ve)
	if p.Status != http.StatusBadRequest || p.Type != "validation_error" {
		t.Errorf("validation problem = %+v", p)
	}
	if len(p.Errors) != 1 || p.Errors[0].Field != "email" {
		t.Errorf("field errors not carried: %+v", p.Errors)
	}
}

func TestToProblemHidesInternalDetail(t *testing.T) {
	p := ToProblem(errPlain())
	if p.Status != http.StatusInternalServerError {
		t.Errorf("status = %d", p.Status)
	}
	if p.Detail != "" {
		t.Errorf("5xx detail should be withheld, got %q", p.Detail)
	}
	// A 4xx keeps its detail.
	p2 := ToProblem(NewError(http.StatusNotFound, "nf", "missing widget"))
	if p2.Detail != "missing widget" {
		t.Errorf("4xx detail = %q, want %q", p2.Detail, "missing widget")
	}
}

func TestDefaultErrorHandlerWritesProblem(t *testing.T) {
	rec := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	h := DefaultErrorHandler{Writer: JSONResponseWriter{}}
	h.Handle(context.Background(), rec, r, NewError(http.StatusNotFound, "user_not_found", "missing"))

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.Code)
	}
	var p Problem
	if err := json.Unmarshal(rec.Body.Bytes(), &p); err != nil {
		t.Fatalf("decoding problem: %v", err)
	}
	if p.Code != "user_not_found" || p.Status != http.StatusNotFound {
		t.Errorf("problem = %+v", p)
	}
}

func TestJSONResponseWriterNoContent(t *testing.T) {
	rec := httptest.NewRecorder()
	if err := (JSONResponseWriter{}).Write(context.Background(), rec, http.StatusNoContent, nil); err != nil {
		t.Fatalf("write: %v", err)
	}
	if rec.Code != http.StatusNoContent {
		t.Errorf("status = %d", rec.Code)
	}
	if rec.Body.Len() != 0 {
		t.Errorf("204 should have empty body, got %q", rec.Body.String())
	}
}

func TestRecover(t *testing.T) {
	rec := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	eh := DefaultErrorHandler{Writer: JSONResponseWriter{}}
	func() {
		defer Recover(context.Background(), rec, r, eh)
		panic("boom")
	}()
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("recovered status = %d, want 500", rec.Code)
	}
}

func TestDefaultBinderScalarKinds(t *testing.T) {
	type Req struct {
		Flag  bool    `query:"flag"`
		Count uint    `query:"count"`
		Ratio float64 `query:"ratio"`
		Token string  `cookie:"token"`
	}
	r := httptest.NewRequest(http.MethodGet, "/x?flag=true&count=7&ratio=1.5", nil)
	r.AddCookie(&http.Cookie{Name: "token", Value: "abc"})
	var req Req
	if err := (DefaultBinder{}).Bind(context.Background(), r, &req); err != nil {
		t.Fatalf("bind: %v", err)
	}
	if !req.Flag || req.Count != 7 || req.Ratio != 1.5 || req.Token != "abc" {
		t.Errorf("scalar kinds wrong: %+v", req)
	}
}

func TestExposeDetailForInternalError(t *testing.T) {
	rec := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	h := DefaultErrorHandler{Writer: JSONResponseWriter{}, ExposeDetail: true}
	h.Handle(context.Background(), rec, r, errPlain())
	var p Problem
	_ = json.Unmarshal(rec.Body.Bytes(), &p)
	if p.Detail != "plain failure" {
		t.Errorf("ExposeDetail should surface the message, got %q", p.Detail)
	}
}

func TestErrorfAndUnwrap(t *testing.T) {
	e := Errorf(http.StatusTeapot, "teapot", "short and %s", "stout")
	if e.Error() != "short and stout" || e.HTTPStatus() != http.StatusTeapot {
		t.Errorf("Errorf = %q status %d", e.Error(), e.HTTPStatus())
	}
	if e.Unwrap() != nil {
		t.Errorf("unwrapped cause should be nil")
	}
}

func TestDefaultDependencies(t *testing.T) {
	deps := DefaultHTTPHandlerDependencies()
	if deps.Binder == nil || deps.Validator == nil || deps.Authenticator == nil ||
		deps.Authorizer == nil || deps.ErrorHandler == nil || deps.ResponseWriter == nil ||
		deps.Observer == nil {
		t.Fatal("default dependencies must all be non-nil")
	}
	// The default authorizer permits a route with no role restriction.
	if err := deps.Authorizer.Authorize(context.Background(), AuthorizationRequest{}); err != nil {
		t.Errorf("default authorizer should allow an unrestricted route: %v", err)
	}
	// A secured route with no principal is denied (secure by default).
	if err := deps.Authorizer.Authorize(context.Background(), AuthorizationRequest{Roles: []string{"user"}}); StatusOf(err) != 401 {
		t.Errorf("default authorizer should 401 a secured route without a principal, got %v", err)
	}
	ctx, obs := deps.Observer.Begin(context.Background(), HTTPRequestOperation{})
	obs.End(200, nil)
	if ctx == nil {
		t.Errorf("observer should return a context")
	}
}

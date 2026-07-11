package runtime

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func request(contentType, accept string) *http.Request {
	r := httptest.NewRequest(http.MethodPost, "/", nil)
	if contentType != "" {
		r.Header.Set("Content-Type", contentType)
	}
	if accept != "" {
		r.Header.Set("Accept", accept)
	}
	return r
}

func TestConsumesAcceptsListedType(t *testing.T) {
	err := NegotiateContent(request("application/json; charset=utf-8", ""), []string{"application/json"}, nil)
	if err != nil {
		t.Errorf("json body should be consumable: %v", err)
	}
}

func TestConsumesRejectsUnlistedType(t *testing.T) {
	err := NegotiateContent(request("text/xml", ""), []string{"application/json"}, nil)
	if StatusOf(err) != http.StatusUnsupportedMediaType {
		t.Errorf("status = %d, want 415", StatusOf(err))
	}
}

func TestConsumesSkipsWhenNoContentType(t *testing.T) {
	// No Content-Type header → nothing to reject.
	if err := NegotiateContent(request("", ""), []string{"application/json"}, nil); err != nil {
		t.Errorf("missing Content-Type should pass: %v", err)
	}
}

func TestProducesMatchesAccept(t *testing.T) {
	if err := NegotiateContent(request("", "application/json"), nil, []string{"application/json"}); err != nil {
		t.Errorf("matching Accept should pass: %v", err)
	}
}

func TestProducesRejectsUnacceptable(t *testing.T) {
	err := NegotiateContent(request("", "text/html"), nil, []string{"application/json"})
	if StatusOf(err) != http.StatusNotAcceptable {
		t.Errorf("status = %d, want 406", StatusOf(err))
	}
}

func TestProducesWildcardAccept(t *testing.T) {
	for _, accept := range []string{"", "*/*", "application/*", "text/html, application/json;q=0.9"} {
		if err := NegotiateContent(request("", accept), nil, []string{"application/json"}); err != nil {
			t.Errorf("Accept %q should be acceptable: %v", accept, err)
		}
	}
}

func TestConsumesWildcardList(t *testing.T) {
	if err := NegotiateContent(request("image/png", ""), []string{"image/*"}, nil); err != nil {
		t.Errorf("image/* should accept image/png: %v", err)
	}
}

func TestNoConstraintsAlwaysPass(t *testing.T) {
	if err := NegotiateContent(request("application/xml", "text/plain"), nil, nil); err != nil {
		t.Errorf("no constraints should always pass: %v", err)
	}
}

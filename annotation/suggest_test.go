package annotation

import (
	"strings"
	"testing"
)

func TestClosest(t *testing.T) {
	cands := []string{"Service", "RestController", "GetMapping", "Cacheable"}
	tests := []struct {
		name   string
		want   string
		wantOK bool
	}{
		{"Serivce", "Service", true},      // transposition
		{"Cacheble", "Cacheable", true},   // deletion
		{"GetMaping", "GetMapping", true}, // deletion
		{"Zzzzzz", "", false},             // nothing close
		{"", "", false},                   // empty
	}
	for _, tc := range tests {
		got, ok := closest(tc.name, cands)
		if ok != tc.wantOK || (ok && got != tc.want) {
			t.Errorf("closest(%q) = (%q, %v), want (%q, %v)", tc.name, got, ok, tc.want, tc.wantOK)
		}
	}
}

func TestUnknownAnnotationSuggests(t *testing.T) {
	reg := DefaultRegistry()
	got := reg.Validate(Annotation{Name: "Cacheble"}, TargetMethod)
	if len(got) != 1 || !strings.Contains(got[0].Message, "did you mean @Cacheable?") {
		t.Errorf("expected a @Cacheable suggestion, got %q", messages(got))
	}
	// A name unlike anything registered gets no suggestion (no noise).
	distant := reg.Validate(Annotation{Name: "Zqxjw"}, TargetMethod)
	if len(distant) != 1 || strings.Contains(distant[0].Message, "did you mean") {
		t.Errorf("distant name should not suggest, got %q", messages(distant))
	}
}

func TestUnknownArgumentSuggests(t *testing.T) {
	reg := DefaultRegistry()
	// @RateLimit has a "limit" argument; "limt" should be suggested.
	ann := Annotation{
		Name:      "RateLimit",
		Arguments: map[string]Value{"limt": IntValue{Val: 5}},
	}
	got := reg.Validate(ann, TargetMethod)
	found := false
	for _, d := range got {
		if strings.Contains(d.Message, `unknown argument "limt"`) && strings.Contains(d.Message, "did you mean limit?") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected a 'limit' argument suggestion, got %q", messages(got))
	}
}

func messages(diags []*Diagnostic) []string {
	out := make([]string, len(diags))
	for i, d := range diags {
		out[i] = d.Message
	}
	return out
}

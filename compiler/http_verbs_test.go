package compiler

import "testing"

func TestHTTPVerbMappings(t *testing.T) {
	res := analyzeApp(t, "./testdata/verbapp")
	if errs := errorDiags(res.Diagnostics); len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}

	// Index routes by handler name.
	type ms struct {
		method string
		status int
	}
	got := map[string]ms{}
	for _, r := range res.App.Routes {
		got[r.HandlerName] = ms{r.Method, r.SuccessStatus}
	}

	want := map[string]ms{
		"GetWidget":     {"GET", 200},
		"CreateWidget":  {"POST", 201},
		"ReplaceWidget": {"PUT", 200},
		"PatchWidget":   {"PATCH", 200},
		"DeleteWidget":  {"DELETE", 204},
	}
	for handler, w := range want {
		g, ok := got[handler]
		if !ok {
			t.Errorf("no route discovered for %s", handler)
			continue
		}
		if g != w {
			t.Errorf("%s = %+v, want %+v", handler, g, w)
		}
	}
}

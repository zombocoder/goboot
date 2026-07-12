package compiler

import "testing"

// FuzzJoinPath asserts the route path-template builder never panics and always
// yields a non-empty pattern with a leading slash, for any base/sub path.
func FuzzJoinPath(f *testing.F) {
	for _, s := range [][2]string{
		{"/users", "{id}"}, {"", ""}, {"///", "///"},
		{"/a/", "/b/"}, {"a", ""}, {"", "{x}/{y}"}, {"/", "/"},
	} {
		f.Add(s[0], s[1])
	}
	f.Fuzz(func(t *testing.T, base, sub string) {
		got := joinPath(base, sub)
		if got == "" || got[0] != '/' {
			t.Errorf("joinPath(%q, %q) = %q, want a non-empty pattern with a leading slash", base, sub, got)
		}
	})
}

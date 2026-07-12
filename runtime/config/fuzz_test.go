package config

import (
	"strings"
	"testing"
)

// FuzzEnvName asserts the config-key → environment-variable transform never
// leaks a "." or "-" separator and always upper-cases, for any prefix/key.
func FuzzEnvName(f *testing.F) {
	for _, s := range [][2]string{
		{"app", "server.port"}, {"", ""}, {"a.b", "c-d"},
		{"-", "."}, {"x", "a.b.c"}, {"MY-APP", "db.pool-size"},
	} {
		f.Add(s[0], s[1])
	}
	f.Fuzz(func(t *testing.T, prefix, key string) {
		got := EnvName(prefix, key)
		if strings.ContainsAny(got, ".-") {
			t.Errorf("EnvName(%q, %q) = %q still contains a . or - separator", prefix, key, got)
		}
		if got != strings.ToUpper(got) {
			t.Errorf("EnvName(%q, %q) = %q is not upper-case", prefix, key, got)
		}
	})
}

// FuzzSplitList asserts the comma-list parser never yields a blank or untrimmed
// element, for any input.
func FuzzSplitList(f *testing.F) {
	for _, s := range []string{"a,b,c", "", " , ,x, ", ",,,", "  spaced  "} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, raw string) {
		for _, part := range splitList(raw) {
			if part == "" {
				t.Errorf("splitList(%q) produced a blank element", raw)
			}
			if strings.TrimSpace(part) != part {
				t.Errorf("splitList(%q) produced an untrimmed element %q", raw, part)
			}
		}
	})
}

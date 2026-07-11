package annotation

import (
	"testing"
)

// FuzzLexer asserts that the lexer terminates and never panics on arbitrary
// input, and that repeated next() calls eventually reach EOF. See §48.6.
func FuzzLexer(f *testing.F) {
	seeds := []string{
		"", "@Service", `@X(a="s", b=[1,2], c={k=true})`,
		"@Query(`sql`)", `@X(a=`, "@X(\x00)", `@X(a="\q")`,
		"@X(" + repeat("[", 100) + ")",
	}
	for _, s := range seeds {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, input string) {
		lx := newLexer(input)
		for i := 0; i < len(input)+16; i++ {
			if lx.next().Kind == tokEOF {
				return
			}
		}
		t.Fatalf("lexer did not reach EOF within bound for %q", input)
	})
}

// FuzzParseComment asserts that ParseComment never panics on arbitrary input
// and never returns a nil annotation entry. See §48.6 and the Milestone 1
// acceptance criterion "parser never panics on arbitrary input".
func FuzzParseComment(f *testing.F) {
	seeds := []string{
		"", "@Service", "not an annotation",
		`@Service(name="x", scope="singleton")`,
		"@Authorize(\n roles=[\"a\",\"b\"],\n mode=\"any\"\n)",
		"@Query(`\nSELECT 1\n`)",
		"@X(a=", "@X(((((", "@@@@", "@X(a={b={c=[1,[2,[3]]]}}})",
		"email me at foo@bar.com", "@1nvalid",
	}
	for _, s := range seeds {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, input string) {
		anns, diags := ParseComment(input, basePos())
		for _, a := range anns {
			if a.Name == "" {
				t.Fatalf("parsed annotation with empty name from %q", input)
			}
			if a.Arguments == nil {
				t.Fatalf("parsed annotation with nil Arguments map from %q", input)
			}
		}
		// Validation must also never panic on fuzzed annotations.
		reg := DefaultRegistry()
		for _, a := range anns {
			_ = reg.Validate(a, TargetStruct)
		}
		_ = diags
	})
}

func repeat(s string, n int) string {
	out := make([]byte, 0, len(s)*n)
	for range n {
		out = append(out, s...)
	}
	return string(out)
}

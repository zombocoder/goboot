package annotation

import "testing"

// lexAll drains the lexer into a slice of tokens (excluding the final EOF).
func lexAll(input string) []lexToken {
	lx := newLexer(input)
	var toks []lexToken
	for {
		tok := lx.next()
		if tok.Kind == tokEOF {
			return toks
		}
		toks = append(toks, tok)
	}
}

func TestLexStringEscapes(t *testing.T) {
	toks := lexAll(`"a\n\t\r\"\\b"`)
	if len(toks) != 1 || toks[0].Kind != tokString {
		t.Fatalf("got %v", toks)
	}
	if want := "a\n\t\r\"\\b"; toks[0].Text != want {
		t.Fatalf("decoded = %q, want %q", toks[0].Text, want)
	}
}

func TestLexErrorTokens(t *testing.T) {
	cases := []string{
		`"unterminated`,     // no closing quote
		"\"line\nbreak\"",   // newline in double-quoted string
		`"bad\qescape"`,     // invalid escape
		"`unterminated raw", // no closing backtick
		"1.2.3",             // malformed number (two dots)
		"#",                 // unexpected character
		`"trailing\`,        // unterminated escape at EOF
	}
	for _, c := range cases {
		toks := lexAll(c)
		found := false
		for _, tk := range toks {
			if tk.Kind == tokError {
				found = true
			}
		}
		if !found {
			t.Errorf("expected an error token for %q, got %v", c, toks)
		}
	}
}

func TestLexNumbers(t *testing.T) {
	cases := []struct {
		in   string
		kind tokenKind
	}{
		{"42", tokInt},
		{"-42", tokInt},
		{"+42", tokInt},
		{"3.14", tokFloat},
		{"1e10", tokFloat},
		{"2.5e-3", tokFloat},
	}
	for _, c := range cases {
		toks := lexAll(c.in)
		if len(toks) != 1 || toks[0].Kind != c.kind {
			t.Errorf("lex(%q) = %v, want kind %v", c.in, toks, c.kind)
		}
	}
}

func TestLexBareSignIsError(t *testing.T) {
	toks := lexAll("-")
	if len(toks) != 1 || toks[0].Kind != tokError {
		t.Fatalf("lex(%q) = %v, want error", "-", toks)
	}
}

func TestTokenKindString(t *testing.T) {
	// Exercise the diagnostic-facing token names.
	kinds := []tokenKind{
		tokEOF, tokAt, tokIdent, tokLParen, tokRParen, tokLBracket,
		tokRBracket, tokLBrace, tokRBrace, tokEquals, tokComma,
		tokString, tokInt, tokFloat, tokError, tokenKind(99),
	}
	for _, k := range kinds {
		if k.String() == "" {
			t.Errorf("tokenKind(%d).String() empty", k)
		}
	}
}

func TestParseArrayAndObjectErrors(t *testing.T) {
	cases := []string{
		`@X(a=[1 2])`,     // missing comma in array
		`@X(a={k=1 j=2})`, // missing comma in object
		`@X(a={1=2})`,     // non-identifier object key
		`@X(a={k 1})`,     // missing '=' in object
		`@X(a=[1,)`,       // unterminated array
		`@X(a={k=1,)`,     // unterminated object
	}
	for _, c := range cases {
		_, diags := ParseComment(c, basePos())
		if len(diags) == 0 {
			t.Errorf("expected diagnostic for %q", c)
		}
	}
}

func TestParseNestedTooDeep(t *testing.T) {
	// Build an array nested well beyond maxNestingDepth.
	deep := "@X(a=" + repeat("[", 40) + repeat("]", 40) + ")"
	_, diags := ParseComment(deep, basePos())
	if len(diags) == 0 {
		t.Fatalf("expected a nesting-depth diagnostic")
	}
}

func TestArgumentTypeStringNames(t *testing.T) {
	names := map[ArgumentType]string{
		ArgString: "string", ArgInteger: "integer", ArgFloat: "float",
		ArgBoolean: "boolean", ArgIdentifier: "identifier",
		ArgStringOrIdent: "string or identifier", ArgObject: "object", ArgAny: "any",
	}
	for at, want := range names {
		if at.String() != want {
			t.Errorf("ArgumentType(%d) = %q, want %q", at, at.String(), want)
		}
	}
}

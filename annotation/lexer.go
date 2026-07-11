package annotation

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// lexer tokenizes the annotation argument language. It operates over a single
// annotation buffer (for example `@Service(name="x")`) and produces tokens on
// demand via next. The lexer never panics: malformed input yields a tokError
// token that the parser converts into a source-aware diagnostic.
type lexer struct {
	input string
	pos   int // byte offset of the next unread rune
}

func newLexer(input string) *lexer {
	return &lexer{input: input}
}

// next scans and returns the next token.
func (l *lexer) next() lexToken {
	l.skipSpace()
	if l.pos >= len(l.input) {
		return lexToken{Kind: tokEOF, Offset: l.pos}
	}

	start := l.pos
	r, size := utf8.DecodeRuneInString(l.input[l.pos:])

	switch r {
	case '@':
		l.pos += size
		return lexToken{Kind: tokAt, Text: "@", Offset: start}
	case '(':
		l.pos += size
		return lexToken{Kind: tokLParen, Text: "(", Offset: start}
	case ')':
		l.pos += size
		return lexToken{Kind: tokRParen, Text: ")", Offset: start}
	case '[':
		l.pos += size
		return lexToken{Kind: tokLBracket, Text: "[", Offset: start}
	case ']':
		l.pos += size
		return lexToken{Kind: tokRBracket, Text: "]", Offset: start}
	case '{':
		l.pos += size
		return lexToken{Kind: tokLBrace, Text: "{", Offset: start}
	case '}':
		l.pos += size
		return lexToken{Kind: tokRBrace, Text: "}", Offset: start}
	case '=':
		l.pos += size
		return lexToken{Kind: tokEquals, Text: "=", Offset: start}
	case ',':
		l.pos += size
		return lexToken{Kind: tokComma, Text: ",", Offset: start}
	case '"':
		return l.lexQuotedString(start)
	case '`':
		return l.lexRawString(start)
	}

	if r == '-' || r == '+' || (r >= '0' && r <= '9') {
		return l.lexNumber(start)
	}
	if isIdentStart(r) {
		return l.lexIdent(start)
	}

	l.pos += size
	return lexToken{Kind: tokError, Text: "unexpected character " + string(r), Offset: start}
}

func (l *lexer) skipSpace() {
	for l.pos < len(l.input) {
		r, size := utf8.DecodeRuneInString(l.input[l.pos:])
		if !unicode.IsSpace(r) {
			return
		}
		l.pos += size
	}
}

// lexQuotedString scans a double-quoted string with Go-style escapes. An
// unterminated or badly escaped literal produces a tokError.
func (l *lexer) lexQuotedString(start int) lexToken {
	l.pos++ // consume opening quote
	var b strings.Builder
	for l.pos < len(l.input) {
		r, size := utf8.DecodeRuneInString(l.input[l.pos:])
		switch r {
		case '"':
			l.pos += size
			return lexToken{Kind: tokString, Text: b.String(), Offset: start}
		case '\n':
			return lexToken{Kind: tokError, Text: "unterminated string", Offset: start}
		case '\\':
			l.pos += size
			if l.pos >= len(l.input) {
				return lexToken{Kind: tokError, Text: "unterminated escape sequence", Offset: start}
			}
			esc, esize := utf8.DecodeRuneInString(l.input[l.pos:])
			switch esc {
			case 'n':
				b.WriteByte('\n')
			case 't':
				b.WriteByte('\t')
			case 'r':
				b.WriteByte('\r')
			case '"':
				b.WriteByte('"')
			case '\\':
				b.WriteByte('\\')
			default:
				return lexToken{Kind: tokError, Text: "invalid escape sequence \\" + string(esc), Offset: start}
			}
			l.pos += esize
		default:
			b.WriteRune(r)
			l.pos += size
		}
	}
	return lexToken{Kind: tokError, Text: "unterminated string", Offset: start}
}

// lexRawString scans a backtick-delimited raw string. Raw strings may span
// multiple lines and perform no escape processing; they carry SQL and other
// multi-line payloads.
func (l *lexer) lexRawString(start int) lexToken {
	l.pos++ // consume opening backtick
	contentStart := l.pos
	for l.pos < len(l.input) {
		if l.input[l.pos] == '`' {
			content := l.input[contentStart:l.pos]
			l.pos++ // consume closing backtick
			return lexToken{Kind: tokString, Text: content, Offset: start, Raw: true}
		}
		l.pos++
	}
	return lexToken{Kind: tokError, Text: "unterminated raw string", Offset: start}
}

// lexNumber scans an integer or float literal with an optional sign.
func (l *lexer) lexNumber(start int) lexToken {
	if l.input[l.pos] == '-' || l.input[l.pos] == '+' {
		l.pos++
	}
	isFloat := false
	for l.pos < len(l.input) {
		c := l.input[l.pos]
		switch {
		case c >= '0' && c <= '9':
			l.pos++
		case c == '.':
			if isFloat {
				return lexToken{Kind: tokError, Text: "malformed number", Offset: start}
			}
			isFloat = true
			l.pos++
		case c == 'e' || c == 'E':
			isFloat = true
			l.pos++
			if l.pos < len(l.input) && (l.input[l.pos] == '-' || l.input[l.pos] == '+') {
				l.pos++
			}
		default:
			goto done
		}
	}
done:
	text := l.input[start:l.pos]
	// Reject a bare sign or dot with no digits.
	if strings.IndexFunc(text, func(r rune) bool { return r >= '0' && r <= '9' }) < 0 {
		return lexToken{Kind: tokError, Text: "malformed number " + text, Offset: start}
	}
	if isFloat {
		return lexToken{Kind: tokFloat, Text: text, Offset: start}
	}
	return lexToken{Kind: tokInt, Text: text, Offset: start}
}

// lexIdent scans an identifier or keyword. Dots are permitted so that
// qualified names such as domain.UserNotFoundError can appear unquoted.
func (l *lexer) lexIdent(start int) lexToken {
	for l.pos < len(l.input) {
		r, size := utf8.DecodeRuneInString(l.input[l.pos:])
		if !isIdentPart(r) {
			break
		}
		l.pos += size
	}
	return lexToken{Kind: tokIdent, Text: l.input[start:l.pos], Offset: start}
}

func isIdentStart(r rune) bool {
	return r == '_' || unicode.IsLetter(r)
}

func isIdentPart(r rune) bool {
	return r == '_' || r == '.' || unicode.IsLetter(r) || unicode.IsDigit(r)
}

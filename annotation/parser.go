package annotation

import (
	"go/token"
	"strconv"
	"strings"
	"unicode/utf8"
)

// PositionalKey is the argument-map key under which a single positional
// argument is stored, e.g. the "2s" in @Timeout("2s") or the SQL in @Query(`...`).
// Additional positional arguments are stored under PositionalKey followed by
// their index ("value1", "value2", ...).
const PositionalKey = "value"

// maxNestingDepth bounds recursion into nested arrays and objects. It exists
// purely to keep the parser from overflowing the stack on adversarial input;
// no legitimate annotation approaches it.
const maxNestingDepth = 32

// Annotation is a single parsed annotation attached to a Go declaration.
//
// See specification §9.3. Position points at the '@' that begins the
// annotation. Raw holds the exact source text the annotation was parsed from.
type Annotation struct {
	Name      string
	Arguments map[string]Value
	Position  token.Position
	Raw       string
}

// Arg returns the named argument and true, or nil and false if absent.
func (a Annotation) Arg(name string) (Value, bool) {
	v, ok := a.Arguments[name]
	return v, ok
}

// Positional returns the single positional argument and true, or nil and false.
func (a Annotation) Positional() (Value, bool) {
	return a.Arg(PositionalKey)
}

// ParseComment parses every annotation found in the cleaned text of a comment
// group. base is the position of text's first byte (typically the position of
// the comment group). It returns the annotations in source order together with
// any diagnostics produced by malformed annotations. Well-formed annotations
// are still returned even when siblings fail to parse.
//
// ParseComment never panics, regardless of input.
func ParseComment(text string, base token.Position) ([]Annotation, []*Diagnostic) {
	pos := positioner{base: base, text: text}
	var (
		annotations []Annotation
		diagnostics []*Diagnostic
	)

	lines := splitLines(text)
	for i := 0; i < len(lines); i++ {
		ln := lines[i]
		if !isAnnotationStart(ln.text) {
			continue
		}

		// Accumulate lines until the annotation's delimiters balance, so that
		// multi-line annotations (§9.1) are collected as a single buffer.
		end := i
		var st scanState
		for {
			st.feed(lines[end].text)
			if st.complete() || end == len(lines)-1 {
				break
			}
			end++
		}

		bufStart := ln.leadingOffset() // offset of '@' within text
		bufEnd := lines[end].offset + len(lines[end].text)
		buf := text[bufStart:bufEnd]

		ann, diag := parseOne(buf, bufStart, pos)
		if diag != nil {
			diagnostics = append(diagnostics, diag)
		}
		if ann != nil {
			annotations = append(annotations, *ann)
		}
		i = end
	}

	return annotations, diagnostics
}

// line is a single physical line of comment text with its absolute byte offset.
type line struct {
	text   string
	offset int // byte offset of text[0] within the full comment text
}

// leadingOffset returns the absolute offset of the first non-space rune, i.e.
// the position of the '@' for an annotation start.
func (l line) leadingOffset() int {
	trimmed := strings.TrimLeftFunc(l.text, func(r rune) bool { return r == ' ' || r == '\t' })
	return l.offset + (len(l.text) - len(trimmed))
}

func splitLines(text string) []line {
	var lines []line
	offset := 0
	for offset <= len(text) {
		nl := strings.IndexByte(text[offset:], '\n')
		if nl < 0 {
			lines = append(lines, line{text: text[offset:], offset: offset})
			break
		}
		lines = append(lines, line{text: text[offset : offset+nl], offset: offset})
		offset += nl + 1
	}
	return lines
}

// isAnnotationStart reports whether a line begins (after leading whitespace)
// with '@' followed by an identifier-start rune. This distinguishes annotations
// from stray '@' characters in prose and from documentation comments (§37.4).
func isAnnotationStart(s string) bool {
	s = strings.TrimLeft(s, " \t")
	if !strings.HasPrefix(s, "@") {
		return false
	}
	r, _ := utf8.DecodeRuneInString(s[1:])
	return isIdentStart(r)
}

// scanState tracks delimiter and string state while accumulating the lines of a
// possibly multi-line annotation. It mirrors the lexer's string rules so that
// parentheses inside string literals do not affect balance.
type scanState struct {
	depth    int  // net paren/bracket/brace depth
	opened   bool // whether any delimiter has been opened
	inString bool // inside a double-quoted string
	inRaw    bool // inside a backtick raw string (may span lines)
	escaped  bool // previous rune in a string was a backslash
}

// feed consumes one physical line, updating delimiter and string state.
func (s *scanState) feed(text string) {
	for _, r := range text {
		switch {
		case s.inRaw:
			if r == '`' {
				s.inRaw = false
			}
		case s.inString:
			if s.escaped {
				s.escaped = false
			} else if r == '\\' {
				s.escaped = true
			} else if r == '"' {
				s.inString = false
			}
		default:
			switch r {
			case '"':
				s.inString = true
			case '`':
				s.inRaw = true
			case '(', '[', '{':
				s.opened = true
				s.depth++
			case ')', ']', '}':
				s.depth--
			}
		}
	}
	// A double-quoted string cannot span physical lines; reset at line end.
	s.inString = false
	s.escaped = false
}

// complete reports whether the accumulated annotation is finished: either it
// opened delimiters that are now balanced, or it never opened any (a marker).
func (s *scanState) complete() bool {
	if s.inRaw {
		return false
	}
	if !s.opened {
		return true
	}
	return s.depth <= 0
}

// parseOne parses a single annotation from buf. bufStart is buf's absolute
// offset within the full comment text, used to map token offsets to positions.
func parseOne(buf string, bufStart int, pos positioner) (*Annotation, *Diagnostic) {
	p := &parser{
		stream:   &tokenStream{lex: newLexer(buf)},
		bufStart: bufStart,
		pos:      pos,
	}
	return p.parseAnnotation(strings.TrimSpace(buf))
}

// parser drives the token stream to build one Annotation.
type parser struct {
	stream   *tokenStream
	bufStart int
	pos      positioner
}

// posOf maps a token's buffer offset to an absolute source position.
func (p *parser) posOf(tok lexToken) token.Position {
	return p.pos.at(p.bufStart + tok.Offset)
}

func (p *parser) parseAnnotation(raw string) (*Annotation, *Diagnostic) {
	at := p.stream.next()
	if at.Kind != tokAt {
		return nil, newError(CodeSyntax, p.posOf(at), "expected '@' to start annotation")
	}
	nameTok := p.stream.next()
	if nameTok.Kind != tokIdent {
		return nil, newError(CodeSyntax, p.posOf(nameTok), "expected annotation name after '@'")
	}

	ann := &Annotation{
		Name:      nameTok.Text,
		Arguments: map[string]Value{},
		Position:  p.posOf(at),
		Raw:       raw,
	}

	if p.stream.peek().Kind == tokLParen {
		if diag := p.parseArguments(ann); diag != nil {
			return nil, diag
		}
	}

	// An annotation must be the entire comment content (§37.4): a bare marker,
	// or a name plus its balanced argument list, and nothing else. Trailing
	// prose means this @Name is a documentation mention — for example a comment
	// that wraps so "the @Transactional method ..." starts a line — not an
	// annotation. Skip it silently rather than emitting a spurious marker.
	if p.stream.peek().Kind != tokEOF {
		return nil, nil
	}
	return ann, nil
}

func (p *parser) parseArguments(ann *Annotation) *Diagnostic {
	p.stream.next() // consume '('
	positional := 0

	if p.stream.peek().Kind == tokRParen {
		p.stream.next()
		return nil
	}

	for {
		// Distinguish "name = value" from a bare positional value.
		first := p.stream.next()
		if first.Kind == tokError {
			return newError(CodeSyntax, p.posOf(first), "%s in annotation @%s", first.Text, ann.Name)
		}
		if first.Kind == tokIdent && p.stream.peek().Kind == tokEquals {
			p.stream.next() // consume '='
			valTok := p.stream.next()
			val, diag := p.parseValue(valTok, ann.Name, 0)
			if diag != nil {
				return diag
			}
			if _, exists := ann.Arguments[first.Text]; exists {
				return newError(CodeDuplicateArgument, p.posOf(first),
					"duplicate argument %q in annotation @%s", first.Text, ann.Name)
			}
			ann.Arguments[first.Text] = val
		} else {
			val, diag := p.parseValue(first, ann.Name, 0)
			if diag != nil {
				return diag
			}
			key := PositionalKey
			if positional > 0 {
				key = PositionalKey + strconv.Itoa(positional)
			}
			ann.Arguments[key] = val
			positional++
		}

		sep := p.stream.next()
		switch sep.Kind {
		case tokComma:
			// Tolerate a trailing comma before the closing paren.
			if p.stream.peek().Kind == tokRParen {
				p.stream.next()
				return nil
			}
			continue
		case tokRParen:
			return nil
		case tokError:
			return newError(CodeSyntax, p.posOf(sep), "%s in annotation @%s", sep.Text, ann.Name)
		default:
			return newError(CodeSyntax, p.posOf(sep),
				"expected ',' or ')' in annotation @%s, found %s", ann.Name, sep.Kind)
		}
	}
}

// parseValue parses a value beginning with tok. depth guards nesting.
func (p *parser) parseValue(tok lexToken, annName string, depth int) (Value, *Diagnostic) {
	if depth > maxNestingDepth {
		return nil, newError(CodeSyntax, p.posOf(tok), "annotation @%s nested too deeply", annName)
	}
	switch tok.Kind {
	case tokString:
		return StringValue{Val: tok.Text, Raw: tok.Raw}, nil
	case tokInt:
		n, err := strconv.ParseInt(tok.Text, 10, 64)
		if err != nil {
			return nil, newError(CodeSyntax, p.posOf(tok), "invalid integer %q", tok.Text)
		}
		return IntValue{Val: n}, nil
	case tokFloat:
		f, err := strconv.ParseFloat(tok.Text, 64)
		if err != nil {
			return nil, newError(CodeSyntax, p.posOf(tok), "invalid float %q", tok.Text)
		}
		return FloatValue{Val: f}, nil
	case tokIdent:
		switch tok.Text {
		case "true":
			return BoolValue{Val: true}, nil
		case "false":
			return BoolValue{Val: false}, nil
		case "null":
			return NullValue{}, nil
		default:
			return IdentValue{Name: tok.Text}, nil
		}
	case tokLBracket:
		return p.parseArray(annName, depth)
	case tokLBrace:
		return p.parseObject(annName, depth)
	case tokError:
		return nil, newError(CodeSyntax, p.posOf(tok), "%s in annotation @%s", tok.Text, annName)
	default:
		return nil, newError(CodeSyntax, p.posOf(tok),
			"unexpected %s where a value was expected in annotation @%s", tok.Kind, annName)
	}
}

func (p *parser) parseArray(annName string, depth int) (Value, *Diagnostic) {
	arr := ArrayValue{}
	if p.stream.peek().Kind == tokRBracket {
		p.stream.next()
		return arr, nil
	}
	for {
		elemTok := p.stream.next()
		elem, diag := p.parseValue(elemTok, annName, depth+1)
		if diag != nil {
			return nil, diag
		}
		arr.Elements = append(arr.Elements, elem)

		sep := p.stream.next()
		switch sep.Kind {
		case tokComma:
			if p.stream.peek().Kind == tokRBracket {
				p.stream.next()
				return arr, nil
			}
			continue
		case tokRBracket:
			return arr, nil
		default:
			return nil, newError(CodeSyntax, p.posOf(sep),
				"expected ',' or ']' in array in annotation @%s, found %s", annName, sep.Kind)
		}
	}
}

func (p *parser) parseObject(annName string, depth int) (Value, *Diagnostic) {
	obj := ObjectValue{Fields: map[string]Value{}}
	if p.stream.peek().Kind == tokRBrace {
		p.stream.next()
		return obj, nil
	}
	for {
		keyTok := p.stream.next()
		if keyTok.Kind != tokIdent {
			return nil, newError(CodeSyntax, p.posOf(keyTok),
				"expected field name in object in annotation @%s, found %s", annName, keyTok.Kind)
		}
		if eq := p.stream.next(); eq.Kind != tokEquals {
			return nil, newError(CodeSyntax, p.posOf(eq),
				"expected '=' after field %q in annotation @%s", keyTok.Text, annName)
		}
		valTok := p.stream.next()
		val, diag := p.parseValue(valTok, annName, depth+1)
		if diag != nil {
			return nil, diag
		}
		if _, exists := obj.Fields[keyTok.Text]; exists {
			return nil, newError(CodeDuplicateArgument, p.posOf(keyTok),
				"duplicate field %q in object in annotation @%s", keyTok.Text, annName)
		}
		obj.Fields[keyTok.Text] = val

		sep := p.stream.next()
		switch sep.Kind {
		case tokComma:
			if p.stream.peek().Kind == tokRBrace {
				p.stream.next()
				return obj, nil
			}
			continue
		case tokRBrace:
			return obj, nil
		default:
			return nil, newError(CodeSyntax, p.posOf(sep),
				"expected ',' or '}' in object in annotation @%s, found %s", annName, sep.Kind)
		}
	}
}

// tokenStream wraps the lexer with one-token lookahead.
type tokenStream struct {
	lex    *lexer
	peeked *lexToken
}

func (s *tokenStream) next() lexToken {
	if s.peeked != nil {
		t := *s.peeked
		s.peeked = nil
		return t
	}
	return s.lex.next()
}

func (s *tokenStream) peek() lexToken {
	if s.peeked == nil {
		t := s.lex.next()
		s.peeked = &t
	}
	return *s.peeked
}

// positioner maps byte offsets within a comment's text to source positions,
// relative to the position of the text's first byte.
type positioner struct {
	base token.Position
	text string
}

func (p positioner) at(offset int) token.Position {
	if offset < 0 {
		offset = 0
	}
	if offset > len(p.text) {
		offset = len(p.text)
	}
	newlines := strings.Count(p.text[:offset], "\n")
	pos := token.Position{
		Filename: p.base.Filename,
		Offset:   p.base.Offset + offset,
		Line:     p.base.Line + newlines,
	}
	lineStart := strings.LastIndexByte(p.text[:offset], '\n') + 1
	if newlines == 0 {
		// Same line as base: columns accumulate from base.Column.
		pos.Column = p.base.Column + offset
	} else {
		pos.Column = offset - lineStart + 1
	}
	return pos
}

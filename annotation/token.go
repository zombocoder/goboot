package annotation

// tokenKind enumerates lexical tokens of the annotation argument language.
type tokenKind int

const (
	tokEOF      tokenKind = iota
	tokAt                 // @
	tokIdent              // identifier or keyword (true/false/null resolved by parser)
	tokLParen             // (
	tokRParen             // )
	tokLBracket           // [
	tokRBracket           // ]
	tokLBrace             // {
	tokRBrace             // }
	tokEquals             // =
	tokComma              // ,
	tokString             // "..." or `...` (Text holds decoded content)
	tokInt                // integer literal
	tokFloat              // floating-point literal
	tokError              // lexing error (Text holds the message)
)

// String returns a human-readable name for the token kind, used in diagnostics.
func (k tokenKind) String() string {
	switch k {
	case tokEOF:
		return "end of input"
	case tokAt:
		return "'@'"
	case tokIdent:
		return "identifier"
	case tokLParen:
		return "'('"
	case tokRParen:
		return "')'"
	case tokLBracket:
		return "'['"
	case tokRBracket:
		return "']'"
	case tokLBrace:
		return "'{'"
	case tokRBrace:
		return "'}'"
	case tokEquals:
		return "'='"
	case tokComma:
		return "','"
	case tokString:
		return "string"
	case tokInt:
		return "integer"
	case tokFloat:
		return "float"
	case tokError:
		return "error"
	default:
		return "unknown"
	}
}

// lexToken is a single lexed token.
//
// Offset is the byte offset of the token's first rune within the lexer input.
// For tokString, Raw reports whether the literal used backticks. Text holds the
// decoded string content for tokString, the literal text for numbers and
// identifiers, and the error message for tokError.
type lexToken struct {
	Kind   tokenKind
	Text   string
	Offset int
	Raw    bool
}

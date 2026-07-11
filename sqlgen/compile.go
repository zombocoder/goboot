package sqlgen

import "strings"

// Compiled is the result of compiling named-parameter SQL.
type Compiled struct {
	// SQL is the query with named parameters replaced by dialect placeholders.
	SQL string
	// Params lists the named-parameter references in the order they appear,
	// e.g. ["id", "user.ID"]. Each occurrence produces one entry, so a name used
	// twice appears twice and is passed twice (dialect-uniform, since ?-style
	// dialects cannot reuse positions).
	Params []string
}

// Compile rewrites `:name` and `:arg.field` references in the SQL into the
// dialect's positional placeholders and returns the ordered parameter list.
//
// It skips references inside single-quoted string literals and treats `::` as a
// PostgreSQL type cast rather than a parameter. It never panics on arbitrary
// input.
func Compile(sql string, dialect Dialect) Compiled {
	var out strings.Builder
	var params []string
	inString := false

	for i := 0; i < len(sql); {
		c := sql[i]

		if inString {
			// Inside a string literal: copy verbatim, handling '' escapes.
			out.WriteByte(c)
			if c == '\'' {
				if i+1 < len(sql) && sql[i+1] == '\'' {
					out.WriteByte('\'')
					i += 2
					continue
				}
				inString = false
			}
			i++
			continue
		}

		switch {
		case c == '\'':
			inString = true
			out.WriteByte(c)
			i++
		case c == ':' && i+1 < len(sql) && sql[i+1] == ':':
			// PostgreSQL cast operator "::"; copy literally.
			out.WriteString("::")
			i += 2
		case c == ':' && i+1 < len(sql) && isNameStart(sql[i+1]):
			name, next := scanName(sql, i+1)
			params = append(params, name)
			out.WriteString(dialect.Placeholder(len(params)))
			i = next
		default:
			out.WriteByte(c)
			i++
		}
	}

	return Compiled{SQL: out.String(), Params: params}
}

// scanName reads a parameter name starting at index i and returns it with the
// index just past it. Names are [A-Za-z_][A-Za-z0-9_.]*, with a trailing dot
// trimmed so that "user." reads as "user".
func scanName(sql string, i int) (string, int) {
	start := i
	for i < len(sql) && isNamePart(sql[i]) {
		i++
	}
	name := sql[start:i]
	name = strings.TrimRight(name, ".")
	// If we trimmed dots, rewind the index so the dots are emitted as literals.
	return name, start + len(name)
}

func isNameStart(c byte) bool {
	return c == '_' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

func isNamePart(c byte) bool {
	return isNameStart(c) || (c >= '0' && c <= '9') || c == '.'
}

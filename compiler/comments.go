package compiler

import (
	"go/ast"
	"go/token"
	"strings"

	"github.com/zombocoder/goboot/annotation"
)

// lineInfo records where a single line of cleaned comment text originated in
// the source file, so that positions computed over the cleaned text can be
// remapped to true source locations.
type lineInfo struct {
	filename  string
	srcLine   int
	srcColumn int // 1-based byte column of the line's first cleaned byte
	srcOffset int // byte offset of the line's first cleaned byte in the file
}

// cleanComments strips the comment markers from a comment group and returns the
// annotation-parseable text (physical lines joined by newlines) together with a
// per-line table mapping cleaned lines back to source positions.
//
// The annotation parser is oblivious to Go comment syntax: it consumes text as
// though it were a contiguous source slice. Line ("//") comments are not
// contiguous in the source, so rather than feed the parser a synthetic base
// position, we parse with a zero base and remap every resulting position
// through this table (see remapPosition). This keeps the annotation package
// free of comment-format concerns while preserving accurate positions (§37.3).
func cleanComments(group *ast.CommentGroup, fset *token.FileSet) (string, []lineInfo) {
	var (
		lines []string
		table []lineInfo
	)
	for _, c := range group.List {
		start := fset.Position(c.Slash)
		switch {
		case strings.HasPrefix(c.Text, "//"):
			// A line comment contributes exactly one line. Its content begins
			// two bytes past the slash (after "//").
			lines = append(lines, c.Text[2:])
			table = append(table, lineInfo{
				filename:  start.Filename,
				srcLine:   start.Line,
				srcColumn: start.Column + 2,
				srcOffset: start.Offset + 2,
			})
		case strings.HasPrefix(c.Text, "/*"):
			inner := strings.TrimSuffix(c.Text[2:], "*/")
			appendBlockLines(inner, start, &lines, &table)
		default:
			// Not a recognized comment form; keep it as an opaque line so
			// nothing shifts, but it will never look like an annotation.
			lines = append(lines, c.Text)
			table = append(table, lineInfo{
				filename:  start.Filename,
				srcLine:   start.Line,
				srcColumn: start.Column,
				srcOffset: start.Offset,
			})
		}
	}
	return strings.Join(lines, "\n"), table
}

// appendBlockLines splits the inner text of a block comment into physical lines,
// tracking the source position of each line's first byte. The inner text is a
// contiguous slice of the source, so a newline resets the column to 1.
func appendBlockLines(inner string, start token.Position, lines *[]string, table *[]lineInfo) {
	srcLine := start.Line
	srcColumn := start.Column + 2 // first inner byte sits after "/*"
	srcOffset := start.Offset + 2
	lineStart := 0
	for i := 0; i <= len(inner); i++ {
		if i == len(inner) || inner[i] == '\n' {
			*lines = append(*lines, inner[lineStart:i])
			*table = append(*table, lineInfo{
				filename:  start.Filename,
				srcLine:   srcLine,
				srcColumn: srcColumn,
				srcOffset: srcOffset,
			})
			if i < len(inner) {
				srcLine++
				srcColumn = 1
				srcOffset = start.Offset + 2 + i + 1
			}
			lineStart = i + 1
		}
	}
}

// remapPosition converts a position expressed in cleaned-text coordinates (line
// is 1-based within the cleaned text, column is a 1-based byte column) into a
// true source position using the line table. Positions outside the table are
// returned unchanged.
func remapPosition(p token.Position, table []lineInfo) token.Position {
	if p.Line < 1 || p.Line > len(table) {
		return p
	}
	info := table[p.Line-1]
	delta := p.Column - 1
	return token.Position{
		Filename: info.filename,
		Line:     info.srcLine,
		Column:   info.srcColumn + delta,
		Offset:   info.srcOffset + delta,
	}
}

// parseDoc parses the annotations in a doc comment group and returns them with
// positions resolved to true source locations, along with any parse diagnostics
// (also remapped). A nil group yields no annotations.
func parseDoc(group *ast.CommentGroup, fset *token.FileSet) ([]annotation.Annotation, []*annotation.Diagnostic) {
	if group == nil {
		return nil, nil
	}
	text, table := cleanComments(group, fset)
	// Parse with a zero base; every position is remapped through the table.
	zero := token.Position{Line: 1, Column: 1, Offset: 0}
	anns, diags := annotation.ParseComment(text, zero)
	for i := range anns {
		anns[i].Position = remapPosition(anns[i].Position, table)
	}
	for _, d := range diags {
		d.Position = remapPosition(d.Position, table)
	}
	return anns, diags
}

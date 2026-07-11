package compiler

import (
	"strconv"
	"strings"
)

// splitPos parses a "file:line:col" position string as produced by
// packages.Error.Pos. It parses the trailing ":line:col" from the right so that
// colons inside the filename (for example a Windows drive letter) are kept as
// part of the filename. It reports ok=false when the string does not end in the
// expected numeric suffix.
func splitPos(s string) (line, col int, file string, ok bool) {
	lastColon := strings.LastIndexByte(s, ':')
	if lastColon < 0 {
		return 0, 0, "", false
	}
	col, err := strconv.Atoi(s[lastColon+1:])
	if err != nil {
		return 0, 0, "", false
	}
	rest := s[:lastColon]
	prevColon := strings.LastIndexByte(rest, ':')
	if prevColon < 0 {
		return 0, 0, "", false
	}
	line, err = strconv.Atoi(rest[prevColon+1:])
	if err != nil {
		return 0, 0, "", false
	}
	return line, col, rest[:prevColon], true
}

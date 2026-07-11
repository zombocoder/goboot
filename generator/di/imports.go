package di

import (
	"fmt"
	"go/types"
	"sort"
	"strconv"
	"strings"
)

// imports collects the packages a generated file references and assigns each a
// stable, non-colliding alias. It doubles as a go/types qualifier so that types
// rendered with types.TypeString automatically register their packages.
type imports struct {
	// selfPath is the import path of the package being generated into; symbols
	// in it are referenced without qualification.
	selfPath string
	aliasOf  map[string]string // import path -> alias
	pathOf   map[string]string // alias -> import path (for collision checks)
}

func newImports(selfPath string) *imports {
	return &imports{
		selfPath: selfPath,
		aliasOf:  map[string]string{},
		pathOf:   map[string]string{},
	}
}

// add registers a package and returns the alias to reference it by. Registering
// the package being generated into, or a package with an empty path, returns an
// empty alias (no qualification).
func (im *imports) add(path, name string) string {
	if path == "" || path == im.selfPath {
		return ""
	}
	if alias, ok := im.aliasOf[path]; ok {
		return alias
	}
	base := name
	if base == "" {
		base = lastSegment(path)
	}
	alias := base
	for i := 2; ; i++ {
		if existing, taken := im.pathOf[alias]; !taken || existing == path {
			break
		}
		alias = base + strconv.Itoa(i)
	}
	im.aliasOf[path] = alias
	im.pathOf[alias] = path
	return alias
}

// aliases returns every alias currently registered. Used to keep generated
// local variable names from colliding with imported package identifiers.
func (im *imports) aliases() []string {
	out := make([]string, 0, len(im.pathOf))
	for alias := range im.pathOf {
		out = append(out, alias)
	}
	return out
}

// qualifier returns a types.Qualifier that registers packages as it renders
// them and yields the assigned alias.
func (im *imports) qualifier() types.Qualifier {
	return func(p *types.Package) string {
		return im.add(p.Path(), p.Name())
	}
}

// qualify renders a package-qualified symbol, e.g. "service.NewUserService",
// registering the import. For the self package it returns just the symbol.
func (im *imports) qualify(path, name, symbol string) string {
	alias := im.add(path, name)
	if alias == "" {
		return symbol
	}
	return alias + "." + symbol
}

// block renders the import block (empty string when nothing is imported). Paths
// are sorted for deterministic output; an alias that differs from the package's
// default name is emitted explicitly.
func (im *imports) block() string {
	if len(im.aliasOf) == 0 {
		return ""
	}
	paths := make([]string, 0, len(im.aliasOf))
	for path := range im.aliasOf {
		paths = append(paths, path)
	}
	sort.Strings(paths)

	var b strings.Builder
	b.WriteString("import (\n")
	for _, path := range paths {
		alias := im.aliasOf[path]
		if alias == lastSegment(path) {
			b.WriteString(fmt.Sprintf("\t%s\n", strconv.Quote(path)))
		} else {
			b.WriteString(fmt.Sprintf("\t%s %s\n", alias, strconv.Quote(path)))
		}
	}
	b.WriteString(")\n")
	return b.String()
}

// lastSegment returns the final path element, used as a package's default name.
func lastSegment(path string) string {
	if i := strings.LastIndexByte(path, '/'); i >= 0 {
		return path[i+1:]
	}
	return path
}

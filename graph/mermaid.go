package graph

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/zombocoder/goboot/model"
)

// Mermaid renders the graph as a Mermaid flowchart (§43.4). Output is
// deterministic: nodes and edges follow the graph's sorted order. Each edge
// "A --> B" reads as "A depends on B".
func (g *Graph) Mermaid() string {
	var b strings.Builder
	b.WriteString("flowchart TD\n")
	for _, id := range g.nodes {
		b.WriteString(fmt.Sprintf("    %s[%q]\n", nodeKey(id), g.label(id)))
	}
	for _, id := range g.nodes {
		for _, dep := range g.edges[id] {
			b.WriteString(fmt.Sprintf("    %s --> %s\n", nodeKey(id), nodeKey(dep)))
		}
	}
	return b.String()
}

// label returns a short human-readable label for a node.
func (g *Graph) label(id model.ComponentID) string {
	if c := g.index[id]; c != nil && c.Name != "" {
		return c.Name
	}
	return string(id)
}

// nodeKey derives a stable, Mermaid-safe identifier from a component ID by
// hashing it, since component IDs contain characters (slashes, colons, hashes)
// that are not valid Mermaid node identifiers.
func nodeKey(id model.ComponentID) string {
	sum := sha1.Sum([]byte(id))
	return "n" + hex.EncodeToString(sum[:6])
}

// Package graph builds and analyzes the component dependency graph (§15). It
// provides deterministic construction ordering via topological sort, cycle
// detection with readable diagnostic paths, reverse ordering for shutdown, and
// Mermaid export for visualization. Edges point from a consumer to each
// component it depends on.
package graph

import (
	"sort"

	"github.com/zombocoder/goboot/model"
)

// Graph is a directed dependency graph over components. An edge from A to B
// means "A depends on B", so B must be constructed before A.
type Graph struct {
	nodes []model.ComponentID                       // node IDs, sorted
	edges map[model.ComponentID][]model.ComponentID // consumer -> sorted, deduped deps
	index map[model.ComponentID]*model.Component
}

// Build constructs a graph from the resolved components. Dependency edges are
// taken from each component's resolved dependencies. Nodes and adjacency lists
// are sorted so that all downstream traversal is deterministic (§6.7).
// Dependencies that do not correspond to a known node (e.g. unresolved) are
// ignored here; the resolver is responsible for reporting them.
func Build(components []*model.Component) *Graph {
	g := &Graph{
		edges: make(map[model.ComponentID][]model.ComponentID),
		index: make(map[model.ComponentID]*model.Component),
	}
	for _, c := range components {
		g.nodes = append(g.nodes, c.ID)
		g.index[c.ID] = c
	}
	sort.Slice(g.nodes, func(i, j int) bool { return g.nodes[i] < g.nodes[j] })

	for _, c := range components {
		seen := make(map[model.ComponentID]bool)
		var deps []model.ComponentID
		for _, dep := range c.DependsOn() {
			if _, ok := g.index[dep]; !ok || seen[dep] || dep == c.ID {
				continue
			}
			seen[dep] = true
			deps = append(deps, dep)
		}
		sort.Slice(deps, func(i, j int) bool { return deps[i] < deps[j] })
		g.edges[c.ID] = deps
	}
	return g
}

// Nodes returns the component IDs in deterministic order.
func (g *Graph) Nodes() []model.ComponentID {
	out := make([]model.ComponentID, len(g.nodes))
	copy(out, g.nodes)
	return out
}

// Dependencies returns the sorted dependency IDs of a component.
func (g *Graph) Dependencies(id model.ComponentID) []model.ComponentID {
	return g.edges[id]
}

// Component returns the component for an ID, or nil.
func (g *Graph) Component(id model.ComponentID) *model.Component {
	return g.index[id]
}

// Len returns the number of nodes.
func (g *Graph) Len() int { return len(g.nodes) }

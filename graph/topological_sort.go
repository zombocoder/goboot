package graph

import "github.com/zombocoder/goboot/model"

// Cycle is a dependency cycle: an ordered path of component IDs where the last
// depends on the first (§15.1). The path lists each component once, in the
// order they depend on one another.
type Cycle struct {
	Path []model.ComponentID
}

// ConstructionOrder returns the component IDs in an order where every
// component's dependencies precede it, suitable for singleton initialization
// (§15.2). If the graph contains a cycle, it returns nil and the cycle.
//
// The traversal visits nodes and their dependencies in sorted order, so the
// resulting order is deterministic for a given graph.
func (g *Graph) ConstructionOrder() ([]model.ComponentID, *Cycle) {
	const (
		white = 0 // unvisited
		gray  = 1 // on the current DFS stack
		black = 2 // fully processed
	)
	color := make(map[model.ComponentID]int, len(g.nodes))
	order := make([]model.ComponentID, 0, len(g.nodes))
	var stack []model.ComponentID

	var visit func(id model.ComponentID) *Cycle
	visit = func(id model.ComponentID) *Cycle {
		color[id] = gray
		stack = append(stack, id)
		for _, dep := range g.edges[id] {
			switch color[dep] {
			case white:
				if cyc := visit(dep); cyc != nil {
					return cyc
				}
			case gray:
				return &Cycle{Path: cycleFromStack(stack, dep)}
			}
		}
		stack = stack[:len(stack)-1]
		color[id] = black
		order = append(order, id)
		return nil
	}

	for _, id := range g.nodes {
		if color[id] == white {
			if cyc := visit(id); cyc != nil {
				return nil, cyc
			}
		}
	}
	return order, nil
}

// ShutdownOrder returns the reverse of ConstructionOrder, so that components are
// torn down before their dependencies (§15.3). It returns the cycle if one
// exists.
func (g *Graph) ShutdownOrder() ([]model.ComponentID, *Cycle) {
	order, cyc := g.ConstructionOrder()
	if cyc != nil {
		return nil, cyc
	}
	reversed := make([]model.ComponentID, len(order))
	for i, id := range order {
		reversed[len(order)-1-i] = id
	}
	return reversed, nil
}

// cycleFromStack extracts the cycle path from the DFS stack, starting at the
// node the back-edge points to.
func cycleFromStack(stack []model.ComponentID, start model.ComponentID) []model.ComponentID {
	for i, id := range stack {
		if id == start {
			path := make([]model.ComponentID, len(stack)-i)
			copy(path, stack[i:])
			return path
		}
	}
	return append([]model.ComponentID(nil), start)
}

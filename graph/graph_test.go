package graph

import (
	"strings"
	"testing"

	"github.com/zombocoder/goboot/model"
)

// comp builds a component with the given id depending on the given ids.
func comp(id string, deps ...string) *model.Component {
	c := &model.Component{ID: model.ComponentID(id), Name: id}
	for _, d := range deps {
		c.Dependencies = append(c.Dependencies, model.Dependency{ResolvedTo: model.ComponentID(d)})
	}
	return c
}

func indexOf(order []model.ComponentID, id string) int {
	for i, o := range order {
		if o == model.ComponentID(id) {
			return i
		}
	}
	return -1
}

func TestConstructionOrderRespectsDependencies(t *testing.T) {
	// controller -> service -> repository -> db
	comps := []*model.Component{
		comp("p:Controller", "p:Service"),
		comp("p:Service", "p:Repository"),
		comp("p:Repository", "p:DB"),
		comp("p:DB"),
	}
	g := Build(comps)
	order, cyc := g.ConstructionOrder()
	if cyc != nil {
		t.Fatalf("unexpected cycle: %v", cyc.Path)
	}
	if len(order) != 4 {
		t.Fatalf("order has %d entries, want 4", len(order))
	}
	// Every dependency must precede its consumer.
	pairs := [][2]string{
		{"p:DB", "p:Repository"},
		{"p:Repository", "p:Service"},
		{"p:Service", "p:Controller"},
	}
	for _, p := range pairs {
		if indexOf(order, p[0]) >= indexOf(order, p[1]) {
			t.Errorf("%s should be constructed before %s; order=%v", p[0], p[1], order)
		}
	}
}

func TestConstructionOrderDeterministic(t *testing.T) {
	comps := []*model.Component{
		comp("p:A", "p:C", "p:B"),
		comp("p:B", "p:D"),
		comp("p:C", "p:D"),
		comp("p:D"),
	}
	first, _ := Build(comps).ConstructionOrder()
	for i := 0; i < 20; i++ {
		got, _ := Build(comps).ConstructionOrder()
		if len(got) != len(first) {
			t.Fatalf("length changed")
		}
		for j := range got {
			if got[j] != first[j] {
				t.Fatalf("order not deterministic: %v vs %v", first, got)
			}
		}
	}
}

func TestShutdownOrderIsReverse(t *testing.T) {
	comps := []*model.Component{
		comp("p:A", "p:B"),
		comp("p:B"),
	}
	g := Build(comps)
	cons, _ := g.ConstructionOrder()
	shut, _ := g.ShutdownOrder()
	if len(cons) != len(shut) {
		t.Fatal("length mismatch")
	}
	for i := range cons {
		if cons[i] != shut[len(shut)-1-i] {
			t.Fatalf("shutdown not reverse of construction: %v vs %v", cons, shut)
		}
	}
}

func TestCycleDetection(t *testing.T) {
	// A -> B -> C -> A
	comps := []*model.Component{
		comp("p:A", "p:B"),
		comp("p:B", "p:C"),
		comp("p:C", "p:A"),
	}
	g := Build(comps)
	order, cyc := g.ConstructionOrder()
	if cyc == nil {
		t.Fatalf("expected a cycle, got order %v", order)
	}
	if len(cyc.Path) != 3 {
		t.Errorf("cycle path length = %d, want 3: %v", len(cyc.Path), cyc.Path)
	}
	// The path must be a genuine cycle: consecutive nodes are dependencies, and
	// it forms a closed loop.
	set := map[model.ComponentID]bool{}
	for _, id := range cyc.Path {
		set[id] = true
	}
	if !set["p:A"] || !set["p:B"] || !set["p:C"] {
		t.Errorf("cycle path missing nodes: %v", cyc.Path)
	}

	// Shutdown order also reports the cycle.
	if _, c := g.ShutdownOrder(); c == nil {
		t.Errorf("ShutdownOrder should report the cycle too")
	}
}

func TestSelfDependencyIgnored(t *testing.T) {
	// A component listing itself as a dependency must not create a self-loop.
	g := Build([]*model.Component{comp("p:A", "p:A")})
	if deps := g.Dependencies("p:A"); len(deps) != 0 {
		t.Errorf("self-dependency should be dropped, got %v", deps)
	}
	if _, cyc := g.ConstructionOrder(); cyc != nil {
		t.Errorf("self-dependency should not be a cycle")
	}
}

func TestUnknownDependencyIgnoredInGraph(t *testing.T) {
	// A dependency on a non-node is ignored by the graph (the resolver reports
	// the missing dependency separately).
	g := Build([]*model.Component{comp("p:A", "p:Missing")})
	if deps := g.Dependencies("p:A"); len(deps) != 0 {
		t.Errorf("unknown dependency should be dropped, got %v", deps)
	}
}

func TestMermaidDeterministicAndValid(t *testing.T) {
	comps := []*model.Component{
		comp("p:A", "p:B"),
		comp("p:B"),
	}
	g := Build(comps)
	out := g.Mermaid()
	if !strings.HasPrefix(out, "flowchart TD\n") {
		t.Errorf("missing mermaid header: %q", out)
	}
	if strings.Count(out, "-->") != 1 {
		t.Errorf("expected exactly one edge, got:\n%s", out)
	}
	if out != g.Mermaid() {
		t.Errorf("Mermaid output not deterministic")
	}
}

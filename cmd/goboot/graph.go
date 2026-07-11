package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"sort"

	"github.com/zombocoder/goboot/compiler"
	"github.com/zombocoder/goboot/graph"
	"github.com/zombocoder/goboot/model"
)

func cmdGraph(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("graph", flag.ContinueOnError)
	fs.SetOutput(stderr)
	var (
		dir    = fs.String("dir", ".", "working directory")
		tags   = fs.String("tags", "", "comma-separated build tags")
		format = fs.String("format", "text", "output format: text, mermaid, dot, json")
	)
	if err := fs.Parse(args); err != nil {
		return 2
	}

	cfg, err := loadConfig(*dir)
	if err != nil {
		fmt.Fprintf(stderr, "goboot: %v\n", err)
		return 1
	}
	patterns := resolvePatterns(fs.Args(), cfg)

	// Graph output tolerates non-fatal diagnostics; only a load failure aborts.
	res, _, _ := analyzeCommon(*dir, patterns, *tags, false, compiler.Options{}, stderr)
	if res == nil {
		return 1
	}

	out, err := renderGraph(res.App, res.Graph, *format)
	if err != nil {
		fmt.Fprintf(stderr, "goboot: %v\n", err)
		return 2
	}
	fmt.Fprint(stdout, out)
	return 0
}

// renderGraph renders the dependency graph in the requested format (§43.4).
func renderGraph(app *model.Application, g *graph.Graph, format string) (string, error) {
	switch format {
	case "mermaid":
		return g.Mermaid(), nil
	case "dot":
		return dotGraph(g), nil
	case "json":
		return jsonGraph(g)
	case "text", "":
		return textGraph(g), nil
	default:
		return "", fmt.Errorf("unknown graph format %q (want text, mermaid, dot, or json)", format)
	}
}

// textGraph renders a human-readable adjacency listing.
func textGraph(g *graph.Graph) string {
	var b []byte
	for _, id := range g.Nodes() {
		b = append(b, id...)
		deps := g.Dependencies(id)
		if len(deps) == 0 {
			b = append(b, '\n')
			continue
		}
		b = append(b, " ->\n"...)
		for _, dep := range deps {
			b = append(b, "    "...)
			b = append(b, dep...)
			b = append(b, '\n')
		}
	}
	return string(b)
}

// dotGraph renders a Graphviz DOT digraph.
func dotGraph(g *graph.Graph) string {
	var b []byte
	b = append(b, "digraph goboot {\n"...)
	for _, id := range g.Nodes() {
		for _, dep := range g.Dependencies(id) {
			b = append(b, fmt.Sprintf("  %q -> %q;\n", string(id), string(dep))...)
		}
	}
	b = append(b, "}\n"...)
	return string(b)
}

// jsonGraph renders the graph as a deterministic JSON object of adjacency lists.
func jsonGraph(g *graph.Graph) (string, error) {
	nodes := g.Nodes()
	adjacency := make(map[string][]string, len(nodes))
	ids := make([]string, 0, len(nodes))
	for _, id := range nodes {
		deps := g.Dependencies(id)
		strs := make([]string, len(deps))
		for i, d := range deps {
			strs[i] = string(d)
		}
		sort.Strings(strs)
		adjacency[string(id)] = strs
		ids = append(ids, string(id))
	}
	sort.Strings(ids)
	payload := struct {
		Nodes []string            `json:"nodes"`
		Edges map[string][]string `json:"edges"`
	}{Nodes: ids, Edges: adjacency}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data) + "\n", nil
}

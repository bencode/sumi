package dagre

import (
	"fmt"
	"maps"
	"slices"
	"testing"
)

// The orders below are dagre-d3-es 7.0.14's for the same graphs. dagre's
// names for what it adds depend on a counter, so a dummy shows as the edge
// it stands for and a border node as its subgraph.
func TestOrderGraph(t *testing.T) {
	n := func(id, parent string) Node { return Node{ID: id, Parent: parent, Width: 40, Height: 30} }
	e := func(name, v, w string) Edge { return Edge{V: v, W: w, Name: name, MinLen: 1} }
	labeled := e("L0", "A", "B")
	labeled.Width, labeled.Height = 30, 20
	for _, c := range []struct {
		name    string
		in      Graph
		layers  [][]string
		parents map[string]string
		labels  map[string][2]float64
	}{{
		name: "a crossing sorted away",
		in: Graph{
			Nodes: []Node{n("A", ""), n("B", ""), n("C", ""), n("D", ""), n("E", ""), n("F", "")},
			Edges: []Edge{e("L0", "A", "F"), e("L1", "D", "F"), e("L2", "B", "C")},
		},
		layers:  [][]string{{"A", "D", "B", "E"}, {"edge:L0#1", "edge:L1#1", "edge:L2#1"}, {"F", "C"}},
		parents: map[string]string{"edge:L0#1": "", "edge:L1#1": "", "edge:L2#1": ""},
	}, {
		name: "array indices listed first",
		in: Graph{
			Nodes: []Node{n("b", ""), n("10", ""), n("2", ""), n("c", ""), n("d", "")},
			Edges: []Edge{e("L0", "b", "d"), e("L1", "10", "c"), e("L2", "2", "d")},
		},
		layers:  [][]string{{"b", "2", "10"}, {"edge:L0#1", "edge:L2#1", "edge:L1#1"}, {"d", "c"}},
		parents: map[string]string{"edge:L0#1": "", "edge:L1#1": "", "edge:L2#1": ""},
	}, {
		name: "subgraphs side by side, with an edge from one into the other",
		in: Graph{
			Nodes: []Node{n("x", ""), n("s1", ""), n("a", "s1"), n("b", "s1"), n("s2", ""), n("c", "s2"), n("d", "s2")},
			Edges: []Edge{e("L0", "x", "c"), e("L1", "a", "b"), e("L2", "c", "d"), e("L3", "a", "d")},
		},
		layers: [][]string{
			{"x"},
			{"edge:L0#1"},
			{"bl:s1#2", "bt:s1", "br:s1#2", "bl:s2#2", "bt:s2", "edge:L0#2", "br:s2#2"},
			{"bl:s1#3", "a", "br:s1#3", "bl:s2#3", "c", "br:s2#3"},
			{"bl:s1#4", "edge:L1#4", "edge:L3#4", "br:s1#4", "bl:s2#4", "edge:L2#4", "br:s2#4"},
			{"bl:s1#5", "b", "br:s1#5", "bl:s2#5", "d", "br:s2#5"},
			{"bl:s1#6", "br:s1#6", "bl:s2#6", "br:s2#6", "bb:s1", "bb:s2"},
		},
		parents: map[string]string{
			"edge:L0#1": "", "edge:L0#2": "s2", "edge:L1#4": "s1", "edge:L2#4": "s2", "edge:L3#4": "s1",
		},
	}, {
		name: "a label takes up room in its rank",
		in: Graph{
			Nodes: []Node{n("A", ""), n("B", ""), n("C", "")},
			Edges: []Edge{labeled, e("L1", "A", "C")},
		},
		layers:  [][]string{{"A"}, {"edge:L0#1", "edge:L1#1"}, {"B", "C"}},
		parents: map[string]string{"edge:L0#1": "", "edge:L1#1": ""},
		labels:  map[string][2]float64{"edge:L0#1": {30, 20}},
	}} {
		g := orderGraph(c.in, &counter{})
		parents, labels := map[string]string{}, map[string][2]float64{}
		for _, v := range g.nodeNames() {
			l := g.node(v)
			if l.edgeLabel != nil {
				parents[canonical(g, v)] = g.parentOf(v)
			}
			if l.dummy == "edge-label" {
				labels[canonical(g, v)] = [2]float64{l.width, l.height}
			}
		}
		layers := orderedLayers(g)
		if !slices.EqualFunc(layers, c.layers, slices.Equal[[]string]) ||
			!maps.Equal(parents, c.parents) || !maps.Equal(labels, orEmpty(c.labels)) {
			t.Errorf("%s: layers %q, parents %v, labels %v; want %q, %v, %v",
				c.name, layers, parents, labels, c.layers, c.parents, c.labels)
		}
	}
}

func orderedLayers(g *layoutGraph) [][]string {
	layers := layerMatrix(g)
	for _, vs := range layers {
		for i, v := range vs {
			vs[i] = canonical(g, v)
		}
	}
	return layers
}

func canonical(g *layoutGraph, v string) string {
	n, parent := g.node(v), g.parentOf(v)
	switch {
	case n.edgeLabel != nil && n.edgeLabel.reversed:
		return fmt.Sprintf("edge:%s#%d", n.edgeLabel.forwardName, n.rank)
	case n.edgeLabel != nil:
		return fmt.Sprintf("edge:%s#%d", n.edgeObj.name, n.rank)
	case n.borderType == "borderLeft":
		return fmt.Sprintf("bl:%s#%d", parent, n.rank)
	case n.borderType == "borderRight":
		return fmt.Sprintf("br:%s#%d", parent, n.rank)
	case n.dummy == "border" && g.node(parent).borderTop == v:
		return "bt:" + parent
	case n.dummy == "border":
		return "bb:" + parent
	}
	return v
}

package dagre

import (
	"fmt"
	"slices"
	"strings"
	"testing"
)

// The orders below are dagre-d3-es 7.0.14's for the same graphs. dagre's
// names for what it adds depend on a counter, so a dummy shows as the edge
// it stands for and a border node as its subgraph. Each layer is its nodes
// in order; parents and labels are some of the dummies'.
func TestOrderGraph(t *testing.T) {
	n := func(id, parent string) Node { return Node{ID: id, Parent: parent, Width: 40, Height: 30} }
	e := func(name, v, w string, minlen int) Edge { return Edge{V: v, W: w, Name: name, MinLen: minlen} }
	l := func(name, v, w string, minlen int) Edge {
		return Edge{V: v, W: w, Name: name, MinLen: minlen, Width: 30, Height: 20}
	}
	for _, c := range []struct {
		name    string
		in      Graph
		layers  []string
		parents map[string]string
		labels  map[string][2]float64
	}{{
		name: "a crossing sorted away",
		in: Graph{
			Nodes: []Node{n("A", ""), n("B", ""), n("C", ""), n("D", ""), n("E", ""), n("F", "")},
			Edges: []Edge{e("L0", "A", "F", 1), e("L1", "D", "F", 1), e("L2", "B", "C", 1)},
		},
		layers: []string{"A D B E", "edge:L0#1 edge:L1#1 edge:L2#1", "F C"},
	}, {
		name: "array indices listed first",
		in: Graph{
			Nodes: []Node{n("b", ""), n("10", ""), n("2", ""), n("c", ""), n("d", "")},
			Edges: []Edge{e("L0", "b", "d", 1), e("L1", "10", "c", 1), e("L2", "2", "d", 1)},
		},
		layers: []string{"b 2 10", "edge:L0#1 edge:L2#1 edge:L1#1", "d c"},
	}, {
		name: "subgraphs side by side, with an edge from one into the other",
		in: Graph{
			Nodes: []Node{n("x", ""), n("s1", ""), n("a", "s1"), n("b", "s1"), n("s2", ""), n("c", "s2"), n("d", "s2")},
			Edges: []Edge{e("L0", "x", "c", 1), e("L1", "a", "b", 1), e("L2", "c", "d", 1), e("L3", "a", "d", 1)},
		},
		layers: []string{
			"x",
			"edge:L0#1",
			"bl:s1#2 bt:s1 br:s1#2 bl:s2#2 bt:s2 edge:L0#2 br:s2#2",
			"bl:s1#3 a br:s1#3 bl:s2#3 c br:s2#3",
			"bl:s1#4 edge:L1#4 edge:L3#4 br:s1#4 bl:s2#4 edge:L2#4 br:s2#4",
			"bl:s1#5 b br:s1#5 bl:s2#5 d br:s2#5",
			"bl:s1#6 br:s1#6 bl:s2#6 br:s2#6 bb:s1 bb:s2",
		},
		parents: map[string]string{
			"edge:L0#1": "", "edge:L0#2": "s2", "edge:L1#4": "s1", "edge:L2#4": "s2", "edge:L3#4": "s1",
		},
	}, {
		name: "a label takes up room in its rank",
		in: Graph{
			Nodes: []Node{n("A", ""), n("B", ""), n("C", "")},
			Edges: []Edge{l("L0", "A", "B", 1), e("L1", "A", "C", 1)},
		},
		layers: []string{"A", "edge:L0#1 edge:L1#1", "B C"},
		labels: map[string][2]float64{"edge:L0#1": {30, 20}},
	}, {
		name: "nested subgraphs from rank 0, sorted leaning both ways",
		in: Graph{
			Nodes: []Node{n("v4", "4294967295"), n("3", "v5"), n("_q", ""), n("v12", "v5"), n("b", "v10"),
				n("v6", "v5"), n("v10", "4294967295"), n("01", "v10"), n("v5", "4294967295"),
				n("x1", "4294967295"), n("12", "v10"), n("a", "v10"), n("4294967295", "")},
			Edges: []Edge{l("L0", "12", "b", 1), e("L1", "x1", "v4", 3), l("L2", "v12", "01", 1),
				e("L3", "x1", "v4", 1), e("L4", "x1", "v6", 3), l("L5", "12", "v4", 1)},
		},
		layers: []string{
			"bl:4294967295#0 bt:4294967295 br:4294967295#0",
			"bl:4294967295#1 x1 br:4294967295#1 _q",
			"bl:4294967295#2 edge:L4#2 edge:L3#2 edge:L1#2 br:4294967295#2",
			"bl:4294967295#3 edge:L4#3 edge:L3#3 edge:L1#3 br:4294967295#3",
			"bl:4294967295#4 edge:L4#4 edge:L3#4 edge:L1#4 br:4294967295#4",
			"bl:4294967295#5 bl:v5#5 bt:v5 edge:L4#5 br:v5#5 bl:v10#5 bt:v10 br:v10#5 edge:L3#5 edge:L1#5 br:4294967295#5",
			"bl:4294967295#6 bl:v5#6 v12 edge:L4#6 br:v5#6 bl:v10#6 12 br:v10#6 edge:L3#6 edge:L1#6 br:4294967295#6",
			"bl:4294967295#7 bl:v5#7 edge:L2#7 edge:L4#7 br:v5#7 bl:v10#7 edge:L5#7 edge:L0#7 br:v10#7 edge:L3#7 edge:L1#7 br:4294967295#7",
			"bl:4294967295#8 bl:v5#8 3 v6 br:v5#8 bl:v10#8 01 b a br:v10#8 v4 br:4294967295#8",
			"bl:4294967295#9 bl:v5#9 bb:v5 br:v5#9 bl:v10#9 bb:v10 br:v10#9 br:4294967295#9",
			"bl:4294967295#10 bb:4294967295 br:4294967295#10",
		},
		parents: map[string]string{"edge:L4#4": "4294967295", "edge:L4#5": "v5"},
		labels:  map[string][2]float64{"edge:L2#7": {30, 20}},
	}, {
		name: "stopping three sweeps after the best",
		in: Graph{
			Nodes: []Node{n("7", "2"), n("_q", "100"), n("end", "100"), n("B", ""), n("A", ""), n("2", ""),
				n("v5", ""), n("100", "2"), n("E", ""), n("0", "")},
			Edges: []Edge{l("L0", "end", "0", 3), l("L1", "_q", "A", 3), e("L2", "E", "A", 1),
				l("L3", "v5", "B", 1), l("L4", "7", "0", 1)},
		},
		layers: []string{
			"bl:2#0 bt:2 br:2#0",
			"bl:2#1 bl:100#1 bt:100 br:100#1 br:2#1",
			"v5 bl:2#2 bl:100#2 end _q br:100#2 7 br:2#2",
			"edge:L3#3 bl:2#3 bl:100#3 bb:100 edge:L0#3 edge:L1#3 br:100#3 edge:L4#3 br:2#3",
			"edge:L3#4 bl:2#4 bb:2 edge:L0#4 edge:L4#4 edge:L1#4 br:2#4",
			"edge:L3#5 edge:L0#5 edge:L4#5 edge:L1#5",
			"B edge:L0#6 edge:L4#6 edge:L1#6",
			"edge:L0#7 edge:L4#7 edge:L1#7",
			"edge:L0#8 edge:L4#8 edge:L1#8 E",
			"edge:L0#9 edge:L4#9 edge:L1#9 edge:L2#9",
			"0 A",
		},
		parents: map[string]string{"edge:L0#3": "100", "edge:L0#4": "2", "edge:L0#5": ""},
		labels:  map[string][2]float64{"edge:L3#5": {30, 20}},
	}, {
		name: "subgraphs keeping their order from rank to rank",
		in: Graph{
			Nodes: []Node{n("n2_1", "s2"), n("n0_1", "s0"), n("s1", ""), n("t1", ""), n("n0_0", "s0"),
				n("n1_0", "s1"), n("s2", ""), n("n1_1", "s1"), n("s0", ""), n("n2_0", "s2")},
			Edges: []Edge{e("L0", "n0_0", "n0_1", 1), e("L1", "n1_0", "n2_0", 1), e("L2", "n0_1", "n1_1", 1),
				e("L3", "n2_1", "n0_1", 1)},
		},
		layers: []string{
			"bl:s2#0 bt:s2 br:s2#0 bl:s1#0 bt:s1 br:s1#0 bl:s0#0 bt:s0 br:s0#0",
			"bl:s2#1 n2_1 br:s2#1 t1 bl:s1#1 n1_0 br:s1#1 bl:s0#1 n0_0 br:s0#1",
			"bl:s2#2 edge:L3#2 br:s2#2 bl:s1#2 edge:L1#2 br:s1#2 bl:s0#2 edge:L0#2 br:s0#2",
			"bl:s2#3 n2_0 br:s2#3 bl:s1#3 br:s1#3 bl:s0#3 n0_1 br:s0#3",
			"bl:s2#4 bb:s2 br:s2#4 bl:s1#4 br:s1#4 bl:s0#4 bb:s0 edge:L2#4 br:s0#4",
			"bl:s1#5 edge:L2#5 br:s1#5",
			"bl:s1#6 n1_1 br:s1#6",
			"bl:s1#7 bb:s1 br:s1#7",
		},
		parents: map[string]string{"edge:L2#4": "s0", "edge:L2#5": "s1"},
	}} {
		g := orderGraph(c.in, &counter{})
		layers := orderedLayers(g)
		parents, labels := map[string]string{}, map[string][2]float64{}
		for _, v := range g.nodeNames() {
			d := g.node(v)
			if d.edgeLabel != nil {
				parents[canonical(g, v)] = g.parentOf(v)
			}
			if d.dummy == "edge-label" {
				labels[canonical(g, v)] = [2]float64{d.width, d.height}
			}
		}
		if !slices.Equal(layers, c.layers) || !holds(parents, c.parents) || !holds(labels, c.labels) {
			t.Errorf("%s: layers %q, parents %v, labels %v; want %q, %v, %v",
				c.name, layers, parents, labels, c.layers, c.parents, c.labels)
		}
	}
}

// Whole graphs seldom have one entry merge two others, so this goes
// straight to resolveConflicts, with dagre's result for the same input.
func TestResolveConflicts(t *testing.T) {
	entries := []*entry{
		{vs: []string{"g"}, i: 0, center: center{1.5, 1, true}},
		{vs: []string{"d"}, i: 1, center: center{10, 4, true}},
		{vs: []string{"a"}, i: 2, center: center{0.5, 4, true}},
		{vs: []string{"0"}, i: 3, center: center{0.5, 4, true}},
		{vs: []string{"f"}, i: 4, center: center{1.5, 3, true}},
		{vs: []string{"1"}, i: 5, center: center{4, 4, true}},
	}
	cg := newGraph[struct{}, struct{}](true, false)
	for _, k := range [][2]string{{"0", "d"}, {"g", "f"}, {"1", "a"}, {"0", "a"}, {"a", "f"}} {
		cg.setEdge(k[0], k[1], "", nil)
	}
	want := []entry{
		{vs: []string{"g"}, i: 0, center: center{1.5, 1, true}},
		{vs: []string{"1", "0", "a", "f"}, i: 2, center: center{1.6333333333333333, 15, true}},
		{vs: []string{"d"}, i: 1, center: center{10, 4, true}},
	}
	got := resolveConflicts(entries, cg)
	same := slices.EqualFunc(got, want, func(g *entry, w entry) bool {
		return slices.Equal(g.vs, w.vs) && g.i == w.i && g.center == w.center
	})
	if !same {
		t.Errorf("got %v, want %v", got, want)
	}
}

func holds[V comparable](m, some map[string]V) bool {
	for k, v := range some {
		if got, ok := m[k]; !ok || got != v {
			return false
		}
	}
	return true
}

func orderedLayers(g *layoutGraph) []string {
	var layers []string
	for _, vs := range layerMatrix(g) {
		names := make([]string, len(vs))
		for i, v := range vs {
			names[i] = canonical(g, v)
		}
		layers = append(layers, strings.Join(names, " "))
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

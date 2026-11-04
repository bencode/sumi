package dagre

import (
	"maps"
	"slices"
	"testing"
)

// The results below are dagre-d3-es 7.0.14's for the same graphs.
func TestRankGraph(t *testing.T) {
	n := func(id, parent string) Node { return Node{ID: id, Parent: parent, Width: 40, Height: 30} }
	e := func(name, v, w string, minlen int) Edge { return Edge{V: v, W: w, Name: name, MinLen: minlen} }
	labeled := e("L0", "A", "B", 1)
	labeled.Width, labeled.Height = 30, 20
	for _, c := range []struct {
		name     string
		in       Graph
		ranks    map[string]int
		reversed []string
		clusters map[string][2]int
		labels   map[string]float64
	}{{
		name: "a cycle and two edges between the same nodes",
		in: Graph{
			Nodes: []Node{n("A", ""), n("B", ""), n("C", ""), n("D", "")},
			Edges: []Edge{e("L0", "A", "B", 1), e("L1", "B", "D", 1), e("L2", "A", "C", 1),
				e("L3", "C", "D", 1), e("L4", "A", "D", 1), e("L5", "D", "A", 1)},
		},
		ranks:    map[string]int{"A": 0, "B": 2, "C": 2, "D": 4},
		reversed: []string{"L5"},
	}, {
		name: "an array index comes first, so its edge back is the one reversed",
		in: Graph{
			Nodes: []Node{n("Z9", ""), n("0", "")},
			Edges: []Edge{e("L0", "0", "Z9", 1), e("L1", "Z9", "0", 1)},
		},
		ranks:    map[string]int{"0": 0, "Z9": 2},
		reversed: []string{"L1"},
	}, {
		name: "minlen, and a node pulled up to the edge it hangs on",
		in: Graph{
			Nodes: []Node{n("A", ""), n("B", ""), n("C", "")},
			Edges: []Edge{e("L0", "A", "B", 3), e("L1", "A", "C", 1), e("L2", "C", "B", 1)},
		},
		ranks: map[string]int{"A": 0, "B": 6, "C": 4},
	}, {
		name: "nested subgraphs",
		in: Graph{
			Nodes: []Node{n("outer", ""), n("inner", "outer"), n("x", "inner"), n("y", "outer"), n("z", "")},
			Edges: []Edge{e("L0", "z", "x", 1), e("L1", "x", "y", 1)},
		},
		ranks:    map[string]int{"x": 4, "y": 7, "z": 0},
		clusters: map[string][2]int{"outer": {2, 8}, "inner": {3, 5}},
	}, {
		name: "a label's rank",
		in: Graph{
			Nodes: []Node{n("A", ""), n("B", ""), n("C", "")},
			Edges: []Edge{labeled, e("L1", "B", "C", 1), e("L2", "A", "C", 1)},
		},
		ranks:  map[string]int{"A": 0, "B": 2, "C": 4},
		labels: map[string]float64{"L0": 1},
	}} {
		g := rankGraph(c.in, &counter{})
		ranks, clusters, labels := map[string]int{}, map[string][2]int{}, map[string]float64{}
		for _, nd := range c.in.Nodes {
			l := g.node(nd.ID)
			if hasChildren(g, nd.ID) {
				clusters[nd.ID] = [2]int{l.minRank, l.maxRank}
			} else {
				ranks[nd.ID] = l.rank
			}
		}
		var reversed []string
		for _, k := range g.edgeList() {
			l, name := g.edgeOf(k), k.name
			if l.reversed {
				name = l.forwardName
				reversed = append(reversed, name)
			}
			if l.hasLabelRank {
				labels[name] = l.labelRank
			}
		}
		if !maps.Equal(ranks, c.ranks) || !slices.Equal(reversed, c.reversed) ||
			!maps.Equal(clusters, orEmpty(c.clusters)) || !maps.Equal(labels, orEmpty(c.labels)) {
			t.Errorf("%s: ranks %v, reversed %v, subgraphs %v, labels %v; want %v, %v, %v, %v",
				c.name, ranks, reversed, clusters, labels, c.ranks, c.reversed, c.clusters, c.labels)
		}
	}
}

func orEmpty[V any](m map[string]V) map[string]V {
	if m == nil {
		return map[string]V{}
	}
	return m
}

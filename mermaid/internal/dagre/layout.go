package dagre

import "math"

// A Graph is what Mermaid hands dagre: its nodes, in the order it adds
// them, each with its parent, "" for none, and its edges, in the order it
// adds them, each named with Mermaid's edge id. MinLen is how many ranks
// an edge spans at least, 0 for dagre's default of 1; an edge with a label
// has the label's size. As in Mermaid's graph, no edge goes from a node to
// itself (Mermaid splits those up first) or ends at a subgraph.
type Graph struct {
	Nodes []Node
	Edges []Edge
}

type Node struct {
	ID, Parent    string
	Width, Height float64
}

type Edge struct {
	V, W, Name    string
	MinLen        int
	Width, Height float64
}

// rankGraph runs the first steps of dagre's layout (layout.js lines
// 26-36): it ranks the nodes, after turning around the edges that close
// cycles, and leaves each subgraph's border nodes and each label's rank.
// c names what it adds.
func rankGraph(in Graph, c *counter) *layoutGraph {
	g := build(in)
	for _, e := range g.edgeList() {
		g.edgeOf(e).minlen *= 2 // makeSpaceForEdgeLabels; labelpos is c
	}
	acyclic(g, c)
	n := nest(g, c)
	networkSimplex(asNonCompound(g))
	injectEdgeLabelProxies(g, c)
	removeEmptyRanks(g, n.factor)
	unnest(g, n)
	normalizeRanks(g)
	assignRankMinMax(g)
	removeEdgeLabelProxies(g)
	return g
}

// build is the graph dagre lays out (buildLayoutGraph, layout.js line
// 114): in's nodes in the order graphlib lists Mermaid's graph's, which
// has each parent from the first time it is named, then its edges.
func build(in Graph) *layoutGraph {
	src := newGraph[Node, struct{}](true, true)
	for _, n := range in.Nodes {
		src.setNode(n.ID, &n)
		if n.Parent != "" {
			src.setParent(n.ID, n.Parent)
		}
	}
	g := newGraph[node, edge](true, true)
	for _, v := range src.nodeNames() {
		n := src.node(v)
		g.setNode(v, &node{width: n.Width, height: n.Height})
		g.setParent(v, src.parentOf(v))
	}
	for _, e := range in.Edges {
		minlen := e.MinLen
		if minlen == 0 {
			minlen = 1
		}
		g.setEdge(e.V, e.W, e.Name, &edge{minlen: minlen, weight: 1, width: e.Width, height: e.Height})
	}
	return g
}

// asNonCompound is g without its subgraphs' nodes, sharing g's labels
// (util.js line 55).
func asNonCompound(g *layoutGraph) *layoutGraph {
	s := newGraph[node, edge](true, false)
	for _, v := range g.nodeNames() {
		if !hasChildren(g, v) {
			s.setNode(v, g.node(v))
		}
	}
	for _, e := range g.edgeList() {
		s.setEdge(e.v, e.w, e.name, g.edgeOf(e))
	}
	return s
}

func hasChildren(g *layoutGraph, v string) bool { return len(g.childrenOf(v)) > 0 }

// injectEdgeLabelProxies adds, for each edge with a label, a node at the
// rank halfway along it, which keeps that rank from being removed as
// empty (layout.js line 169).
func injectEdgeLabelProxies(g *layoutGraph, c *counter) {
	for _, e := range g.edgeList() {
		l := g.edgeOf(e)
		if l.width == 0 || l.height == 0 {
			continue
		}
		v, w := float64(g.node(e.v).rank), float64(g.node(e.w).rank)
		addDummyNode(g, c, "edge-proxy", &node{proxyOf: e, proxyRank: (w-v)/2 + v}, "_ep")
	}
}

// removeEmptyRanks closes up the empty ranks the nesting graph left
// between nodes' ranks, those that are not a multiple of factor from the
// first (util.js line 165). A proxy at a half rank is in no rank, as dagre
// files it under a key that is not an array index.
func removeEmptyRanks(g *layoutGraph, factor int) {
	offset := math.Inf(1)
	for _, v := range ranked(g) {
		offset = min(offset, rankOf(g.node(v)))
	}
	var layers [][]string
	for _, v := range ranked(g) {
		r := rankOf(g.node(v)) - offset
		if r != math.Trunc(r) {
			continue
		}
		for len(layers) <= int(r) {
			layers = append(layers, nil)
		}
		layers[int(r)] = append(layers[int(r)], v)
	}
	delta := 0
	for i, vs := range layers {
		if vs == nil && i%factor != 0 {
			delta--
			continue
		}
		for _, v := range vs {
			shift(g.node(v), float64(delta))
		}
	}
}

// normalizeRanks moves the ranks so that the least is 0 (util.js line
// 151).
func normalizeRanks(g *layoutGraph) {
	least := math.Inf(1)
	for _, v := range ranked(g) {
		least = min(least, rankOf(g.node(v)))
	}
	for _, v := range ranked(g) {
		shift(g.node(v), -least)
	}
}

// ranked are the nodes that have a rank: all but the subgraphs'.
func ranked(g *layoutGraph) []string {
	var out []string
	for _, v := range g.nodeNames() {
		if !hasChildren(g, v) {
			out = append(out, v)
		}
	}
	return out
}

func rankOf(n *node) float64 {
	if n.dummy == "edge-proxy" {
		return n.proxyRank
	}
	return float64(n.rank)
}

func shift(n *node, d float64) {
	if n.dummy == "edge-proxy" {
		n.proxyRank += d
		return
	}
	n.rank += int(d)
}

// assignRankMinMax gives each subgraph the ranks of its border nodes
// (layout.js line 181).
func assignRankMinMax(g *layoutGraph) {
	for _, v := range g.nodeNames() {
		if n := g.node(v); n.borderTop != "" {
			n.minRank, n.maxRank = g.node(n.borderTop).rank, g.node(n.borderBottom).rank
		}
	}
}

// removeEdgeLabelProxies gives each edge with a label its proxy's rank and
// removes the proxy (layout.js line 195).
func removeEdgeLabelProxies(g *layoutGraph) {
	for _, v := range g.nodeNames() {
		if n := g.node(v); n.dummy == "edge-proxy" {
			e := g.edgeOf(n.proxyOf)
			e.labelRank, e.hasLabelRank = n.proxyRank, true
			g.removeNode(v)
		}
	}
}

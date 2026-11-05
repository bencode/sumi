package dagre

import "strconv"

// A node is what dagre keeps on a node of the graph it lays out.
type node struct {
	width, height float64
	rank          int
	// order is the node's place in its rank, from 0.
	order int
	// dummy is the kind of node dagre added: "root", "border", "edge-proxy",
	// "edge" or "edge-label"; "" for the caller's.
	dummy string
	// borderTop and borderBottom are a subgraph's border nodes, and
	// minRank and maxRank their ranks.
	borderTop, borderBottom string
	minRank, maxRank        int
	// proxyOf is the edge an edge-proxy stands for, and proxyRank the rank
	// it stands at: halfway along the edge, so it may be a half, which
	// dagre then leaves out of its layers (injectEdgeLabelProxies,
	// layout.js line 169).
	proxyOf   edgeKey
	proxyRank float64
	// edgeLabel and edgeObj are the edge a dummy of a split edge stands for.
	edgeLabel *edge
	edgeObj   edgeKey
	// borderLeft and borderRight are a subgraph's left and right border
	// nodes, one for each rank from minRank on.
	borderLeft, borderRight []string
	// borderType is a border node's side: "borderLeft" or "borderRight";
	// "" for a top or bottom one.
	borderType string
}

// An edge is what dagre keeps on an edge of the graph it lays out.
type edge struct {
	minlen, weight int
	width, height  float64
	// reversed is whether acyclic turned the edge around, and forwardName
	// the name it had.
	reversed    bool
	forwardName string
	nestingEdge bool
	// labelRank is the rank the edge's label goes in, when it has one.
	labelRank    float64
	hasLabelRank bool
}

type layoutGraph = graph[node, edge]

// A counter names the nodes and edges dagre adds, as lodash's
// _.uniqueId does: the prefix and a number one higher each time, the
// first being 1. lodash counts for the whole page; a counter counts for
// what it is handed to.
type counter struct{ n int }

func (c *counter) next(prefix string) string {
	c.n++
	return prefix + strconv.Itoa(c.n)
}

// addDummyNode adds a node labeled label, of kind dummy, named prefix and
// the first number from c that no node has (util.js line 24).
func addDummyNode(g *layoutGraph, c *counter, dummy string, label *node, prefix string) string {
	v := c.next(prefix)
	for g.hasNode(v) {
		v = c.next(prefix)
	}
	label.dummy = dummy
	g.setNode(v, label)
	return v
}

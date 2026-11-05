package dagre

// addBorderSegments gives each subgraph a left and a right border node in
// each of its ranks, those of a side chained rank to rank, children first
// (add-border-segments.js line 6).
func addBorderSegments(g *layoutGraph, c *counter) {
	var dfs func(v string)
	dfs = func(v string) {
		for _, w := range g.childrenOf(v) {
			dfs(w)
		}
		n := g.node(v)
		if n.borderTop == "" {
			return
		}
		for rank := n.minRank; rank <= n.maxRank; rank++ {
			n.borderLeft = addBorderNode(g, c, n.borderLeft, "borderLeft", "_bl", v, rank)
			n.borderRight = addBorderNode(g, c, n.borderRight, "borderRight", "_br", v, rank)
		}
	}
	for _, v := range g.childrenOf("") {
		dfs(v)
	}
}

// addBorderNode adds side's node in rank to subgraph sg, and returns side
// with it (add-border-segments.js line 27).
func addBorderNode(g *layoutGraph, c *counter, side []string, borderType, prefix, sg string, rank int) []string {
	v := addDummyNode(g, c, "border", &node{rank: rank, borderType: borderType}, prefix)
	g.setParent(v, sg)
	if len(side) > 0 {
		g.setEdge(side[len(side)-1], v, "", &edge{weight: 1})
	}
	return append(side, v)
}

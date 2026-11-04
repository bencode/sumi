package dagre

// A nesting is what nest added: its root, and how many ranks a node's
// rank now stands for.
type nesting struct {
	root   string
	factor int
}

// nest adds a nesting graph (nesting-graph.js lines 29-98, after Sander,
// "Layout of Compound Directed Graphs"): a root with an edge to every
// node, so that the graph is connected, and for each subgraph a top and a
// bottom border node, edges that hold its nodes between them, and an edge
// from the root to the top of each outermost one. Edges' minlen is
// multiplied by factor, 2 * the subgraphs' depth + 1, so that border nodes
// get ranks of their own between the nodes'.
func nest(g *layoutGraph, c *counter) nesting {
	root := addDummyNode(g, c, "root", &node{}, "_root")
	depths := treeDepths(g)
	height := 0
	for _, d := range depths {
		height = max(height, d)
	}
	height--
	factor := 2*height + 1
	for _, e := range g.edgeList() {
		g.edgeOf(e).minlen *= factor
	}
	weight := 1
	for _, e := range g.edgeList() {
		weight += g.edgeOf(e).weight
	}
	n := nester{g: g, c: c, root: root, factor: factor, weight: weight, height: height, depths: depths}
	for _, v := range g.childrenOf("") {
		n.dfs(v)
	}
	return nesting{root: root, factor: factor}
}

// A nester is what nest's depth-first walk over the subgraphs needs
// (nesting-graph.js line 55).
type nester struct {
	g                      *layoutGraph
	c                      *counter
	root                   string
	factor, weight, height int
	depths                 map[string]int
}

func (n nester) dfs(v string) {
	g := n.g
	children := g.childrenOf(v)
	if len(children) == 0 {
		if v != n.root {
			g.setEdge(n.root, v, "", &edge{weight: 0, minlen: n.factor})
		}
		return
	}
	bt := addDummyNode(g, n.c, "border", &node{}, "_bt")
	bb := addDummyNode(g, n.c, "border", &node{}, "_bb")
	label := g.node(v)
	g.setParent(bt, v)
	label.borderTop = bt
	g.setParent(bb, v)
	label.borderBottom = bb
	for _, child := range children {
		n.dfs(child)
		cn := g.node(child)
		childTop, childBottom, w := child, child, 2*n.weight
		if cn.borderTop != "" {
			childTop, childBottom, w = cn.borderTop, cn.borderBottom, n.weight
		}
		minlen := n.height - n.depths[v] + 1
		if childTop != childBottom {
			minlen = 1
		}
		g.setEdge(bt, childTop, "", &edge{weight: w, minlen: minlen, nestingEdge: true})
		g.setEdge(childBottom, bb, "", &edge{weight: w, minlen: minlen, nestingEdge: true})
	}
	if g.parentOf(v) == "" {
		g.setEdge(n.root, bt, "", &edge{weight: 0, minlen: n.height + n.depths[v]})
	}
}

// treeDepths are how deep each node is: 1 at the top level.
func treeDepths(g *layoutGraph) map[string]int {
	depths := map[string]int{}
	var dfs func(v string, depth int)
	dfs = func(v string, depth int) {
		for _, c := range g.childrenOf(v) {
			dfs(c, depth+1)
		}
		depths[v] = depth
	}
	for _, v := range g.childrenOf("") {
		dfs(v, 1)
	}
	return depths
}

// unnest removes the nesting graph's root and edges, keeping the border
// nodes (nesting-graph.js line 127).
func unnest(g *layoutGraph, n nesting) {
	g.removeNode(n.root)
	for _, e := range g.edgeList() {
		if g.edgeOf(e).nestingEdge {
			g.removeEdgeKey(e)
		}
	}
}

package dagre

// networkSimplex ranks g's nodes so that the edges' lengths, each times
// its weight, add up to the least they can, every edge at least its
// minlen long (network-simplex.js lines 50-62, after Gansner et al., "A
// Technique for Drawing Directed Graphs"). It starts from the longest-path
// ranks and a tight spanning tree, then swaps a tree edge with a negative
// cut value for the edge across with the least slack, until no cut value
// is negative. g must be a connected DAG; its labels get the ranks.
func networkSimplex(g *layoutGraph) {
	g = simplify(g)
	longestPath(g)
	t := feasibleTree(g)
	initLowLim(t)
	initCutValues(t, g)
	for e, ok := leaveEdge(t); ok; e, ok = leaveEdge(t) {
		exchangeEdges(t, g, e, enterEdge(t, g, e))
	}
}

// simplify is g with each set of edges between the same two nodes made
// one: their weights added, their longest minlen. The nodes' labels are
// g's (util.js line 39).
func simplify(g *layoutGraph) *layoutGraph {
	s := newGraph[node, edge](true, false)
	for _, v := range g.nodeNames() {
		s.setNode(v, g.node(v))
	}
	for _, e := range g.edgeList() {
		weight, minlen := 0, 1
		if l := s.edge(e.v, e.w, ""); l != nil {
			weight, minlen = l.weight, l.minlen
		}
		l := g.edgeOf(e)
		s.setEdge(e.v, e.w, "", &edge{weight: weight + l.weight, minlen: max(minlen, l.minlen)})
	}
	return s
}

// initLowLim numbers t's nodes from its first, depth first: lim in the
// order the walk leaves them, low the least lim below each, and parent
// the node it came from (network-simplex.js lines 121-149).
func initLowLim(t *tree) {
	visited := map[string]bool{}
	var dfs func(v, parent string, next int) int
	dfs = func(v, parent string, next int) int {
		low := next
		visited[v] = true
		for _, w := range t.neighbors(v) {
			if !visited[w] {
				next = dfs(w, v, next)
			}
		}
		label := t.node(v)
		label.low, label.lim, label.parent = low, next, parent
		return next + 1
	}
	dfs(t.nodeNames()[0], "", 1)
}

// initCutValues sets the cut value of every tree edge, children before
// their parents (network-simplex.js lines 67-79).
func initCutValues(t *tree, g *layoutGraph) {
	vs := walk(t, t.nodeNames(), true)
	for _, child := range vs[:len(vs)-1] {
		t.edge(child, t.node(child).parent, "").cutvalue = cutValue(t, g, child)
	}
}

// cutValue is the cut value of the tree edge between child and its
// parent: the weight of the graph's edges from the side holding the
// edge's tail to the other, less those back. It builds on the cut values
// of child's other tree edges (network-simplex.js line 85).
func cutValue(t *tree, g *layoutGraph, child string) int {
	parent := t.node(child).parent
	childIsTail := true
	e := g.edge(child, parent, "")
	if e == nil {
		childIsTail = false
		e = g.edge(parent, child, "")
	}
	cv := e.weight
	for _, ge := range g.nodeEdges(child) {
		isOut := ge.v == child
		other := ge.v
		if isOut {
			other = ge.w
		}
		if other == parent {
			continue
		}
		sign := -1
		if isOut == childIsTail {
			sign = 1
		}
		cv += sign * g.edgeOf(ge).weight
		if te := t.edge(child, other, ""); te != nil {
			cv -= sign * te.cutvalue
		}
	}
	return cv
}

// leaveEdge is the first tree edge with a negative cut value.
func leaveEdge(t *tree) (edgeKey, bool) {
	for _, e := range t.edgeList() {
		if t.edgeOf(e).cutvalue < 0 {
			return e, true
		}
	}
	return edgeKey{}, false
}

// enterEdge is the graph edge to swap in for the tree edge e: of those
// crossing from the side of e's head to the side of its tail, the first
// with the least slack (network-simplex.js line 157).
func enterEdge(t *tree, g *layoutGraph, e edgeKey) edgeKey {
	v, w := e.v, e.w
	if !g.hasEdge(v, w, "") {
		v, w = w, v
	}
	vl, wl := t.node(v), t.node(w)
	tail, flip := vl, false
	if vl.lim > wl.lim {
		tail, flip = wl, true
	}
	return leastSlack(g, func(c edgeKey) bool {
		return flip == isDescendant(t.node(c.v), tail) && flip != isDescendant(t.node(c.w), tail)
	})
}

// exchangeEdges swaps the tree edge e for the graph edge f, then numbers
// the tree, its cut values and the ranks afresh.
func exchangeEdges(t *tree, g *layoutGraph, e, f edgeKey) {
	t.removeEdgeKey(e)
	t.setEdge(f.v, f.w, "", &treeEdge{})
	initLowLim(t)
	initCutValues(t, g)
	updateRanks(t, g)
}

// updateRanks ranks each node from its parent in the tree, along the tight
// edge between them, from the root down (network-simplex.js line 203).
// dagre's root is the first node whose graph label has no parent; no
// label has one, so it is the first node.
func updateRanks(t *tree, g *layoutGraph) {
	vs := walk(t, t.nodeNames()[:1], false)
	for _, v := range vs[1:] {
		parent := t.node(v).parent
		if e := g.edge(v, parent, ""); e != nil {
			g.node(v).rank = g.node(parent).rank - e.minlen
		} else {
			g.node(v).rank = g.node(parent).rank + g.edge(parent, v, "").minlen
		}
	}
}

// isDescendant is whether v is below root in the tree, by their numbers.
func isDescendant(v, root *treeNode) bool {
	return root.low <= v.lim && v.lim <= root.lim
}

// walk visits t depth first from each of roots in turn, along neighbors,
// and lists the nodes before their children or, post, after them
// (graphlib/alg/dfs.js).
func walk(t *tree, roots []string, post bool) []string {
	var out []string
	visited := map[string]bool{}
	var dfs func(v string)
	dfs = func(v string) {
		if visited[v] {
			return
		}
		visited[v] = true
		if !post {
			out = append(out, v)
		}
		for _, w := range t.neighbors(v) {
			dfs(w)
		}
		if post {
			out = append(out, v)
		}
	}
	for _, v := range roots {
		dfs(v)
	}
	return out
}

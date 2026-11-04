package dagre

// A treeNode and a treeEdge are what network simplex keeps on its
// spanning tree: each node's low and lim numbers and parent, and each
// edge's cut value.
type (
	treeNode struct {
		low, lim int
		parent   string
	}
	treeEdge struct{ cutvalue int }
	tree     = graph[treeNode, treeEdge]
)

// longestPath ranks every node as low as it goes: the lowest of its
// successors' ranks less the edges' minlen, or 0 with none, from each
// source in turn (rank/util.js line 26).
func longestPath(g *layoutGraph) {
	visited := map[string]bool{}
	var dfs func(v string) int
	dfs = func(v string) int {
		label := g.node(v)
		if visited[v] {
			return label.rank
		}
		visited[v] = true
		rank, seen := 0, false
		for _, e := range g.outEdges(v) {
			r := dfs(e.w) - g.edgeOf(e).minlen
			if !seen || r < rank {
				rank, seen = r, true
			}
		}
		label.rank = rank
		return rank
	}
	for _, v := range g.sources() {
		dfs(v)
	}
}

// slack is how much longer e is than it has to be (rank/util.js line 61).
func slack(g *layoutGraph, e edgeKey) int {
	return g.node(e.w).rank - g.node(e.v).rank - g.edgeOf(e).minlen
}

// feasibleTree grows a spanning tree of tight edges, ones with no slack,
// from the first node, moving the whole tree by the smallest slack of an
// edge leaving it whenever it can grow no more (feasible-tree.js lines
// 32-88). g must be a connected DAG.
func feasibleTree(g *layoutGraph) *tree {
	t := newGraph[treeNode, treeEdge](false, false)
	t.setNode(g.nodeNames()[0], nil)
	for tightTree(t, g) < g.nodeCount() {
		e := leastSlack(g, func(e edgeKey) bool { return t.hasNode(e.v) != t.hasNode(e.w) })
		delta := slack(g, e)
		if !t.hasNode(e.v) {
			delta = -delta
		}
		for _, v := range t.nodeNames() {
			g.node(v).rank += delta
		}
	}
	return t
}

// tightTree adds to t every node a tight edge joins it to, and returns how
// many nodes t has.
func tightTree(t *tree, g *layoutGraph) int {
	var dfs func(v string)
	dfs = func(v string) {
		for _, e := range g.nodeEdges(v) {
			w := e.v
			if v == e.v {
				w = e.w
			}
			if !t.hasNode(w) && slack(g, e) == 0 {
				t.setNode(w, nil)
				t.setEdge(v, w, "", &treeEdge{})
				dfs(w)
			}
		}
	}
	for _, v := range t.nodeNames() {
		dfs(v)
	}
	return t.nodeCount()
}

// leastSlack is the first of g's edges that keep keeps with the least
// slack, as lodash's _.minBy finds it (findMinSlackEdge, feasible-tree.js
// line 75, and enterEdge).
func leastSlack(g *layoutGraph, keep func(edgeKey) bool) edgeKey {
	var best edgeKey
	least, found := 0, false
	for _, e := range g.edgeList() {
		if !keep(e) {
			continue
		}
		if s := slack(g, e); !found || s < least {
			best, least, found = e, s, true
		}
	}
	return best
}

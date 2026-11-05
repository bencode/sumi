package dagre

import "slices"

// normalize splits each edge that spans more than one rank into edges
// that span one, with a dummy node in each rank between. It returns each
// split edge's first dummy (normalize.js line 27).
func normalize(g *layoutGraph, c *counter) []string {
	var chains []string
	for _, e := range g.edgeList() {
		if first := normalizeEdge(g, c, e); first != "" {
			chains = append(chains, first)
		}
	}
	return chains
}

// normalizeEdge splits e, and returns its first dummy, "" for an edge
// that spans one rank. The dummy in the rank of e's label takes the
// label's size (normalize.js line 37).
func normalizeEdge(g *layoutGraph, c *counter, e edgeKey) string {
	from, to := g.node(e.v).rank, g.node(e.w).rank
	if to == from+1 {
		return ""
	}
	l := g.edgeOf(e)
	g.removeEdgeKey(e)
	first, v := "", e.v
	for rank := from + 1; rank < to; rank++ {
		label := &node{rank: rank, edgeLabel: l, edgeObj: e}
		dummy := addDummyNode(g, c, "edge", label, "_d")
		if l.hasLabelRank && float64(rank) == l.labelRank {
			label.width, label.height, label.dummy = l.width, l.height, "edge-label"
		}
		g.setEdge(v, dummy, e.name, &edge{weight: l.weight})
		if first == "" {
			first = dummy
		}
		v = dummy
	}
	g.setEdge(v, e.w, e.name, &edge{weight: l.weight})
	return first
}

// denormalize puts each split edge back, with its dummies' places as its
// bends and its label's dummy's as its label's (normalize.js line 90).
func denormalize(g *layoutGraph, chains []string) {
	for _, v := range chains {
		n := g.node(v)
		l, e := n.edgeLabel, n.edgeObj
		g.setEdge(e.v, e.w, e.name, l)
		for n.dummy != "" {
			w := g.successors(v)[0]
			g.removeNode(v)
			l.points = append(l.points, Point{n.x, n.y})
			if n.dummy == "edge-label" {
				l.x, l.y, l.hasXY = n.x, n.y, true
				l.width, l.height = n.width, n.height
			}
			v, n = w, g.node(w)
		}
	}
}

// parentDummyChains puts each split edge's dummies in subgraphs. The path
// from the edge's start up to the two ends' lowest common ancestor and
// down to its end names the subgraphs the edge goes through; each dummy
// takes the one that holds its rank, going up until the ancestor and down
// from there (parent-dummy-chains.js line 5).
func parentDummyChains(g *layoutGraph, chains []string) {
	nums := postorder(g)
	for _, v := range chains {
		e := g.node(v).edgeObj
		path, lca := findPath(g, nums, e.v, e.w)
		i, ascending := 0, true
		for v != e.w {
			rank := g.node(v).rank
			if ascending {
				for path[i] != lca && g.node(path[i]).maxRank < rank {
					i++
				}
				ascending = path[i] != lca
			}
			if !ascending {
				for i < len(path)-1 && g.node(path[i+1]).minRank <= rank {
					i++
				}
			}
			g.setParent(v, path[i])
			v = g.successors(v)[0]
		}
	}
}

// A span is the postorder numbers of a node's subtree: from low, its first
// descendant's, to lim, its own.
type span struct{ low, lim int }

// findPath is the path from v's parent to w's through their lowest common
// ancestor, and the ancestor; "" stands for the top level
// (parent-dummy-chains.js line 49).
func findPath(g *layoutGraph, nums map[string]span, v, w string) ([]string, string) {
	low, lim := min(nums[v].low, nums[w].low), max(nums[v].lim, nums[w].lim)
	var path []string
	lca := v
	for {
		lca = g.parentOf(lca)
		path = append(path, lca)
		if lca == "" || nums[lca].low <= low && lim <= nums[lca].lim {
			break
		}
	}
	var down []string
	for p := g.parentOf(w); p != lca; p = g.parentOf(p) {
		down = append(down, p)
	}
	slices.Reverse(down)
	return append(path, down...), lca
}

// postorder numbers g's nodes by subgraph (parent-dummy-chains.js line 74).
func postorder(g *layoutGraph) map[string]span {
	nums := map[string]span{}
	lim := 0
	var dfs func(v string)
	dfs = func(v string) {
		low := lim
		for _, w := range g.childrenOf(v) {
			dfs(w)
		}
		nums[v] = span{low, lim}
		lim++
	}
	for _, v := range g.childrenOf("") {
		dfs(v)
	}
	return nums
}

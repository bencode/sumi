package dagre

// acyclic turns around the edges that close a cycle, so that the graph
// has none: those a depth-first search, from each node in order, finds
// going back to a node on its path. Each is removed and added back the
// other way, named afresh from c, and marked reversed with the name it
// had (acyclic.js lines 6-46). Mermaid sets no acyclicer, so dagre's
// greedy alternative is left out.
func acyclic(g *layoutGraph, c *counter) {
	var fas []edgeKey
	visited, stack := map[string]bool{}, map[string]bool{}
	var dfs func(v string)
	dfs = func(v string) {
		if visited[v] {
			return
		}
		visited[v], stack[v] = true, true
		for _, e := range g.outEdges(v) {
			if stack[e.w] {
				fas = append(fas, e)
			} else {
				dfs(e.w)
			}
		}
		delete(stack, v)
	}
	for _, v := range g.nodeNames() {
		dfs(v)
	}
	for _, e := range fas {
		label := g.edgeOf(e)
		g.removeEdgeKey(e)
		label.forwardName, label.reversed = e.name, true
		g.setEdge(e.w, e.v, c.next("rev"), label)
	}
}

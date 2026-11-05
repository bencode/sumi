package dagre

import "math"

// edgeSep is dagre's default, which Mermaid leaves.
const edgeSep = 20

type blockGraph = graph[struct{}, float64]

// horizontalCompaction is each node's x, its block's. Unlike Brandes and
// Köpf, dagre sweeps a graph of the blocks twice: first to the least xs,
// then to the greatest that keep their distances, a subgraph's far border
// staying put (bk.js line 223). reverseSep is whether layering runs right
// to left.
func horizontalCompaction(g *layoutGraph, layering [][]string, root, align map[string]string, reverseSep bool, nodeSep float64) map[string]float64 {
	xs := map[string]float64{}
	blocks := buildBlockGraph(g, layering, root, nodeSep)
	borderType := "borderRight"
	if reverseSep {
		borderType = "borderLeft"
	}
	iterate(blocks, blocks.predecessors, func(v string) {
		x := 0.0
		for _, e := range blocks.inEdges(v) {
			x = max(x, xs[e.v]+*blocks.edgeOf(e))
		}
		xs[v] = x
	})
	iterate(blocks, blocks.successors, func(v string) {
		least := math.Inf(1)
		for _, e := range blocks.outEdges(v) {
			least = min(least, xs[e.w]-*blocks.edgeOf(e))
		}
		if !math.IsInf(least, 1) && g.node(v).borderType != borderType {
			xs[v] = max(xs[v], least)
		}
	})
	for _, v := range align { // every node is in align's values
		xs[v] = xs[root[v]]
	}
	return xs
}

// iterate sets each block after those next lists, working through a stack
// of them (bk.js line 234).
func iterate(blocks *blockGraph, next func(string) []string, set func(string)) {
	stack := blocks.nodeNames()
	visited := map[string]bool{}
	for len(stack) > 0 {
		v := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if visited[v] {
			set(v)
			continue
		}
		visited[v] = true
		stack = append(append(stack, v), next(v)...)
	}
}

// buildBlockGraph is a graph of the blocks, with an edge between those of
// neighbors in a rank, carrying the greatest distance they must keep
// (bk.js line 281).
func buildBlockGraph(g *layoutGraph, layering [][]string, root map[string]string, nodeSep float64) *blockGraph {
	blocks := newGraph[struct{}, float64](true, false)
	for _, layer := range layering {
		u := ""
		for _, v := range layer {
			blocks.setNode(root[v], nil)
			if u != "" {
				s := sep(g, v, u, nodeSep)
				if prev := blocks.edge(root[u], root[v], ""); prev != nil {
					s = max(s, *prev)
				}
				blocks.setEdge(root[u], root[v], "", &s)
			}
			u = v
		}
	}
	return blocks
}

// sep is how far apart the centers of v and w must be: their half widths
// and half of each one's gap (bk.js line 396). With labels centered,
// reverseSep changes nothing here.
func sep(g *layoutGraph, v, w string, nodeSep float64) float64 {
	gap := func(n *node) float64 {
		if n.dummy != "" {
			return edgeSep
		}
		return nodeSep
	}
	vl, wl := g.node(v), g.node(w)
	return vl.width/2 + gap(vl)/2 + gap(wl)/2 + wl.width/2
}

package dagre

import (
	"math"
	"slices"
)

// orderGraph runs dagre's layout up to the order of each rank (layout.js
// lines 26-40). c names what it adds.
func orderGraph(in Graph, c *counter) *layoutGraph {
	g := rankGraph(in, c)
	parentDummyChains(g, normalize(g, c))
	addBorderSegments(g, c)
	order(g, c)
	return g
}

// order gives each node its place in its rank. From a first order it
// sorts the ranks in turn, each by the one before, going up and down the
// ranks and leaning left and right, and keeps the order with the fewest
// crossings. It stops three sweeps after the last that did better
// (order/index.js line 27).
func order(g *layoutGraph, c *counter) {
	last := maxRank(g)
	var down, up []layer
	for rank := 1; rank <= last; rank++ {
		down = append(down, buildLayerGraph(g, c, rank, g.inEdges))
	}
	for rank := last - 1; rank >= 0; rank-- {
		up = append(up, buildLayerGraph(g, c, rank, g.outEdges))
	}
	assignOrder(g, initOrder(g))
	var best [][]string
	bestCC := math.MaxInt
	for i, lastBest := 0, 0; lastBest < 4; i, lastBest = i+1, lastBest+1 {
		layers := up
		if i%2 == 1 {
			layers = down
		}
		sweep(layers, i%4 >= 2)
		layering := layerMatrix(g)
		if cc := crossCount(g, layering); cc < bestCC {
			lastBest, best, bestCC = 0, layering, cc
		}
	}
	assignOrder(g, best)
}

// sweep sorts layers in turn. What one finds of subgraphs' order holds for
// those after it (order/index.js line 59).
func sweep(layers []layer, biasRight bool) {
	cg := newGraph[struct{}, struct{}](true, false)
	for _, l := range layers {
		vs := sortSubgraph(l.g, l.root, cg, biasRight).vs
		for i, v := range vs {
			l.g.node(v).n.order = i
		}
		addSubgraphConstraints(l.g, cg, vs)
	}
}

// assignOrder gives each node its place in layering (order/index.js line
// 71).
func assignOrder(g *layoutGraph, layering [][]string) {
	for _, vs := range layering {
		for i, v := range vs {
			g.node(v).order = i
		}
	}
}

// initOrder is the first order: each rank's nodes as a depth-first search
// reaches them, starting from the nodes in order of rank (init-order.js
// line 14).
func initOrder(g *layoutGraph) [][]string {
	layers := make([][]string, maxRank(g)+1)
	visited := map[string]bool{}
	var dfs func(v string)
	dfs = func(v string) {
		if visited[v] {
			return
		}
		visited[v] = true
		rank := g.node(v).rank
		layers[rank] = append(layers[rank], v)
		for _, w := range g.successors(v) {
			dfs(w)
		}
	}
	vs := ranked(g)
	slices.SortStableFunc(vs, func(a, b string) int { return g.node(a).rank - g.node(b).rank })
	for _, v := range vs {
		dfs(v)
	}
	return layers
}

// layerMatrix is each rank's nodes in order (buildLayerMatrix, util.js
// line 133).
func layerMatrix(g *layoutGraph) [][]string {
	layers := make([][]string, maxRank(g)+1)
	for _, v := range ranked(g) {
		rank := g.node(v).rank
		layers[rank] = append(layers[rank], v)
	}
	for _, vs := range layers {
		slices.SortFunc(vs, func(a, b string) int { return g.node(a).order - g.node(b).order })
	}
	return layers
}

// maxRank is the last rank, -1 for a graph without nodes (util.js line
// 207).
func maxRank(g *layoutGraph) int {
	last := -1
	for _, v := range ranked(g) {
		last = max(last, g.node(v).rank)
	}
	return last
}

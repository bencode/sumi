// SPDX-License-Identifier: EPL-2.0 OR GPL-3.0-or-later

package elk

import (
	"math"
	"slices"
)

// iterationFactor is ELK's default thoroughness, 7, times ITER_LIMIT_FACTOR
// (p2layers/NetworkSimplexLayerer.java line 229).
const iterationFactor = 7 * 4

// assignLayers moves all nodes out of Layerless into layers, solving each
// connected component on its own. All components start at the first layer, and
// each one sees how full the others left it (process,
// p2layers/NetworkSimplexLayerer.java line 244).
func assignLayers(g *Graph) {
	parts := components(g.Layerless)
	var previous []int
	for _, part := range parts {
		limit := iterationFactor * int(math.Sqrt(float64(len(part))))
		order, layers := networkSimplex(len(part), simplexEdges(part), limit, previous)
		for _, i := range order {
			for len(g.Layers) <= layers[i] {
				g.Layers = append(g.Layers, &Layer{})
			}
			l := g.Layers[layers[i]]
			l.Nodes = append(l.Nodes, part[i])
		}
		previous = layerSizes(g)
	}
	g.Layerless = nil
}

func layerSizes(g *Graph) []int {
	sizes := make([]int, len(g.Layers))
	for i, l := range g.Layers {
		sizes[i] = len(l.Nodes)
	}
	return sizes
}

// components gives all's connected components, nodes in the order found. A
// component goes first if it is larger than the first one so far, and last
// otherwise, so the result is not fully sorted (connectedComponents,
// NetworkSimplexLayerer.java line 108).
func components(all []*Node) [][]*Node {
	visited := map[*Node]bool{}
	var parts [][]*Node
	for _, n := range all {
		if visited[n] {
			continue
		}
		part := collect(n, visited, nil)
		if len(parts) == 0 || len(parts[0]) < len(part) {
			parts = slices.Insert(parts, 0, part)
		} else {
			parts = append(parts, part)
		}
	}
	return parts
}

// collect adds to part what n reaches, through each port's incoming edges
// before its outgoing ones (connectedComponentsDFS,
// NetworkSimplexLayerer.java line 153).
func collect(n *Node, visited map[*Node]bool, part []*Node) []*Node {
	visited[n] = true
	part = append(part, n)
	for _, p := range n.Ports {
		for _, e := range p.In {
			if next := e.Source.Owner; !visited[next] {
				part = collect(next, visited, part)
			}
		}
		for _, e := range p.Out {
			if next := e.Target.Owner; !visited[next] {
				part = collect(next, visited, part)
			}
		}
	}
	return part
}

// simplexEdges gives part's edges as indexes into part (initialize,
// NetworkSimplexLayerer.java line 182).
func simplexEdges(part []*Node) [][2]int {
	index := make(map[*Node]int, len(part))
	for i, n := range part {
		index[n] = i
	}
	var all [][2]int
	for i, n := range part {
		for _, e := range outgoing(n) {
			all = append(all, [2]int{i, index[e.Target.Owner]})
		}
	}
	return all
}

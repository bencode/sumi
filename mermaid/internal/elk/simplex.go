// SPDX-License-Identifier: EPL-2.0 OR GPL-3.0-or-later

package elk

import (
	"math"
	"slices"
)

// subtreeThreshold is the node count from which a run takes the graph's trees
// off first (NetworkSimplex.java line 62, org.eclipse.elk.alg.common).
const subtreeThreshold = 40

// networkSimplex gives each of nodeCount connected nodes a layer so that every
// edge goes at least one layer forward, with the edges as short as possible in
// total. It stops after limit exchanges. Mermaid sets neither edge weights nor
// lengths, so all count 1. previous holds the sizes of the layers already in
// use. order is the nodes in the order the run leaves them, which is not the
// order they came in (execute, NetworkSimplex.java line 274).
func networkSimplex(nodeCount int, edges [][2]int, limit int, previous []int) (order, layers []int) {
	s := newSimplex(nodeCount, edges)
	var taken []leaf
	if nodeCount >= subtreeThreshold {
		taken = s.removeSubtrees()
	}
	s.initialize()
	s.feasibleTree()
	for i := 0; i < limit; i++ {
		leave := s.leaveEdge()
		if leave == nil {
			break
		}
		s.exchange(leave, s.enterEdge(leave))
	}
	s.reattachSubtrees(taken)
	s.balance(s.normalize(previous))

	order, layers = make([]int, nodeCount), make([]int, nodeCount)
	for i, n := range s.nodes {
		order[i], layers[n.id] = n.id, n.layer
	}
	return order, layers
}

// simplex is one run's state. It never leaves networkSimplex.
type simplex struct {
	nodes     []*simplexNode
	edges     []*simplexEdge
	tree      []*simplexEdge // in the order they joined the tree
	postOrder int
}

type simplexNode struct {
	id, layer    int
	in, out      []*simplexEdge
	inTree       bool
	po, lowestPo int
	unknown      []*simplexEdge // tree edges with no cut value yet
}

type simplexEdge struct {
	source, target *simplexNode
	inTree         bool
	visited        bool
	cut            int
}

// leaf is a node taken off the graph, with the edge that held it.
type leaf struct {
	node *simplexNode
	edge *simplexEdge
}

func newSimplex(nodeCount int, edges [][2]int) *simplex {
	s := &simplex{nodes: make([]*simplexNode, nodeCount)}
	for i := range s.nodes {
		s.nodes[i] = &simplexNode{id: i}
	}
	for _, e := range edges {
		source, target := s.nodes[e[0]], s.nodes[e[1]]
		edge := &simplexEdge{source: source, target: target}
		source.out = append(source.out, edge)
		target.in = append(target.in, edge)
	}
	return s
}

// initialize lists the edges of the nodes left (NetworkSimplex.java line 203).
func (s *simplex) initialize() {
	for _, n := range s.nodes {
		s.edges = append(s.edges, n.out...)
	}
	s.postOrder = 1
}

// connected gives n's edges, incoming first (getConnectedEdges, NNode.java line
// 107).
func connected(n *simplexNode) []*simplexEdge { return slices.Concat(n.in, n.out) }

func (e *simplexEdge) other(n *simplexNode) *simplexNode {
	if e.source == n {
		return e.target
	}
	return e.source
}

// removeSubtrees takes nodes with one edge off the graph until none is left.
// Their layers follow from their neighbors', so they cost the run nothing. A
// node keeps its own edge lists; only its neighbor forgets it
// (NetworkSimplex.java line 330).
func (s *simplex) removeSubtrees() []leaf {
	var taken []leaf
	var queue []*simplexNode
	for _, n := range s.nodes {
		if len(connected(n)) == 1 {
			queue = append(queue, n)
		}
	}
	for len(queue) > 0 {
		n := queue[0]
		queue = queue[1:]
		if len(connected(n)) == 0 {
			continue
		}
		edge := connected(n)[0]
		other := edge.other(n)
		if len(n.out) > 0 {
			other.in = without(other.in, edge)
		} else {
			other.out = without(other.out, edge)
		}
		if len(connected(other)) == 1 {
			queue = append(queue, other)
		}
		taken = append(taken, leaf{n, edge})
		s.nodes = without(s.nodes, n)
	}
	return taken
}

// reattachSubtrees puts taken back, last taken first, one layer from their
// neighbors (NetworkSimplex.java line 375).
func (s *simplex) reattachSubtrees(taken []leaf) {
	for _, l := range slices.Backward(taken) {
		placed := l.edge.other(l.node)
		if l.edge.target == l.node {
			placed.out = append(placed.out, l.edge)
			l.node.layer = placed.layer + 1
		} else {
			placed.in = append(placed.in, l.edge)
			l.node.layer = placed.layer - 1
		}
		s.nodes = append(s.nodes, l.node)
	}
}

// normalize moves the lowest layer to 0 and gives each layer's size, with the
// nodes already there (NetworkSimplex.java line 828).
func (s *simplex) normalize(previous []int) []int {
	lowest, highest := math.MaxInt, math.MinInt
	for _, n := range s.nodes {
		lowest, highest = min(lowest, n.layer), max(highest, n.layer)
	}
	filling := make([]int, highest-lowest+1)
	for _, n := range s.nodes {
		n.layer -= lowest
		filling[n.layer]++
	}
	for i := range min(len(previous), len(filling)) {
		filling[i] += previous[i]
	}
	return filling
}

// balance moves nodes with as many edges in as out to emptier layers, which
// leaves the total edge length as it was (NetworkSimplex.java line 866).
func (s *simplex) balance(filling []int) {
	for _, n := range s.nodes {
		if len(n.in) != len(n.out) {
			continue
		}
		layer := n.layer
		spanIn, spanOut := minimalSpan(n)
		for i := n.layer - spanIn + 1; i < n.layer+spanOut; i++ {
			if filling[i] < filling[layer] {
				layer = i
			}
		}
		if filling[layer] < filling[n.layer] {
			filling[n.layer]--
			filling[layer]++
			n.layer = layer
		}
	}
}

// minimalSpan gives the shortest of n's incoming and outgoing edges, -1 if it
// has none. As in ELK, an incoming edge that is not the shortest so far counts
// as an outgoing one, which can narrow the room balance sees
// (NetworkSimplex.java line 486).
func minimalSpan(n *simplexNode) (spanIn, spanOut int) {
	spanIn, spanOut = math.MaxInt, math.MaxInt
	for _, e := range connected(n) {
		span := e.target.layer - e.source.layer
		if e.target == n && span < spanIn {
			spanIn = span
		} else if span < spanOut {
			spanOut = span
		}
	}
	if spanIn == math.MaxInt {
		spanIn = -1
	}
	if spanOut == math.MaxInt {
		spanOut = -1
	}
	return spanIn, spanOut
}

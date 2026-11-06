// SPDX-License-Identifier: EPL-2.0 OR GPL-3.0-or-later

package elk

import (
	"cmp"
	"slices"
)

// rankPorts numbers the input or output ports of a layer top to bottom, input
// ports backwards within their node (calculatePortRanks,
// p3order/AbstractBarycenterPortDistributor.java line 123).
func (s *sweep) rankPorts(layer []*Node, input bool) {
	used := 0.0
	for _, n := range layer {
		var ports []*Port
		for _, p := range n.Ports {
			if input && len(p.In) > 0 || !input && len(p.Out) > 0 {
				ports = append(ports, p)
			}
		}
		if s.nodeRelative {
			s.rankWithinNode(ports, used, input)
			used++
		} else {
			s.rankInLayer(ports, used, input)
			used += float64(len(ports))
		}
	}
}

// rankInLayer gives each port a rank of its own (p3order/LayerTotalPortDistributor.java
// line 41).
func (s *sweep) rankInLayer(ports []*Port, used float64, input bool) {
	for i, p := range ports {
		s.ranks[p] = used + float64(i+1)
		if input {
			s.ranks[p] = used + float64(len(ports)-i)
		}
	}
}

// rankWithinNode shares one rank among a node's ports
// (p3order/NodeRelativePortDistributor.java line 46). There are no north ports
// while ordering.
func (s *sweep) rankWithinNode(ports []*Port, used float64, input bool) {
	step := 1 / float64(len(ports)+1)
	rank := used + step
	if input {
		rank, step = used+1-step, -step
	}
	for _, p := range ports {
		s.ranks[p] = rank
		rank += step
	}
}

// distribute orders the ports facing each other in the layers free and fixed
// (distributePortsWhileSweeping,
// p3order/AbstractBarycenterPortDistributor.java line 83). The fixed layer's
// ranks are still those orderLayer took. A subgraph laid out with its parent keeps
// its ports in the fixed layer (hasNestedGraph, line 100).
func (s *sweep) distribute(free, fixed int, forward bool) {
	side := sideWest
	if !forward {
		side = sideEast
	}
	for _, n := range s.layers[free] {
		s.placePorts(n, side)
	}
	s.rankPorts(s.layers[free], forward)
	for _, n := range s.layers[fixed] {
		if !n.Included {
			s.placePorts(n, opposed[side])
		}
	}
}

// placePorts sorts a node's ports by side, then by the ranks of what they
// connect to on side (distributePorts and sortPorts,
// p3order/AbstractBarycenterPortDistributor.java lines 152, 179 and 351). A port
// has one edge, and ranks start at 1, so ELK's special case for a barycenter of
// 0 sorts the same. Only a subgraph with external ports can hold other ports than
// these, without edges or with an edge within the layer. Its graph has ordered its
// ports by then: only the root may keep its initial order and skip that. This leaves
// ELK's barycenters for ports within a layer (line 238) nothing to sort. An external
// port dummy has one port, maybe with no edge.
func (s *sweep) placePorts(n *Node, side PortSide) {
	if n.FixedOrder || len(n.Ports) < 2 {
		return
	}
	for _, p := range n.Ports {
		if p.Side != side {
			continue
		}
		if side == sideEast {
			s.centers[p] = s.ranks[p.Out[0].Target]
		} else {
			s.centers[p] = -s.ranks[p.In[0].Source]
		}
	}
	slices.SortStableFunc(n.Ports, func(a, b *Port) int {
		return cmp.Or(cmp.Compare(a.Side, b.Side), cmp.Compare(s.centers[a], s.centers[b]))
	})
}

// farEnd gives the port at the other end of p's edge, if it has one: a port holds
// one edge at most.
func farEnd(p *Port) *Port {
	switch {
	case len(p.In) > 0:
		return p.In[0].Source
	case len(p.Out) > 0:
		return p.Out[0].Target
	}
	return nil
}

// crossings counts the crossings of an order (countAllCrossings,
// p3order/counting/AllCrossingsCounter.java line 61). The outer sides of the first
// and last layer hold only edges within those layers, inverted ports having moved
// all others, so ELK's count of edges leaving the side falls away
// (countInLayerCrossingsOnPorts, p3order/counting/CrossingsCounter.java line 387).
func crossings(layers [][]*Node) int {
	total := crossingsOn(portsInOrder(layers[0], sideWest, true))
	total += crossingsOn(portsInOrder(layers[len(layers)-1], sideEast, true))
	for i := 1; i < len(layers); i++ {
		left, right := portsInOrder(layers[i-1], sideEast, true), portsInOrder(layers[i], sideWest, false)
		total += crossingsOn(slices.Concat(left, right))
	}
	return total
}

// portsInOrder lists the ports on side of nodes, top down or bottom up
// (initPositions, CrossingsCounter.java line 460). West sides run bottom to top.
func portsInOrder(nodes []*Node, side PortSide, topDown bool) []*Port {
	var all []*Port
	add := func(n *Node) {
		ports := portsOn(n, side)
		if (side == sideWest) == topDown {
			slices.Reverse(ports)
		}
		all = append(all, ports...)
	}
	if topDown {
		for _, n := range nodes {
			add(n)
		}
	} else {
		for _, n := range slices.Backward(nodes) {
			add(n)
		}
	}
	return all
}

// crossingsOn counts pairs of edges crossing on a line of ports: going down it,
// the edges still open that end before the one that starts here
// (countCrossingsOnPorts, CrossingsCounter.java line 366).
func crossingsOn(ports []*Port) int {
	position := make(map[*Port]int, len(ports))
	for i, p := range ports {
		position[p] = i
	}
	tree := newIndexTree(len(ports))
	total := 0
	for i, p := range ports {
		tree.removeAll(i)
		if end := farEnd(p); end != nil && position[end] > i {
			total += tree.rank(position[end])
			tree.add(position[end])
		}
	}
	return total
}

// indexTree counts numbers below a given one
// (p3order/counting/BinaryIndexedTree.java).
type indexTree struct {
	sums, counts []int
}

func newIndexTree(size int) *indexTree {
	return &indexTree{make([]int, size+1), make([]int, size)}
}

func (t *indexTree) change(at, by int) {
	t.counts[at] += by
	for i := at + 1; i < len(t.sums); i += i & -i {
		t.sums[i] += by
	}
}

func (t *indexTree) add(at int) { t.change(at, 1) }

func (t *indexTree) removeAll(at int) { t.change(at, -t.counts[at]) }

func (t *indexTree) rank(at int) int {
	total := 0
	for i := at; i > 0; i -= i & -i {
		total += t.sums[i]
	}
	return total
}

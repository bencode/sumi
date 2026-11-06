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
// 0 sorts the same.
func (s *sweep) placePorts(n *Node, side PortSide) {
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

// crossings counts the crossings of an order (countAllCrossings,
// p3order/counting/AllCrossingsCounter.java line 61). Edges join neighboring
// layers only, so what ELK counts within the first and last layer is always 0.
func crossings(layers [][]*Node) int {
	total := 0
	for i := 1; i < len(layers); i++ {
		total += crossingsBetween(layers[i-1], layers[i])
	}
	return total
}

// crossingsBetween counts pairs of edges crossing between two layers, numbering
// the right layer's ports from the bottom up and adding up, in a binary indexed
// tree, the edges so far that end below (p3order/counting/CrossingsCounter.java
// lines 169 and 366).
func crossingsBetween(left, right []*Node) int {
	position := map[*Port]int{}
	for _, n := range slices.Backward(right) {
		for _, p := range n.Ports {
			if p.Side == sideWest {
				position[p] = len(position)
			}
		}
	}
	tree := make([]int, len(position)+1)
	total := 0
	for _, n := range left {
		for _, p := range n.Ports {
			if p.Side != sideEast {
				continue
			}
			end := position[p.Out[0].Target]
			for i := end; i > 0; i -= i & -i {
				total += tree[i]
			}
			for i := end + 1; i < len(tree); i += i & -i {
				tree[i]++
			}
		}
	}
	return total
}

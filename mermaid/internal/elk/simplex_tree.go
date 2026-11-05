// SPDX-License-Identifier: EPL-2.0 OR GPL-3.0-or-later

package elk

import "math"

// The spanning tree of network simplex. Line numbers are in NetworkSimplex.java.

func (e *simplexEdge) slack() int { return e.target.layer - e.source.layer - 1 }

func (s *simplex) clearVisited() {
	for _, e := range s.edges {
		e.visited = false
	}
}

// feasibleTree layers the nodes, then shifts the tree of tight edges until it
// spans them all (line 412).
func (s *simplex) feasibleTree() {
	s.topologicalNumbering()
	if len(s.edges) == 0 {
		return
	}
	root := s.nodes[0]
	for s.clearVisited(); s.tightTree(root) < len(s.nodes); s.clearVisited() {
		e := s.minimalSlack()
		shift := e.slack()
		if e.target.inTree {
			shift = -shift
		}
		for _, n := range s.nodes {
			if n.inTree {
				n.layer += shift
			}
		}
	}
	s.clearVisited()
	s.postorder(root)
	s.cutValues()
}

// topologicalNumbering puts each node just after the last of its sources
// (layeringTopologicalNumbering, line 453).
func (s *simplex) topologicalNumbering() {
	waiting := make(map[*simplexNode]int, len(s.nodes))
	var queue []*simplexNode
	for _, n := range s.nodes {
		waiting[n] = len(n.in)
		if len(n.in) == 0 {
			queue = append(queue, n)
		}
	}
	for len(queue) > 0 {
		n := queue[0]
		queue = queue[1:]
		for _, e := range n.out {
			e.target.layer = max(e.target.layer, n.layer+1)
			waiting[e.target]--
			if waiting[e.target] == 0 {
				queue = append(queue, e.target)
			}
		}
	}
}

// tightTree grows the tree through tight edges from n and gives its size
// (tightTreeDFS, line 520).
func (s *simplex) tightTree(n *simplexNode) int {
	count := 1
	n.inTree = true
	for _, e := range connected(n) {
		if e.visited {
			continue
		}
		e.visited = true
		opposite := e.other(n)
		if e.inTree {
			count += s.tightTree(opposite)
		} else if !opposite.inTree && e.slack() == 0 {
			e.inTree = true
			s.tree = append(s.tree, e)
			count += s.tightTree(opposite)
		}
	}
	return count
}

// minimalSlack gives the first of the tightest edges leaving the tree
// (line 554).
func (s *simplex) minimalSlack() *simplexEdge {
	least := math.MaxInt
	var found *simplexEdge
	for _, e := range s.edges {
		if e.source.inTree != e.target.inTree && e.slack() < least {
			least, found = e.slack(), e
		}
	}
	return found
}

// postorder numbers the tree from n and gives the lowest number below n
// (postorderTraversal, line 586).
func (s *simplex) postorder(n *simplexNode) int {
	lowest := math.MaxInt
	for _, e := range connected(n) {
		if e.inTree && !e.visited {
			e.visited = true
			lowest = min(lowest, s.postorder(e.other(n)))
		}
	}
	n.po = s.postOrder
	n.lowestPo = min(lowest, s.postOrder)
	s.postOrder++
	return n.lowestPo
}

// inHead tells whether n is on e's target side of the tree (isInHead,
// line 615).
func inHead(n *simplexNode, e *simplexEdge) bool {
	below := func(top *simplexNode) bool { return top.lowestPo <= n.po && n.po <= top.po }
	return (below(e.source) && below(e.target)) != (e.source.po < e.target.po)
}

// cutValues works out each tree edge's cut value, from the leaves inwards: the
// edges crossing from its tail side to its head side, less those crossing back
// (cutvalues, line 647).
func (s *simplex) cutValues() {
	var leaves []*simplexNode
	for _, n := range s.nodes {
		n.unknown = nil
		for _, e := range connected(n) {
			if e.inTree {
				n.unknown = append(n.unknown, e)
			}
		}
		if len(n.unknown) == 1 {
			leaves = append(leaves, n)
		}
	}
	for _, n := range leaves {
		for len(n.unknown) == 1 {
			e := n.unknown[0]
			e.cut = cutValue(e, n)
			e.source.unknown = without(e.source.unknown, e)
			e.target.unknown = without(e.target.unknown, e)
			n = e.other(n)
		}
	}
}

// cutValue gives e's cut value from those of the tree edges around n, its end
// on the side already known.
func cutValue(e *simplexEdge, n *simplexNode) int {
	cut := 1
	for _, other := range connected(n) {
		if other == e {
			continue
		}
		switch {
		case other.inTree && (e.source == other.source || e.target == other.target):
			cut -= other.cut - 1
		case other.inTree:
			cut += other.cut - 1
		case (n == e.source) == (other.source == n):
			cut++
		default:
			cut--
		}
	}
	return cut
}

// leaveEdge gives the first tree edge with a negative cut value (line 727).
func (s *simplex) leaveEdge() *simplexEdge {
	for _, e := range s.tree {
		if e.cut < 0 {
			return e
		}
	}
	return nil
}

// enterEdge gives the first of the tightest edges going from leave's head side
// to its tail side (line 748).
func (s *simplex) enterEdge(leave *simplexEdge) *simplexEdge {
	least := math.MaxInt
	var found *simplexEdge
	for _, e := range s.edges {
		if inHead(e.source, leave) && !inHead(e.target, leave) && e.slack() < least {
			least, found = e.slack(), e
		}
	}
	return found
}

// exchange swaps leave for enter, shifts the tail side so that enter is tight,
// and renumbers the tree (line 787).
func (s *simplex) exchange(leave, enter *simplexEdge) {
	leave.inTree = false
	s.tree = without(s.tree, leave)
	enter.inTree = true
	s.tree = append(s.tree, enter)
	shift := enter.slack()
	if !inHead(enter.target, leave) {
		shift = -shift
	}
	for _, n := range s.nodes {
		if !inHead(n, leave) {
			n.layer += shift
		}
	}
	s.postOrder = 1
	s.clearVisited()
	s.postorder(s.nodes[0])
	s.cutValues()
}

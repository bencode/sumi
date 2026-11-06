// SPDX-License-Identifier: EPL-2.0 OR GPL-3.0-or-later

package elk

import (
	"math"
	"slices"
)

// sweeps is the work on the graphs of one layout (graphInfoHolders, and
// graphsWhoseNodeOrderChanged as changed).
type sweeps struct {
	of      map[*Graph]*sweep
	changed map[*sweep]bool
}

// minimizeCrossings orders the layers of root and the graphs laid out with it
// (process and initialize, p3order/LayerSweepCrossingMinimizer.java lines 109 and
// 609). Each graph that goes bottom up takes its turn, sweeping through the
// subgraphs that do not, and only then do the orders found reach the graphs.
func minimizeCrossings(root *Graph) {
	if ordersNothing(root) {
		return
	}
	seed := root.Random.nextLong()
	graphs := graphsBottomUp(root)
	all := &sweeps{of: map[*Graph]*sweep{}, changed: map[*sweep]bool{}}
	for _, g := range graphs {
		all.of[g] = newSweep(root, g, all)
	}
	for _, g := range graphs {
		if all.of[g].bottomUp {
			root.Random.setSeed(seed)
			all.of[g].minimize()
		}
	}
	for _, g := range graphs {
		all.of[g].transfer()
	}
}

// ordersNothing tells whether the minimizer leaves g alone, drawing no random
// numbers: a graph with no layers, or a single node on its own (process,
// LayerSweepCrossingMinimizer.java line 109).
func ordersNothing(g *Graph) bool {
	single := len(g.Layers) == 1 && len(g.Layers[0].Nodes) == 1
	return len(g.Layers) == 0 || single && !g.Hierarchical
}

// minimize tries thoroughness times, each attempt going on from the order the
// last one left, and takes the order with the fewest crossings, then gives the
// port order to g's node (compareDifferentRandomizedLayouts and
// setPortOrderOnParentGraph, LayerSweepCrossingMinimizer.java lines 185 and 162).
// Only a root can stop short of a first sweep and so have no order, and a root has
// no node.
func (s *sweep) minimize() {
	clear(s.all.changed)
	fewest := math.MaxInt
	for i := range thoroughness {
		count := s.attempt(i == 0 && s.graph.Settings.modelOrder)
		if count < fewest {
			fewest = count
			s.all.saveBest()
		}
		if fewest == 0 {
			break
		}
	}
	if s.graph.External {
		parent := s.graph.Parent
		sortPortsByDummies(s.best.layers[len(s.best.layers)-1], parent, sideEast)
		sortPortsByDummies(reversed(s.best.layers[0]), parent, sideWest)
		parent.FixedOrder = true
	}
}

// keepOrders notes the current order of every graph moved since minimize began
// (setCurrentlyBestNodeOrders, LayerSweepCrossingMinimizer.java line 551).
func (all *sweeps) keepOrders() {
	for s := range all.changed {
		s.kept = s.snapshot()
	}
}

// saveBest makes those orders the best yet (saveAllNodeOrdersOfChangedGraphs,
// LayerSweepCrossingMinimizer.java line 545).
func (all *sweeps) saveBest() {
	for s := range all.changed {
		s.best = s.kept
	}
}

// transfer writes the best order to the graph, if an attempt got as far as having
// one (transferNodeAndPortOrdersToGraph, LayerSweepCrossingMinimizer.java line 632).
func (s *sweep) transfer() {
	if s.best == nil {
		return
	}
	for i, l := range s.graph.Layers {
		l.Nodes = s.best.layers[i]
		for _, n := range l.Nodes {
			n.Ports = s.best.ports[n]
		}
	}
}

// sweptThrough gives the sweep of n's graph, if the sweep over n goes through it.
func (s *sweep) sweptThrough(n *Node) *sweep {
	if !n.Included || s.all.of[n.Nested].bottomUp {
		return nil
	}
	return s.all.of[n.Nested]
}

// sweepInto sweeps through the subgraphs of layer (sweepInHierarchicalNodes,
// LayerSweepCrossingMinimizer.java line 471).
func (s *sweep) sweepInto(layer []*Node, forward, first bool) {
	for _, n := range layer {
		if nested := s.sweptThrough(n); nested != nil {
			nested.enter(n, forward, first)
		}
	}
}

// enter takes the order of parent's ports on the side the sweep comes from, sweeps
// across s, and leaves the order it reached on the ports of the far side
// (sweepInHierarchicalNode, LayerSweepCrossingMinimizer.java line 482). Without
// ports on the near side nothing orders the first layer, so it starts at random.
func (s *sweep) enter(parent *Node, forward, first bool) {
	start := 0
	if !forward {
		start = len(s.layers) - 1
	}
	if s.layers[start][0].Type == typeExternalPort {
		s.layers[start] = dummiesByPorts(s.layers[start], parent, nearSide(forward))
	} else {
		s.shuffle(forward)
	}
	s.pass(forward, first, false)
	if forward {
		sortPortsByDummies(s.layers[len(s.layers)-1], parent, sideEast)
	} else {
		sortPortsByDummies(reversed(s.layers[0]), parent, sideWest)
	}
}

// dummiesByPorts gives the dummy nodes of layer in the order of their ports on side
// of parent, west sides running bottom to top (sortPortDummiesByPortPositions,
// LayerSweepCrossingMinimizer.java line 525; inNorthSouthEastWestOrder,
// p3order/counting/CrossMinUtil.java line 44). Every port of a subgraph with external
// ports has its dummy, as addMissingDummies sees to, so ELK's check for ports without
// one does not apply.
func dummiesByPorts(layer []*Node, parent *Node, side PortSide) []*Node {
	byOrigin := make(map[*Port]*Node, len(layer))
	for _, n := range layer {
		byOrigin[n.Origin] = n
	}
	ports := portsOn(parent, side)
	if side == sideWest {
		slices.Reverse(ports)
	}
	sorted := make([]*Node, len(ports))
	for i, p := range ports {
		sorted[i] = byOrigin[p]
	}
	return sorted
}

// totalCrossings counts the crossings of s and of what it sweeps through
// (countCurrentNumberOfCrossings, LayerSweepCrossingMinimizer.java line 392).
func (s *sweep) totalCrossings() int {
	total := crossings(s.layers)
	for _, layer := range s.layers {
		for _, n := range layer {
			if nested := s.sweptThrough(n); nested != nil {
				total += nested.totalCrossings()
			}
		}
	}
	return total
}

// sortPortsByDummies orders the ports on side of parent as the dummy nodes of a
// layer came out, for the graph around it to work with
// (sortPortsByDummyPositionsInLastLayer, LayerSweepCrossingMinimizer.java line 504).
// The first and last layer may hold no such dummies.
func sortPortsByDummies(dummies []*Node, parent *Node, side PortSide) {
	if dummies[0].Type != typeExternalPort {
		return
	}
	next := 0
	for i, p := range parent.Ports {
		if p.Side == side {
			parent.Ports[i] = dummies[next].Origin
			next++
		}
	}
}

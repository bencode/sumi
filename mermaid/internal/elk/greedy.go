// SPDX-License-Identifier: EPL-2.0 OR GPL-3.0-or-later

package elk

import "slices"

// greedySwitch swaps neighbors in each layer for as long as that takes crossings
// away on both sides, sweeping back and forth until a sweep changes nothing
// (minimizeCrossingsNoCounter, p3order/LayerSweepCrossingMinimizer.java line 173,
// with intermediate/greedyswitch/GreedySwitchHeuristic.java). It runs after the
// barycenter sweeps, which leave every node's ports fixed in order
// (p3order/SweepCopy.java line 122), so GreedyPortDistributor skips them all.
func greedySwitch(g *Graph) {
	if ordersNothing(g) {
		return
	}
	g.Random.nextLong() // randomSeed, which only randomized layouts go back to
	forward := g.Random.nextBoolean()
	for improved := true; improved; forward = !forward {
		start, step := 0, 1
		if !forward {
			start, step = len(g.Layers)-1, -1
		}
		improved = switchInLayer(g.Layers, start)
		for i := start + step; i >= 0 && i < len(g.Layers); i += step {
			for switchInLayer(g.Layers, i) {
				improved = true
			}
		}
	}
}

// switchInLayer goes down the layer free once, swapping neighbors where that
// leaves fewer crossings, and tells whether it swapped any (sweepDownwardInLayer,
// GreedySwitchHeuristic.java line 100, and doesSwitchReduceCrossings,
// intermediate/greedyswitch/SwitchDecider.java line 124). There are no in-layer
// edges, north or south ports, or constraints for SwitchDecider to count.
func switchInLayer(layers []*Layer, free int) bool {
	position := neighborPositions(layers, free)
	nodes := layers[free].Nodes
	switched := false
	for i := 0; i < len(nodes)-1; i++ {
		upper, lower := nodes[i], nodes[i+1]
		if crossingsOf(upper, lower, position) > crossingsOf(lower, upper, position) {
			nodes[i], nodes[i+1] = lower, upper
			switched = true
		}
	}
	return switched
}

// neighborPositions numbers the ports facing the layer free, in each neighboring
// layer, top to bottom (setPortPositionsForNeighbouringLayers,
// intermediate/greedyswitch/BetweenLayerEdgeTwoNodeCrossingsCounter.java line 64).
func neighborPositions(layers []*Layer, free int) map[*Port]int {
	position := map[*Port]int{}
	number := func(l *Layer, side PortSide) {
		next := 0
		for _, n := range l.Nodes {
			ports := portsOn(n, side)
			if side == sideWest {
				slices.Reverse(ports)
			}
			for _, p := range ports {
				position[p] = next
				next++
			}
		}
	}
	if free > 0 {
		number(layers[free-1], sideEast)
	}
	if free < len(layers)-1 {
		number(layers[free+1], sideWest)
	}
	return position
}

// crossingsOf counts the crossings between the edges of upper and lower, with
// upper above: pairs that end the other way round on the same side. A port has
// one edge, so two ends never meet, which leaves ELK's merge of adjacency lists a
// count of inversions (countCrossingsByMergingAdjacencyLists,
// BetweenLayerEdgeTwoNodeCrossingsCounter.java line 210).
func crossingsOf(upper, lower *Node, position map[*Port]int) int {
	count := 0
	for _, p := range upper.Ports {
		for _, q := range lower.Ports {
			if p.Side == q.Side && position[neighborEnd(p)] > position[neighborEnd(q)] {
				count++
			}
		}
	}
	return count
}

// neighborEnd gives the port at the other end of p's edge.
func neighborEnd(p *Port) *Port {
	if p.Side == sideWest {
		return p.In[0].Source
	}
	return p.Out[0].Target
}

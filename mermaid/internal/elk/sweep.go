// SPDX-License-Identifier: EPL-2.0 OR GPL-3.0-or-later

package elk

import (
	"cmp"
	"math"
	"slices"
)

const (
	thoroughness = 7 // default of the option (Layered.melk line 1265)

	// perturbation is RANDOM_AMOUNT, a float (p3order/BarycenterHeuristic.java
	// line 204).
	perturbation = float64(float32(0.07))
)

// sweep is the crossing minimizer's work on one graph: the order it is working on
// and what it remembers about ports between layers. Directions come from the
// random numbers of the layout's root, whose seed goes back before each graph.
// All else comes from the graph's own, which never go back: the root's are one
// and the same (GraphInfoHolder.java line 94).
type sweep struct {
	random       *random
	noise        *random
	nodeRelative bool // NodeRelativePortDistributor, not LayerTotalPortDistributor
	layers       [][]*Node
	ranks        map[*Port]float64 // portRanks
	centers      map[*Port]float64 // portBarycenter
	kept         *snapshot         // currentlyBestNodeAndPortOrder
}

// snapshot is an order of nodes and ports (p3order/SweepCopy.java).
type snapshot struct {
	layers [][]*Node
	ports  map[*Node][]*Port
}

// centered is a node with its barycenter, NaN while it has none.
type centered struct {
	node   *Node
	center float64
}

// minimizeCrossings orders the layers of root and the graphs laid out with it
// (process and initialize, p3order/LayerSweepCrossingMinimizer.java lines 109 and
// 609). Without edges across their borders each graph goes on its own, bottom up.
func minimizeCrossings(root *Graph) {
	if ordersNothing(root) {
		return
	}
	seed := root.Random.nextLong()
	all := graphsBottomUp(root)
	sweeps := make([]*sweep, len(all))
	for i, g := range all {
		sweeps[i] = newSweep(root, g)
	}
	for i, g := range all {
		root.Random.setSeed(seed)
		sweeps[i].minimize(g)
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
// last one left, and takes the order with the fewest crossings
// (compareDifferentRandomizedLayouts, LayerSweepCrossingMinimizer.java line 185).
func (s *sweep) minimize(g *Graph) {
	var result *snapshot
	fewest := math.MaxInt
	for i := range thoroughness {
		count := s.attempt(i == 0 && g.Settings.modelOrder)
		if count < fewest {
			fewest, result = count, s.kept
		}
		if fewest == 0 {
			break
		}
	}
	if result == nil {
		return
	}
	for i, l := range g.Layers {
		l.Nodes = result.layers[i]
		for _, n := range l.Nodes {
			n.Ports = result.ports[n]
		}
	}
}

// newSweep starts on g, which draws a bit for its port distributor
// (p3order/ISweepPortDistributor.java line 58).
func newSweep(root, g *Graph) *sweep {
	s := &sweep{
		random: root.Random, noise: g.Random, nodeRelative: g.Random.nextBoolean(),
		ranks: map[*Port]float64{}, centers: map[*Port]float64{},
	}
	for _, l := range g.Layers {
		s.layers = append(s.layers, slices.Clone(l.Nodes))
	}
	return s
}

// attempt sweeps back and forth until crossings stop going down, and gives the
// count of the order it keeps (minimizeCrossingsWithCounter,
// p3order/LayerSweepCrossingMinimizer.java line 233). With initial, on the first
// attempt where model order counts, it starts from that order, forward, and
// leaves an order without crossings as it is.
//
// ELK has flags for a first and a second try, but gave both the same property
// id (options/InternalProperties.java lines 443 and 451), so clearing the second
// clears the first and there is no second try.
func (s *sweep) attempt(initial bool) int {
	forward := s.random.nextBoolean()
	if initial {
		if crossings(s.layers) == 0 {
			return 0
		}
		forward = true
	} else {
		s.shuffle(forward)
	}
	s.pass(forward, !initial)
	count := crossings(s.layers)
	for {
		s.kept = s.snapshot()
		if count == 0 {
			return 0
		}
		forward = !forward
		last := count
		s.pass(forward, false)
		count = crossings(s.layers)
		if last <= count {
			return last
		}
	}
}

// pass orders each layer by its neighbor already done (sweepReducingCrossings,
// p3order/LayerSweepCrossingMinimizer.java line 450). ELK distributes ports in
// the layer it starts from too, but on the side facing away, where there are
// none.
func (s *sweep) pass(forward, first bool) {
	start, step := 0, 1
	if !forward {
		start, step = len(s.layers)-1, -1
	}
	for i := start + step; i >= 0 && i < len(s.layers); i += step {
		s.orderLayer(i, i-step, forward, first)
		s.distribute(i, i-step, forward)
	}
}

// orderLayer sorts the layer free by barycenter (minimizeCrossings,
// p3order/BarycenterHeuristic.java lines 77 and 336). ELK's constraint resolver
// has no constraints to apply here.
func (s *sweep) orderLayer(free, fixed int, forward, first bool) {
	s.rankPorts(s.layers[fixed], !forward)
	all := s.barycenters(s.layers[free], forward)
	if first {
		s.fillRandomly(all)
	} else {
		fillBetween(all)
	}
	sortCentered(s.layers[free], all)
}

// shuffle gives the layer a sweep starts from a random order (setFirstLayerOrder
// and randomizeBarycenters, p3order/BarycenterHeuristic.java lines 358 and 112).
func (s *sweep) shuffle(forward bool) {
	layer := s.layers[0]
	if !forward {
		layer = s.layers[len(s.layers)-1]
	}
	all := make([]centered, len(layer))
	for i, n := range layer {
		all[i] = centered{n, s.noise.nextDouble()}
	}
	sortCentered(layer, all)
}

// barycenters averages the ranks of the ports each node connects to in the
// fixed layer, with some noise (calculateBarycenter,
// p3order/BarycenterHeuristic.java line 218). The conversion keeps Go from fusing
// the multiplication and subtraction, which JavaScript rounds apart.
func (s *sweep) barycenters(layer []*Node, forward bool) []centered {
	all := make([]centered, len(layer))
	for i, n := range layer {
		sum, degree := 0.0, 0
		for _, p := range n.Ports {
			for _, q := range fixedEnds(p, forward) {
				sum += s.ranks[q]
				degree++
			}
		}
		all[i] = centered{n, math.NaN()}
		if degree > 0 {
			sum += float64(s.noise.nextFloat()*perturbation) - perturbation/2
			all[i].center = sum / float64(degree)
		}
	}
	return all
}

// fixedEnds gives the ports p connects to on the side a sweep comes from.
func fixedEnds(p *Port, forward bool) []*Port {
	var ends []*Port
	if forward {
		for _, e := range p.In {
			ends = append(ends, e.Source)
		}
		return ends
	}
	for _, e := range p.Out {
		ends = append(ends, e.Target)
	}
	return ends
}

// fillRandomly places nodes without a barycenter anywhere
// (fillInUnknownBarycenters, p3order/BarycenterHeuristic.java line 129).
func (s *sweep) fillRandomly(all []centered) {
	top := 0.0
	for _, c := range all {
		if !math.IsNaN(c.center) {
			top = max(top, c.center)
		}
	}
	top += 2
	for i := range all {
		if math.IsNaN(all[i].center) {
			all[i].center = float64(s.noise.nextFloat()*top) - 1
		}
	}
}

// fillBetween keeps nodes without a barycenter between their neighbors in the
// layer (fillInUnknownBarycenters, p3order/BarycenterHeuristic.java line 129).
func fillBetween(all []centered) {
	last := -1.0
	for i := range all {
		if math.IsNaN(all[i].center) {
			next := last + 1
			j := slices.IndexFunc(all[i+1:], func(c centered) bool { return !math.IsNaN(c.center) })
			if j >= 0 {
				next = all[i+1+j].center
			}
			all[i].center = (last + next) / 2
		}
		last = all[i].center
	}
}

// sortCentered writes the nodes of all into layer by barycenter.
func sortCentered(layer []*Node, all []centered) {
	slices.SortStableFunc(all, func(a, b centered) int { return cmp.Compare(a.center, b.center) })
	for i, c := range all {
		layer[i] = c.node
	}
}

func (s *sweep) snapshot() *snapshot {
	kept := &snapshot{ports: map[*Node][]*Port{}}
	for _, layer := range s.layers {
		kept.layers = append(kept.layers, slices.Clone(layer))
		for _, n := range layer {
			kept.ports[n] = slices.Clone(n.Ports)
		}
	}
	return kept
}

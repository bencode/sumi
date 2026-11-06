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
// random numbers of the layout's root, whose seed goes back before each graph that
// goes bottom up. All else comes from the graph's own, which never go back: the
// root's are one and the same (GraphInfoHolder.java line 94).
type sweep struct {
	graph        *Graph
	all          *sweeps
	bottomUp     bool // dontSweepInto
	random       *random
	noise        *random
	nodeRelative bool // NodeRelativePortDistributor, not LayerTotalPortDistributor
	layers       [][]*Node
	layerOf      map[*Node]int
	ranks        map[*Port]float64 // portRanks
	centers      map[*Port]float64 // portBarycenter
	kept         *snapshot         // currentlyBestNodeAndPortOrder
	best         *snapshot         // bestNodeNPortOrder
}

// snapshot is an order of nodes and ports (p3order/SweepCopy.java).
type snapshot struct {
	layers [][]*Node
	ports  map[*Node][]*Port
}

// centered is a node with its barycenter, NaN while it has none, and what that
// came from (BarycenterState, p3order/BarycenterHeuristic.java line 287).
type centered struct {
	node    *Node
	center  float64
	sum     float64
	degree  int
	visited bool
}

// newSweep starts on g, which draws a bit for its port distributor
// (p3order/ISweepPortDistributor.java line 58).
func newSweep(root, g *Graph, all *sweeps) *sweep {
	s := &sweep{
		graph: g, all: all, random: root.Random, noise: g.Random, nodeRelative: g.Random.nextBoolean(),
		layerOf: map[*Node]int{}, ranks: map[*Port]float64{}, centers: map[*Port]float64{},
	}
	for i, l := range g.Layers {
		s.layers = append(s.layers, slices.Clone(l.Nodes))
		for _, n := range l.Nodes {
			s.layerOf[n] = i
		}
	}
	s.bottomUp = useBottomUp(g, s.layers)
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
		if s.totalCrossings() == 0 {
			return 0
		}
		forward = true
	} else {
		s.shuffle(forward)
	}
	s.pass(forward, true, initial)
	count := s.totalCrossings()
	for {
		s.all.keepOrders()
		if count == 0 {
			return 0
		}
		forward = !forward
		last := count
		s.pass(forward, false, false)
		count = s.totalCrossings()
		if last <= count {
			return last
		}
	}
}

// pass orders each layer by its neighbor already done, going through its
// subgraphs on the way (sweepReducingCrossings,
// p3order/LayerSweepCrossingMinimizer.java line 450). On a first sweep layers start
// from random numbers, but not those of a graph keeping its initial order: what
// that graph sweeps through has no such order to keep.
func (s *sweep) pass(forward, first, initial bool) {
	start, step := 0, 1
	if !forward {
		start, step = len(s.layers)-1, -1
	}
	s.placeSide(s.layers[start], nearSide(forward))
	s.sweepInto(s.layers[start], forward, first)
	for i := start + step; i >= 0 && i < len(s.layers); i += step {
		s.orderLayer(i, i-step, forward, first && !initial)
		s.distribute(i, i-step, forward)
		s.sweepInto(s.layers[i], forward, first)
	}
	s.all.changed[s] = true
}

// orderLayer sorts the layer free by barycenter (minimizeCrossings,
// p3order/BarycenterHeuristic.java lines 77 and 336). A layer of external ports
// keeps nodes without one in place, even on the first sweep. ELK's constraint
// resolver has no constraints to apply here.
func (s *sweep) orderLayer(free, fixed int, forward, first bool) {
	s.rankPorts(s.layers[fixed], !forward)
	all := s.barycenters(s.layers[free], forward)
	if first && s.layers[free][0].Type != typeExternalPort {
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
		all[i] = centered{node: n, center: s.noise.nextDouble()}
	}
	sortCentered(layer, all)
}

// barycenters gives each node of layer its barycenter (calculateBarycenters,
// p3order/BarycenterHeuristic.java line 191).
func (s *sweep) barycenters(layer []*Node, forward bool) []centered {
	all := make([]centered, len(layer))
	byNode := map[*Node]*centered{}
	for i, n := range layer {
		all[i] = centered{node: n, center: math.NaN()}
		byNode[n] = &all[i]
	}
	for _, n := range layer {
		s.barycenter(byNode[n], forward, byNode)
	}
	return all
}

// barycenter averages the ranks of the ports c's node connects to in the fixed
// layer, with some noise. A node of the same layer lends what it has, working
// that out first, and noise follows that order (calculateBarycenter,
// p3order/BarycenterHeuristic.java line 218). The conversion keeps Go from fusing
// the multiplication and subtraction, which JavaScript rounds apart.
func (s *sweep) barycenter(c *centered, forward bool, byNode map[*Node]*centered) {
	if c.visited {
		return
	}
	c.visited = true
	for _, p := range c.node.Ports {
		for _, q := range fixedEnds(p, forward) {
			if other := byNode[q.Owner]; other != nil {
				s.barycenter(other, forward, byNode)
				c.sum += other.sum
				c.degree += other.degree
			} else {
				c.sum += s.ranks[q]
				c.degree++
			}
		}
	}
	if c.degree > 0 {
		c.sum += float64(s.noise.nextFloat()*perturbation) - perturbation/2
		c.center = c.sum / float64(c.degree)
	}
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

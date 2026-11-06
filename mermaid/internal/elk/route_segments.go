// SPDX-License-Identifier: EPL-2.0 OR GPL-3.0-or-later

package elk

import (
	"math"
	"slices"
)

const (
	tolerance        = 1e-3 // ends closer than this are level
	criticalConflict = -1
	conflictFactor   = 0.5
	criticalFactor   = 0.2
	conflictPenalty  = 1
	crossingPenalty  = 16
)

// segment is the vertical part of the edges between two layers, the ports it
// joins and where they meet it (p5edges/orthogonal/HyperEdgeSegment.java). in
// holds coordinates coming from the left, out those leaving to the right, both
// sorted. A port holds one edge here, so a segment serves one edge, and each
// list one coordinate.
type segment struct {
	ports              []*Port
	mark, slot         int
	start, end         float64
	in, out            []float64
	outgoing, incoming []*dependency
	outWeight          int
	inWeight           int
	criticalOutWeight  int
	criticalInWeight   int
	partner, splitBy   *segment
}

// dependency puts source to the left of target
// (p5edges/orthogonal/HyperEdgeSegmentDependency.java).
type dependency struct {
	critical       bool
	source, target *segment
	weight         int
}

// router routes the edges between pairs of layers
// (p5edges/orthogonal/OrthogonalRoutingGenerator.java).
type router struct {
	random   *random
	slotGap  float64 // edgeSpacing
	critical float64 // criticalConflictThreshold
}

// route gives the edges between two layers their bends and tells how many
// slots they take, the first at start (routeEdges,
// OrthogonalRoutingGenerator.java line 159).
func (r *router) route(left, right []*Node, start float64) int {
	all := segmentsOf(left, right)
	r.critical = float64(criticalFactor * minimumDistance(all))
	critical := 0
	for i, first := range all {
		for _, second := range all[i+1:] {
			critical += r.depend(first, second)
		}
	}
	if critical >= 2 {
		all = r.split(findCycles(all, true, r.random), all)
	}
	breakDependencyCycles(all, r.random)
	number(all)
	slots := -1
	for _, s := range all {
		if s.level() {
			continue
		}
		slots = max(slots, s.slot)
		addBends(s, start, r.slotGap)
	}
	return slots + 1
}

// segmentsOf makes a segment for each edge leaving between the layers
// (createHyperEdgeSegments, OrthogonalRoutingGenerator.java line 298).
func segmentsOf(left, right []*Node) []*segment {
	var all []*segment
	known := map[*Port]*segment{}
	add := func(nodes []*Node, side PortSide) {
		for _, n := range nodes {
			for _, p := range n.Ports {
				if p.Side == side && len(p.Out) > 0 && known[p] == nil {
					s := &segment{}
					all = append(all, s)
					s.addPort(p, known)
				}
			}
		}
	}
	add(left, sideEast)
	add(right, sideWest)
	return all
}

// addPort adds p and all ports joined to it (addPortPositions,
// p5edges/orthogonal/HyperEdgeSegment.java line 123).
func (s *segment) addPort(p *Port, known map[*Port]*segment) {
	known[p] = s
	s.ports = append(s.ports, p)
	y := absolute(p).Y
	if p.Side == sideEast {
		s.in = insertSorted(s.in, y)
	} else {
		s.out = insertSorted(s.out, y)
	}
	s.measure()
	for _, e := range p.In {
		if known[e.Source] == nil {
			s.addPort(e.Source, known)
		}
	}
	for _, e := range p.Out {
		if known[e.Target] == nil {
			s.addPort(e.Target, known)
		}
	}
}

// insertSorted adds y unless all has it (HyperEdgeSegment.java line 146).
func insertSorted(all []float64, y float64) []float64 {
	at, found := slices.BinarySearch(all, y)
	if found {
		return all
	}
	return slices.Insert(all, at, y)
}

// measure finds where s starts and ends (recomputeExtent,
// HyperEdgeSegment.java line 341).
func (s *segment) measure() {
	all := slices.Concat(s.in, s.out)
	s.start, s.end = slices.Min(all), slices.Max(all)
}

// level tells whether s is too short to count.
func (s *segment) level() bool { return math.Abs(s.start-s.end) < tolerance }

// hyperedge is representsHyperedge (HyperEdgeSegment.java line 327).
func (s *segment) hyperedge() bool { return len(s.in)+len(s.out) > 2 }

// minimumDistance is the smallest gap between two different coordinates on the
// same side (minimumHorizontalSegmentDistance,
// OrthogonalRoutingGenerator.java line 248).
func minimumDistance(all []*segment) float64 {
	var in, out []float64
	for _, s := range all {
		in = append(in, s.in...)
		out = append(out, s.out...)
	}
	return min(minimumDifference(in), minimumDifference(out))
}

func minimumDifference(all []float64) float64 {
	slices.Sort(all)
	all = slices.Compact(all)
	least := math.MaxFloat64
	for i := 1; i < len(all); i++ {
		least = min(least, all[i]-all[i-1])
	}
	return least
}

// depend sets the order of a and b, the one costing less to the left, and tells
// how many critical dependencies that took (createDependencyIfNecessary,
// OrthogonalRoutingGenerator.java line 325).
func (r *router) depend(a, b *segment) int {
	if a.level() || b.level() {
		return 0
	}
	conflictsA, conflictsB := r.conflicts(a.out, b.in), r.conflicts(b.out, a.in)
	if conflictsA == criticalConflict || conflictsB == criticalConflict {
		critical := 0
		if conflictsA == criticalConflict {
			newDependency(true, b, a, 1)
			critical++
		}
		if conflictsB == criticalConflict {
			newDependency(true, a, b, 1)
			critical++
		}
		return critical
	}
	costA := conflictPenalty*conflictsA + crossingPenalty*orderCrossings(a, b)
	costB := conflictPenalty*conflictsB + crossingPenalty*orderCrossings(b, a)
	switch {
	case costA < costB:
		newDependency(false, a, b, costB-costA)
	case costA > costB:
		newDependency(false, b, a, costA-costB)
	case costA > 0:
		newDependency(false, a, b, 0)
		newDependency(false, b, a, 0)
	}
	return 0
}

// orderCrossings counts the crossings with left to the left of right
// (countCrossingsForSingleOrdering,
// p5edges/orthogonal/HyperEdgeSegmentSplitter.java line 345).
func orderCrossings(left, right *segment) int {
	return crossingsIn(left.out, right.start, right.end) + crossingsIn(right.in, left.start, left.end)
}

// conflicts counts the pairs of coordinates close enough to run on top of each
// other, or gives criticalConflict if a pair is nearly the same (countConflicts,
// OrthogonalRoutingGenerator.java line 397).
func (r *router) conflicts(a, b []float64) int {
	if len(a) == 0 || len(b) == 0 {
		return 0
	}
	count, i, j := 0, 0, 0
	for {
		p, q := a[i], b[j]
		if p > q-r.critical && p < q+r.critical {
			return criticalConflict
		}
		if threshold := conflictFactor * r.slotGap; p > q-threshold && p < q+threshold {
			count++
		}
		switch {
		case p <= q && i < len(a)-1:
			i++
		case q <= p && j < len(b)-1:
			j++
		default:
			return count
		}
	}
}

// crossingsIn counts the sorted coordinates from start to end (countCrossings,
// OrthogonalRoutingGenerator.java line 436).
func crossingsIn(all []float64, start, end float64) int {
	count := 0
	for _, y := range all {
		if y > end {
			break
		}
		if y >= start {
			count++
		}
	}
	return count
}

// newDependency puts source before target (createAndAddRegular and
// createAndAddCritical, HyperEdgeSegmentDependency.java lines 78 and 88).
func newDependency(critical bool, source, target *segment, weight int) *dependency {
	d := &dependency{critical: critical, weight: weight}
	d.setSource(source)
	d.setTarget(target)
	return d
}

func (d *dependency) setSource(s *segment) {
	if d.source != nil {
		d.source.outgoing = without(d.source.outgoing, d)
	}
	d.source = s
	if s != nil {
		s.outgoing = append(s.outgoing, d)
	}
}

func (d *dependency) setTarget(s *segment) {
	if d.target != nil {
		d.target.incoming = without(d.target.incoming, d)
	}
	d.target = s
	if s != nil {
		s.incoming = append(s.incoming, d)
	}
}

func (d *dependency) remove() {
	d.setSource(nil)
	d.setTarget(nil)
}

func (d *dependency) reverse() {
	source, target := d.source, d.target
	d.setSource(target)
	d.setTarget(source)
}

// number gives each segment a slot, one past those before it, then moves
// segments without edges from the left as far right as they go
// (topologicalNumbering, OrthogonalRoutingGenerator.java line 493).
func number(all []*segment) {
	var sources, rightward []*segment
	for _, s := range all {
		s.inWeight, s.outWeight = len(s.incoming), len(s.outgoing)
		if s.inWeight == 0 {
			sources = append(sources, s)
		}
		if s.outWeight == 0 && len(s.in) == 0 {
			rightward = append(rightward, s)
		}
	}
	highest := -1
	for len(sources) > 0 {
		s := sources[0]
		sources = sources[1:]
		for _, d := range s.outgoing {
			target := d.target
			target.slot = max(target.slot, s.slot+1)
			highest = max(highest, target.slot)
			if target.inWeight--; target.inWeight == 0 {
				sources = append(sources, target)
			}
		}
	}
	if highest == -1 {
		return
	}
	for _, s := range rightward {
		s.slot = highest
	}
	for len(rightward) > 0 {
		s := rightward[0]
		rightward = rightward[1:]
		for _, d := range s.incoming {
			source := d.source
			if len(source.in) > 0 {
				continue
			}
			source.slot = min(source.slot, s.slot-1)
			if source.outWeight--; source.outWeight == 0 {
				rightward = append(rightward, source)
			}
		}
	}
}

// addBends gives s's edges their bends, two for a whole segment and four for one
// split in two. The second half of a split holds no ports, so it adds none
// (calculateBendPoints,
// p5edges/orthogonal/direction/WestToEastRoutingStrategy.java line 45).
func addBends(s *segment, start, slotGap float64) {
	for _, p := range s.ports {
		from := absolute(p).Y
		for _, e := range p.Out {
			to := absolute(e.Target).Y
			if math.Abs(from-to) <= tolerance {
				continue
			}
			x := start + float64(s.slot)*slotGap
			e.Bends = append(e.Bends, Vec{x, from})
			if s.partner != nil {
				y := s.partner.in[0]
				e.Bends = append(e.Bends, Vec{x, y})
				x = start + float64(s.partner.slot)*slotGap
				e.Bends = append(e.Bends, Vec{x, y})
			}
			e.Bends = append(e.Bends, Vec{x, to})
		}
	}
}

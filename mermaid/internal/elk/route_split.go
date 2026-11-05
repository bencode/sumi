// SPDX-License-Identifier: EPL-2.0 OR GPL-3.0-or-later

package elk

import (
	"cmp"
	"slices"
)

// freeArea is a stretch between coordinates where a split can run.
type freeArea struct{ start, end float64 }

func (a freeArea) size() float64 { return a.end - a.start }

func (a freeArea) center() float64 { return (a.start + a.end) / 2 }

// areaRating is what a split in an area would cost.
type areaRating struct{ dependencies, crossings int }

// split cuts one segment of each critical cycle in two, the shortest first, and
// gives all segments including the new ones (splitSegments,
// p5edges/orthogonal/HyperEdgeSegmentSplitter.java line 53).
func (r *router) split(cycles []*dependency, all []*segment) []*segment {
	if len(cycles) == 0 {
		return all
	}
	areas := findFreeAreas(all, r.critical)
	chosen := segmentsToSplit(cycles)
	slices.SortStableFunc(chosen, func(a, b *segment) int {
		return cmp.Compare(a.end-a.start, b.end-b.start)
	})
	for _, s := range chosen {
		var at float64
		at, areas = splitPosition(s, areas, r.critical)
		all = append(all, s.splitAt(at))
		r.resolve(s, all)
	}
	return all
}

// findFreeAreas gives the gaps between coordinates wide enough to keep clear of
// both (findFreeAreas, HyperEdgeSegmentSplitter.java line 82).
func findFreeAreas(all []*segment, threshold float64) []freeArea {
	var coordinates []float64
	for _, s := range all {
		coordinates = append(append(coordinates, s.in...), s.out...)
	}
	slices.Sort(coordinates)
	var areas []freeArea
	for i := 1; i < len(coordinates); i++ {
		if coordinates[i]-coordinates[i-1] >= 2*threshold {
			areas = append(areas, freeArea{coordinates[i-1] + threshold, coordinates[i] - threshold})
		}
	}
	return areas
}

// segmentsToSplit chooses a segment of each dependency unless one of its two is
// chosen already, sparing hyperedges (decideWhichSegmentsToSplit,
// HyperEdgeSegmentSplitter.java line 116).
func segmentsToSplit(cycles []*dependency) []*segment {
	var chosen []*segment
	for _, d := range cycles {
		if slices.Contains(chosen, d.source) || slices.Contains(chosen, d.target) {
			continue
		}
		s, by := d.source, d.target
		if s.hyperedge() && !by.hyperedge() {
			s, by = by, s
		}
		chosen = append(chosen, s)
		s.splitBy = by
	}
	return chosen
}

// resolve puts s and its partner around the segment causing the split and
// orders them against all others (updateDependencies,
// HyperEdgeSegmentSplitter.java line 175).
func (r *router) resolve(s *segment, all []*segment) {
	newDependency(true, s, s.splitBy, 1)
	newDependency(true, s.splitBy, s.partner, 1)
	for _, other := range all {
		if other != s.splitBy && other != s && other != s.partner {
			r.depend(other, s)
			r.depend(other, s.partner)
		}
	}
}

// splitAt cuts s at y into itself and a partner taking over where it leaves to
// the right, and drops its dependencies (splitAt,
// p5edges/orthogonal/HyperEdgeSegment.java line 403).
func (s *segment) splitAt(y float64) *segment {
	s.partner = &segment{partner: s, in: []float64{y}, out: s.out}
	s.out = []float64{y}
	s.measure()
	s.partner.measure()
	for len(s.incoming) > 0 {
		s.incoming[0].remove()
	}
	for len(s.outgoing) > 0 {
		s.outgoing[0].remove()
	}
	return s.partner
}

// splitPosition chooses where s splits, the best free area it reaches or its
// middle, and gives the areas left (computePositionToSplitAndUpdateFreeAreas,
// HyperEdgeSegmentSplitter.java line 201).
func splitPosition(s *segment, areas []freeArea, threshold float64) (float64, []freeArea) {
	first, last := -1, -1
	for i, a := range areas {
		if a.start > s.end {
			break
		}
		if a.end >= s.start {
			if first < 0 {
				first = i
			}
			last = i
		}
	}
	if first < 0 {
		return (s.start + s.end) / 2, areas
	}
	best := bestArea(s, areas, first, last)
	return areas[best].center(), useArea(areas, best, threshold)
}

// bestArea rates the areas with a simulated split (chooseBestAreaIndex,
// HyperEdgeSegmentSplitter.java line 243).
func bestArea(s *segment, areas []freeArea, from, to int) int {
	best := from
	if from == to {
		return best
	}
	before, after := s.simulateSplit()
	rating := rateArea(s, before, after, areas[best])
	for i := from + 1; i <= to; i++ {
		current := rateArea(s, before, after, areas[i])
		if isBetter(areas[i], current, areas[best], rating) {
			best, rating = i, current
		}
	}
	return best
}

// simulateSplit gives the two halves of a split, each as long as its own side
// alone: ELK does not measure them again for each area it rates (simulateSplit,
// HyperEdgeSegment.java line 377).
func (s *segment) simulateSplit() (before, after *segment) {
	before = &segment{in: slices.Clone(s.in)}
	before.measure()
	after = &segment{out: slices.Clone(s.out)}
	after.measure()
	return before, after
}

// rateArea counts what splitting s in a would cost (rateArea,
// HyperEdgeSegmentSplitter.java line 277).
func rateArea(s, before, after *segment, a freeArea) areaRating {
	center := a.center()
	before.out = []float64{center}
	after.in = []float64{center}
	rating := areaRating{}
	for _, d := range s.incoming {
		rating.add(before, d.source)
		rating.add(after, d.source)
	}
	for _, d := range s.outgoing {
		rating.add(before, d.target)
		rating.add(after, d.target)
	}
	rating.dependencies += 2
	rating.crossings += orderCrossings(before, s.splitBy)
	rating.crossings += orderCrossings(s.splitBy, after)
	return rating
}

// add counts the better order of a and b, or both if neither is
// (updateConsideringBothOrderings, HyperEdgeSegmentSplitter.java line 322).
func (r *areaRating) add(a, b *segment) {
	ab, ba := orderCrossings(a, b), orderCrossings(b, a)
	if ab != ba {
		r.dependencies++
		r.crossings += min(ab, ba)
	} else if ab > 0 {
		r.dependencies += 2
		r.crossings += ab
	}
}

// isBetter prefers fewer crossings, then fewer dependencies, then more room
// (HyperEdgeSegmentSplitter.java line 355).
func isBetter(a freeArea, rating areaRating, best freeArea, bestRating areaRating) bool {
	if rating.crossings != bestRating.crossings {
		return rating.crossings < bestRating.crossings
	}
	if rating.dependencies != bestRating.dependencies {
		return rating.dependencies < bestRating.dependencies
	}
	return a.size() > best.size()
}

// useArea takes the middle of an area out of the free ones (useArea,
// HyperEdgeSegmentSplitter.java line 382).
func useArea(areas []freeArea, at int, threshold float64) []freeArea {
	old := areas[at]
	areas = slices.Delete(areas, at, at+1)
	if old.size()/2 < threshold {
		return areas
	}
	center := old.center()
	if end := center - threshold; old.start <= end {
		areas = slices.Insert(areas, at, freeArea{old.start, end})
		at++
	}
	if start := center + threshold; start <= old.end {
		areas = slices.Insert(areas, at, freeArea{start, old.end})
	}
	return areas
}

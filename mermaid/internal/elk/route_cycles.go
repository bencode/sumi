// SPDX-License-Identifier: EPL-2.0 OR GPL-3.0-or-later

package elk

import "math"

// findCycles gives the dependencies that go against a linear order of the
// segments, which Eades, Lin and Smyth's heuristic finds; with criticalOnly it
// looks at critical ones alone (detectCycles,
// p5edges/orthogonal/HyperEdgeCycleDetector.java line 62).
func findCycles(all []*segment, criticalOnly bool, r *random) []*dependency {
	sources, sinks := prepareMarks(all, criticalOnly)
	markInOrder(all, sources, sinks, criticalOnly, r)
	var cycles []*dependency
	for _, s := range all {
		for _, d := range s.outgoing {
			if (!criticalOnly || d.critical) && s.mark > d.target.mark {
				cycles = append(cycles, d)
			}
		}
	}
	return cycles
}

// prepareMarks gives each segment a negative mark and its weights, and gives the
// first sources and sinks (initialize, HyperEdgeCycleDetector.java line 96).
func prepareMarks(all []*segment, criticalOnly bool) (sources, sinks []*segment) {
	for i, s := range all {
		s.mark = -i - 1
		s.criticalInWeight = weightOf(s.incoming, true)
		s.criticalOutWeight = weightOf(s.outgoing, true)
		s.inWeight, s.outWeight = s.criticalInWeight, s.criticalOutWeight
		if !criticalOnly {
			s.inWeight, s.outWeight = weightOf(s.incoming, false), weightOf(s.outgoing, false)
		}
		if s.outWeight == 0 {
			sinks = append(sinks, s)
		} else if s.inWeight == 0 {
			sources = append(sources, s)
		}
	}
	return sources, sinks
}

func weightOf(all []*dependency, criticalOnly bool) int {
	sum := 0
	for _, d := range all {
		if !criticalOnly || d.critical {
			sum += d.weight
		}
	}
	return sum
}

// marking is the state of markInOrder.
type marking struct {
	sources, sinks []*segment
	criticalOnly   bool
}

// markInOrder takes out sinks and sources, and when none are left the segment
// with the most outflow, ties going by chance. ELK keeps the segments left in a
// set sorted by mark, so the negative marks make it run from the last segment to
// the first (computeLinearOrderingMarks, HyperEdgeCycleDetector.java line 148).
func markInOrder(all, sources, sinks []*segment, criticalOnly bool, r *random) {
	unprocessed := make(map[int]*segment, len(all))
	for _, s := range all {
		unprocessed[s.mark] = s
	}
	m := &marking{sources, sinks, criticalOnly}
	base := len(all)
	nextSink, nextSource := base-1, base+1
	take := func(s *segment, mark int) {
		delete(unprocessed, s.mark)
		s.mark = mark
		m.update(s)
	}
	for len(unprocessed) > 0 {
		for len(m.sinks) > 0 {
			sink := m.sinks[0]
			m.sinks = m.sinks[1:]
			take(sink, nextSink)
			nextSink--
		}
		for len(m.sources) > 0 {
			source := m.sources[0]
			m.sources = m.sources[1:]
			take(source, nextSource)
			nextSource++
		}
		if most := mostOutflow(unprocessed, len(all), criticalOnly); len(most) > 0 {
			take(most[r.nextInt(int32(len(most)))], nextSource)
			nextSource++
		}
	}
	for _, s := range all {
		if s.mark < base {
			s.mark += len(all) + 1
		}
	}
}

// mostOutflow gives the segments with the highest outflow, or the first with
// critical dependencies leaving it and none coming in
// (HyperEdgeCycleDetector.java lines 181 to 201).
func mostOutflow(unprocessed map[int]*segment, count int, criticalOnly bool) []*segment {
	var most []*segment
	highest := math.MinInt
	for mark := -count; mark < 0; mark++ {
		s := unprocessed[mark]
		if s == nil {
			continue
		}
		if !criticalOnly && s.criticalOutWeight > 0 && s.criticalInWeight <= 0 {
			return []*segment{s}
		}
		outflow := s.outWeight - s.inWeight
		if outflow > highest {
			most, highest = nil, outflow
		}
		if outflow == highest {
			most = append(most, s)
		}
	}
	return most
}

// update takes s out of its neighbors' weights (updateNeighbors,
// HyperEdgeCycleDetector.java line 235).
func (m *marking) update(s *segment) {
	for _, d := range s.outgoing {
		if m.criticalOnly && !d.critical || d.target.mark >= 0 || d.weight <= 0 {
			continue
		}
		t := d.target
		t.inWeight -= d.weight
		if d.critical {
			t.criticalInWeight -= d.weight
		}
		if t.inWeight <= 0 && t.outWeight > 0 {
			m.sources = append(m.sources, t)
		}
	}
	for _, d := range s.incoming {
		if m.criticalOnly && !d.critical || d.source.mark >= 0 || d.weight <= 0 {
			continue
		}
		t := d.source
		t.outWeight -= d.weight
		if d.critical {
			t.criticalOutWeight -= d.weight
		}
		if t.outWeight <= 0 && t.inWeight > 0 {
			m.sinks = append(m.sinks, t)
		}
	}
}

// breakDependencyCycles removes the dependencies closing a cycle that have no
// weight and reverses the others (breakNonCriticalCycles,
// p5edges/orthogonal/OrthogonalRoutingGenerator.java line 471).
func breakDependencyCycles(all []*segment, r *random) {
	for _, d := range findCycles(all, false, r) {
		if d.weight == 0 {
			d.remove()
		} else {
			d.reverse()
		}
	}
}

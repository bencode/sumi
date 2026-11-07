package graphlayout

import (
	"math"
	"slices"

	"github.com/bencode/sumi/mermaid/internal/elk"
)

const (
	jogMax = 16   // TERMINAL_JOG_MAX: the largest step beside a port worth removing
	runMax = 30   // TERMINAL_RUN_MAX: how far from the node that step may be
	jogEps = 0.01 // JOG_EPS: far smaller than EPS, as ELK leaves steps under a pixel
)

type axis int

const (
	diagonal axis = iota
	horizontal
	vertical
)

// axisOf gives the axis of a segment (axisOf, render.ts).
func axisOf(a, b elk.Vec) axis {
	dx, dy := math.Abs(b.X-a.X), math.Abs(b.Y-a.Y)
	switch {
	case dx > jogEps && dy <= jogEps:
		return horizontal
	case dy > jogEps && dx <= jogEps:
		return vertical
	}
	return diagonal
}

// movedRun is a stretch of a path before and after it moved (StraightenedRun, render.ts).
type movedRun struct{ oldA, oldB, newA, newB elk.Vec }

// straightenFront removes the step at the front of pts, by moving the channel after
// it onto the port's row. The port stays, as other edges of the node sit at their
// spread positions. The whole run moves, or a diagonal would join its two halves,
// but never the far end, whose port would move (straightenFront, render.ts).
func straightenFront(pts []elk.Vec) ([]elk.Vec, movedRun, bool) {
	if len(pts) < 5 {
		return pts, movedRun{}, false
	}
	p0, p1, p2, p3 := pts[0], pts[1], pts[2], pts[3]
	along, across := axisOf(p0, p1), horizontal
	if along == horizontal {
		across = vertical
	}
	if along == diagonal || axisOf(p2, p3) != along || axisOf(p1, p2) != across {
		return pts, movedRun{}, false
	}
	if math.Hypot(p1.X-p0.X, p1.Y-p0.Y) > runMax {
		return pts, movedRun{}, false
	}
	jog, forward := math.Abs(p2.X-p1.X), sign(p1.Y-p0.Y) == sign(p3.Y-p2.Y)
	if along == horizontal {
		jog, forward = math.Abs(p2.Y-p1.Y), sign(p1.X-p0.X) == sign(p3.X-p2.X)
	}
	if jog > jogMax || !forward {
		return pts, movedRun{}, false
	}
	last := 3
	for last+1 < len(pts) && axisOf(pts[last], pts[last+1]) == along {
		last++
	}
	if last == len(pts)-1 {
		return pts, movedRun{}, false
	}
	moved := slices.Clone(pts)
	for i := 2; i <= last; i++ {
		if along == horizontal {
			moved[i].Y = p0.Y
		} else {
			moved[i].X = p0.X
		}
	}
	run := movedRun{pts[2], pts[last], moved[2], moved[last]}
	return slices.Delete(moved, 1, 3), run, true
}

// straightenJogs does both ends of pts, and gives the runs it moved
// (straightenTerminalJogsWithRuns, render.ts).
func straightenJogs(pts []elk.Vec) ([]elk.Vec, []movedRun) {
	var runs []movedRun
	front, run, ok := straightenFront(pts)
	if ok {
		runs = append(runs, run)
	}
	back := slices.Clone(front)
	slices.Reverse(back)
	if turned, run, ok := straightenFront(back); ok {
		slices.Reverse(turned)
		return turned, append(runs, run)
	}
	return front, runs
}

// straightenEdges removes the step beside a port of each edge, unless that adds a
// crossing or puts text and an edge on each other, which needs the other edges as
// they are by then. A label moves with the run it sat on (straightenEdgeTerminals,
// render.ts).
func straightenEdges(routes []route) []route {
	straight := slices.Clone(routes)
	for i, r := range straight {
		if len(r.points) < 5 {
			continue
		}
		candidate, runs := straightenJogs(r.points)
		if len(runs) == 0 {
			continue
		}
		label := r.label
		if label != nil {
			label = &edgeLabel{projectLabel(label.center, runs), label.size}
		}
		var before, after, coveredBefore, coveredAfter int
		for j, other := range straight {
			if j == i || len(other.points) < 2 {
				continue
			}
			before += crossings(r.points, other.points)
			after += crossings(candidate, other.points)
			coveredBefore += count(covers(other.points, r.label)) + count(covers(r.points, other.label))
			coveredAfter += count(covers(other.points, label)) + count(covers(candidate, other.label))
		}
		if after <= before && coveredAfter <= coveredBefore {
			straight[i].points, straight[i].label = candidate, label
		}
	}
	return straight
}

func count(b bool) int {
	if b {
		return 1
	}
	return 0
}

// crossings counts where two paths cross, inside segments of both (crossingCount,
// render.ts).
func crossings(a, b []elk.Vec) int {
	n := 0
	for i := 0; i+1 < len(a); i++ {
		for j := 0; j+1 < len(b); j++ {
			if segmentsCross(a[i], a[i+1], b[j], b[j+1]) {
				n++
			}
		}
	}
	return n
}

func segmentsCross(a1, a2, b1, b2 elk.Vec) bool {
	side := func(o, p, q elk.Vec) float64 {
		return float64((p.X-o.X)*(q.Y-o.Y)) - float64((p.Y-o.Y)*(q.X-o.X))
	}
	d1, d2 := side(b1, b2, a1), side(b1, b2, a2)
	d3, d4 := side(a1, a2, b1), side(a1, a2, b2)
	return (d1 > 0 && d2 < 0 || d1 < 0 && d2 > 0) && (d3 > 0 && d4 < 0 || d3 < 0 && d4 > 0)
}

// projectLabel gives where a label goes when runs have moved: the same way along
// the run nearest it, judged where the runs were, since a run moves only across
// itself (projectLabelOntoStraightenedRun, render.ts).
func projectLabel(label elk.Vec, runs []movedRun) elk.Vec {
	best, projected := math.Inf(1), label
	for _, r := range runs {
		t := paramOnSegment(label, r.oldA, r.oldB)
		old := pointAt(r.oldA, r.oldB, t)
		dx, dy := old.X-label.X, old.Y-label.Y
		if d := float64(dx*dx) + float64(dy*dy); d < best {
			best, projected = d, pointAt(r.newA, r.newB, t)
		}
	}
	return projected
}

// paramOnSegment gives where p falls along a-b, from 0 to 1.
func paramOnSegment(p, a, b elk.Vec) float64 {
	abx, aby := b.X-a.X, b.Y-a.Y
	lengthSquared := float64(abx*abx) + float64(aby*aby)
	if lengthSquared == 0 {
		return 0
	}
	return max(0, min(1, (float64((p.X-a.X)*abx)+float64((p.Y-a.Y)*aby))/lengthSquared))
}

func pointAt(a, b elk.Vec, t float64) elk.Vec {
	return elk.Vec{X: a.X + float64(t*(b.X-a.X)), Y: a.Y + float64(t*(b.Y-a.Y))}
}

// covers tells whether a path passes through a label.
func covers(path []elk.Vec, l *edgeLabel) bool {
	if l == nil {
		return false
	}
	for i := 0; i+1 < len(path); i++ {
		if segmentHitsBox(path[i], path[i+1], l) {
			return true
		}
	}
	return false
}

// segmentHitsBox clips a-b to a label's box, after Liang and Barsky (segmentHitsBox,
// render.ts).
func segmentHitsBox(a, b elk.Vec, l *edgeLabel) bool {
	x1, x2 := l.center.X-l.size.Width/2, l.center.X+l.size.Width/2
	y1, y2 := l.center.Y-l.size.Height/2, l.center.Y+l.size.Height/2
	dx, dy := b.X-a.X, b.Y-a.Y
	t0, t1 := 0.0, 1.0
	for _, c := range [][2]float64{{-dx, a.X - x1}, {dx, x2 - a.X}, {-dy, a.Y - y1}, {dy, y2 - a.Y}} {
		p, q := c[0], c[1]
		switch {
		case p == 0:
			if q < 0 {
				return false
			}
		case p < 0:
			t0 = max(t0, q/p)
		default:
			t1 = min(t1, q/p)
		}
	}
	return t0 < t1
}

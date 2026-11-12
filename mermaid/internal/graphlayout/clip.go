package graphlayout

import (
	"math"
	"slices"

	"github.com/bencode/sumi/mermaid/internal/elk"
)

// replaceTolerance is how near a new end has to be to the old one to count as the
// same (replaceEndpoint, geometry.ts).
const replaceTolerance = 0.1

// clipEdge ends path, which runs from the center of start to that of end, on the
// outlines of both. An end on a subgraph's frame stays, as ELK put it there. A path
// that comes out broken falls back on the one it was cut from
// (sanitizeElkEdgePoints, render.ts).
func clipEdge(path []elk.Vec, start, end frame) []elk.Vec {
	startBounds, endBounds := start.bounds(), end.bounds()
	skipStart := start.group && onBorder(startBounds, path[1], borderTolerance)
	skipEnd := end.group && onBorder(endBounds, path[len(path)-2], borderTolerance)
	if skipStart {
		path = path[1:]
	}
	if skipEnd {
		path = path[:len(path)-1]
	}
	if start.group && !skipStart {
		path, skipStart = clipGroupEnd(path, startBounds, true)
	}
	if end.group && !skipEnd {
		path, skipEnd = clipGroupEnd(path, endBounds, false)
	}
	var clipped []elk.Vec
	if skipStart || skipEnd {
		if !skipStart {
			path = intersectStart(path, start, startBounds)
		}
		if !skipEnd {
			path = intersectEnd(path, end, endBounds)
		}
		clipped = path
	} else {
		clipped = cutter(path, start, end)
	}
	if len(clipped) < 2 || slices.ContainsFunc(clipped, invalid) {
		clipped = path
		if finite := slices.DeleteFunc(slices.Clone(path), invalid); len(finite) >= 2 {
			clipped = finite
		}
	}
	return dedupe(clipped)
}

func invalid(p elk.Vec) bool {
	return math.IsNaN(p.X) || math.IsInf(p.X, 0) || math.IsNaN(p.Y) || math.IsInf(p.Y, 0)
}

// clipGroupEnd drops the points inside a frame that has moved and ends the path
// where it crosses the frame, and tells whether it did (clipGroupEndpoint, render.ts).
func clipGroupEnd(points []elk.Vec, b Box, atStart bool) ([]elk.Vec, bool) {
	step, i := 1, 0
	if !atStart {
		step, i = -1, len(points)-1
	}
	terminal := i
	for i >= 0 && i < len(points) && !outsideNode(b, points[i]) {
		i += step
	}
	if i == terminal || i < 0 || i >= len(points) {
		return points, false
	}
	outside, inside := points[i], points[i-step]
	dx, dy := outside.X-inside.X, outside.Y-inside.Y
	t := math.Inf(1)
	if dx != 0 {
		t = (b.X + sign(dx)*b.Width/2 - inside.X) / dx
	}
	if dy != 0 {
		t = min(t, (b.Y+sign(dy)*b.Height/2-inside.Y)/dy)
	}
	crossing := elk.Vec{X: inside.X + t*dx, Y: inside.Y + t*dy}
	if atStart {
		return slices.Concat([]elk.Vec{crossing}, points[i:]), true
	}
	return slices.Concat(points[:i+1], []elk.Vec{crossing}), true
}

// intersectStart replaces the start of points by where the first point outside
// start's bounds leads to (applyStartIntersectionIfNeeded, render.ts).
func intersectStart(points []elk.Vec, f frame, b Box) []elk.Vec {
	i := slices.IndexFunc(points, func(p elk.Vec) bool { return outsideNode(b, p) })
	if i == -1 {
		return points
	}
	return replaceStart(points, nodeIntersection(f, b, points[i], points[0]))
}

// intersectEnd is intersectStart at the other end (applyEndIntersectionIfNeeded,
// render.ts).
func intersectEnd(points []elk.Vec, f frame, b Box) []elk.Vec {
	i := lastOutside(points, b)
	if i == -1 {
		return points
	}
	return replaceEnd(points, nodeIntersection(f, b, points[i], points[len(points)-1]))
}

func lastOutside(points []elk.Vec, b Box) int {
	for i := len(points) - 1; i >= 0; i-- {
		if outsideNode(b, points[i]) {
			return i
		}
	}
	return -1
}

func sameEnd(a, b elk.Vec) bool {
	return math.Abs(a.X-b.X) < replaceTolerance && math.Abs(a.Y-b.Y) < replaceTolerance
}

// replaceStart puts p at the start, or drops the start if p is on it (replaceEndpoint,
// geometry.ts).
func replaceStart(points []elk.Vec, p elk.Vec) []elk.Vec {
	if sameEnd(points[0], p) {
		return points[1:]
	}
	return slices.Concat([]elk.Vec{p}, points[1:])
}

func replaceEnd(points []elk.Vec, p elk.Vec) []elk.Vec {
	last := len(points) - 1
	if sameEnd(points[last], p) {
		return points[:last]
	}
	return slices.Concat(points[:last], []elk.Vec{p})
}

// attachPoint gives the end of an edge at the node f, whose port is points[port] and
// which leaves it toward points[port+step]: on the axis it leaves along, or else on
// the ray from the center. A subgraph's frame is its own outline
// (attachAlongDepartureAxis, render.ts).
func attachPoint(f frame, b Box, points []elk.Vec, port, step int, center elk.Vec) elk.Vec {
	if next := port + step; !f.group && next >= 0 && next < len(points) {
		if p, ok := outlineAttachPoint(f, b, points[port], points[next]); ok {
			return p
		}
	}
	return nodeIntersection(f, b, points[port], center)
}

// cutter ends points on the outlines of start and end, and drops a last point too
// near the one before it (cutter2, render.ts).
func cutter(points []elk.Vec, start, end frame) []elk.Vec {
	startBounds, endBounds := start.bounds(), end.bounds()
	startCenter, endCenter := points[0], points[len(points)-1]

	if i := slices.IndexFunc(points, func(p elk.Vec) bool { return outsideNode(startBounds, p) }); i != -1 {
		points = replaceStart(points, attachPoint(start, startBounds, points, i, 1, startCenter))
	}

	i := lastOutside(points, endBounds)
	if i == -1 && len(points) > 1 {
		i = len(points) - 2
	}
	if i != -1 {
		points = replaceEnd(points, attachPoint(end, endBounds, points, i, -1, endCenter))
	}

	if n := len(points); n > 1 {
		dx, dy := points[n-1].X-points[n-2].X, points[n-1].Y-points[n-2].Y
		if math.Hypot(dx, dy) < 2 {
			points = points[:n-1]
		}
	}
	return points
}

// dedupe drops each point on top of the one before it (dedupeConsecutivePoints,
// render.ts).
func dedupe(points []elk.Vec) []elk.Vec {
	var kept []elk.Vec
	for i, p := range points {
		if i == 0 || math.Abs(p.X-points[i-1].X) > 1e-6 || math.Abs(p.Y-points[i-1].Y) > 1e-6 {
			kept = append(kept, p)
		}
	}
	return kept
}

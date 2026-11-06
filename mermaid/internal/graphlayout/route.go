package graphlayout

import (
	"math"
	"slices"

	"github.com/bencode/sumi/mermaid/internal/elk"
)

// minMarkerSegment is the shortest last segment that needs no check against a
// marker's own room (MIN_END_MARKER_SEGMENT_LENGTH, render.ts).
const minMarkerSegment = 8

// markerOffsets says how far the drawing of a path pulls its end back for a marker:
// the larger of Mermaid's two tables (markerOffsets, markerOffsets2,
// utils/lineWithOffset.ts). A marker not here has none.
var markerOffsets = map[string]float64{
	"aggregation": 17.25, "extension": 17.25, "composition": 17.25, "dependency": 6,
	"lollipop": 13.5, "arrow_point": 4, "arrow_barb_neo": 5.5, "arrow_cross": 12.5,
	"arrow_circle": 12.5, "arrow_open_head": 4, "arrow_hollow_circle": 8,
}

// edgeLabel is an edge's text, placed by its center.
type edgeLabel struct {
	center elk.Vec
	size   elk.Size
}

// route is an edge on its way to being placed, in its own direction.
type route struct {
	id, start, end string
	points         []elk.Vec
	label          *edgeLabel
}

// routeEdges gives edges their paths (applyElkEdgeLayout, render.ts).
func routeEdges(edges []Edge, laid []elkEdge, reversed map[string]bool, frames map[string]frame, t tree) []route {
	byID := map[string]elkEdge{}
	for _, e := range laid {
		byID[e.id] = e
	}
	routes := make([]route, len(edges))
	for i, e := range edges {
		routes[i] = routeEdge(e, byID[e.ID], reversed[e.ID], frames, t)
	}
	return routes
}

// routeEdge runs from the center of a node to the center of the other through what
// ELK gave, then clips that to their outlines.
func routeEdge(e Edge, laid elkEdge, reversed bool, frames map[string]frame, t tree) route {
	start, end := frames[e.Start], frames[e.End]
	offset := edgeOffset(e.Start, e.End, frames, t)
	inner := make([]elk.Vec, len(laid.points))
	for i, p := range laid.points {
		inner[i] = elk.Vec{X: p.X + offset.X, Y: p.Y + offset.Y}
	}
	if reversed {
		slices.Reverse(inner)
	}
	clipped := clipEdge(slices.Concat([]elk.Vec{start.center()}, inner, []elk.Vec{end.center()}), start, end)
	clipped = trimEnd(clipped, end.bounds(), markerOffsets[e.EndMarker])
	clipped = trimStart(clipped, start.bounds(), markerOffsets[e.StartMarker])
	r := route{id: e.ID, start: e.Start, end: e.End, points: clipped}
	if e.Label != nil {
		l := laid.labels[0]
		r.label = &edgeLabel{elk.Vec{X: l.X + offset.X + l.Width/2, Y: l.Y + offset.Y + l.Height/2}, *e.Label}
	}
	return r
}

// tooShort tells whether a marker has no room on the segment from end to next, a
// stub on the node's border: pulling the end back would run past next and flip the
// marker.
func tooShort(end, next elk.Vec, b Box, offset float64) bool {
	return math.Hypot(next.X-end.X, next.Y-end.Y) < max(minMarkerSegment, offset*2) && onBorder(b, next, 1)
}

// trimEnd drops a stub before the end (ensureEndMarkerSegmentLength, render.ts).
func trimEnd(points []elk.Vec, b Box, offset float64) []elk.Vec {
	n := len(points)
	if offset <= 0 || n < 3 || !tooShort(points[n-1], points[n-2], b, offset) {
		return points
	}
	return slices.Concat(points[:n-2], points[n-1:])
}

// trimStart drops a stub after the start (ensureStartMarkerSegmentLength, render.ts).
func trimStart(points []elk.Vec, b Box, offset float64) []elk.Vec {
	if offset <= 0 || len(points) < 3 || !tooShort(points[0], points[1], b, offset) {
		return points
	}
	return slices.Concat(points[:1], points[2:])
}

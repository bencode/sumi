package flowchart

import (
	"math"
	"slices"

	"github.com/bencode/sumi/mermaid/internal/elk"
	"github.com/bencode/sumi/scene"
)

// Edges in the neo look, from edges.js insertEdge and the -margin markers of markers.js.

const cornerRadius = 5 // generateRoundedPath's radius

// pullBack is markerOffsets: how far a marker moves its end of the line in.
var pullBack = map[string]float64{"arrow_point": 4}

// gap is markerOffsets2: how much of the line the neo look leaves undrawn at an end,
// under its marker, with stroke-dasharray.
var gap = map[string]float64{"arrow_point": 4, "arrow_circle": 12.5, "arrow_cross": 12.5}

// widths are the edge-thickness classes' stroke widths; invisible draws none.
var widths = map[string]float64{"thick": 3.5, "invisible": 0}

// line is an edge's path as Mermaid writes its d: the layout's points, their ends
// moved in for the markers, the corners rounded.
func line(pts []elk.Vec, start, end string) []scene.Seg {
	ps := make([]scene.Point, len(pts))
	for i, p := range pts {
		ps[i] = scene.Point{X: p.X, Y: p.Y}
	}
	if n := len(ps); n >= 2 {
		ps[0] = moveIn(ps[0], ps[1], pullBack[start])
		ps[n-1] = moveIn(ps[n-1], ps[n-2], pullBack[end])
	}
	return rounded(ps, cornerRadius)
}

// moveIn is applyMarkerOffsetsToPoints for one end: p moved by d toward q.
func moveIn(p, q scene.Point, d float64) scene.Point {
	if d == 0 {
		return p
	}
	a := math.Atan2(q.Y-p.Y, q.X-p.X)
	return scene.Point{X: p.X + d*math.Cos(a), Y: p.Y + d*math.Sin(a)}
}

// rounded is generateRoundedPath: a line through pts with a quadratic curve about
// each corner, cut no more than half way back along either side.
func rounded(pts []scene.Point, radius float64) []scene.Seg {
	if len(pts) < 2 {
		return nil
	}
	const epsilon = 1e-5
	p := (&scene.Path{}).MoveTo(pts[0].X, pts[0].Y)
	for i := 1; i < len(pts)-1; i++ {
		prev, cur, next := pts[i-1], pts[i], pts[i+1]
		dx1, dy1 := cur.X-prev.X, cur.Y-prev.Y
		dx2, dy2 := next.X-cur.X, next.Y-cur.Y
		len1, len2 := math.Hypot(dx1, dy1), math.Hypot(dx2, dy2)
		if len1 < epsilon || len2 < epsilon {
			p.LineTo(cur.X, cur.Y)
			continue
		}
		nx1, ny1, nx2, ny2 := dx1/len1, dy1/len1, dx2/len2, dy2/len2
		angle := math.Acos(max(-1, min(1, nx1*nx2+ny1*ny2)))
		if angle < epsilon || math.Abs(math.Pi-angle) < epsilon {
			p.LineTo(cur.X, cur.Y)
			continue
		}
		cut := min(radius/math.Sin(angle/2), len1/2, len2/2)
		p.LineTo(cur.X-nx1*cut, cur.Y-ny1*cut)
		p.QuadTo(cur.X, cur.Y, cur.X+nx2*cut, cur.Y+ny2*cut)
	}
	last := pts[len(pts)-1]
	return p.LineTo(last.X, last.Y).Segs
}

// trimmed is segs without the first a and the last b of its length: what the neo
// look's dash array "0 a l-a-b b" draws. When a and b add up to more than the path,
// the array has a negative length, which Chrome does not take, and draws all of it.
func trimmed(segs []scene.Seg, a, b float64) []scene.Seg {
	total := arcLength(segs)
	if a == 0 && b == 0 || total-a-b < 0 {
		return segs
	}
	return span(segs, a, total-b)
}

// dotted is a dotted line as the neo look draws it: generateDashArray's
// "0 a 2 2 … 2 2 b", as many 2 2 as fit between a and b, which SVG takes twice for
// its odd length. That leaves the first a undrawn, then dots 2 on and 2 off, then
// b and the rest solid; with no a, the rest after a gap of 2. A dot here is a dash
// of none under the round cap, set at the middle of Chrome's.
func dotted(segs []scene.Seg, a, b float64, st scene.Stroke) []scene.Path {
	total := arcLength(segs)
	n := max(0, math.Floor((total-a-b)/4))
	end := a + 4*n
	var out []scene.Path
	if n > 0 {
		dots := st
		dots.Dash = []float64{0, 4}
		out = append(out, scene.Path{Segs: span(segs, a+1, end-1), Stroke: dots})
	}
	solid := [][2]float64{{end, total}}
	if a == 0 {
		solid = [][2]float64{{end, min(end+b, total)}, {end + b + 2, total}}
	}
	for _, s := range solid {
		if s[1] > s[0] {
			out = append(out, scene.Path{Segs: span(segs, s[0], s[1]), Stroke: st})
		}
	}
	return out
}

// strokes are what shows of e's line, segs, styled by s: a fill its author set, under
// the parts that are stroked. Those are all but the gaps under its markers, or the
// dots and solid stretches of a dotted line, or the author's dashes, which leave no
// gaps; none when the line is no width, or is invisible and its author set it no
// stroke.
func strokes(e Edge, segs []scene.Seg, s styling) []scene.Path {
	w, ok := widths[e.Stroke]
	if !ok {
		w = 2
	}
	var out []scene.Path
	if s.fill != nil && s.fill.A > 0 {
		out = append(out, scene.Path{Segs: segs, Fill: scene.Fixed(*s.fill)})
	}
	st := s.pen(scene.LineColor, w)
	if st.Width == 0 || e.Stroke == "invisible" && s.stroke == nil { // the class sets no stroke
		return out
	}
	a, b := gap[e.StartMarker], gap[e.EndMarker]
	switch {
	case s.dashed:
		return append(out, dashed(segs, s.dash, st, 0)...)
	case e.Stroke == "dotted":
		return append(out, dotted(segs, a, b, st)...)
	}
	return append(out, scene.Path{Segs: trimmed(segs, a, b), Stroke: st})
}

// heading is the way a path points at its end, from its last bend that has a
// length, or at its start, into the path, when start is set. That bend is a line:
// before rounded's first curve, and after its last, it draws only lines, and the
// one that meets the curve is at least half its side long.
func heading(segs []scene.Seg, start bool) (scene.Point, bool) {
	ps := bends(segs)
	if !start {
		slices.Reverse(ps)
	}
	for _, p := range ps {
		from, to := p[0], p.end()
		if d := math.Hypot(to.X-from.X, to.Y-from.Y); d > 0 {
			return scene.Point{X: (to.X - from.X) / d, Y: (to.Y - from.Y) / d}, true
		}
	}
	return scene.Point{}, false
}

// marker draws an end's marker of a line: at e, where the path ends, with u the
// way the path points there (orient="auto"). Sizes are the -margin markers' in
// user space: the viewBox scaled to markerWidth and markerHeight, refX and refY at e.
// c is the color addEdgeMarker gives the marker's stroke, and an arrow's fill; a
// circle's stroke is no width, so it keeps the theme's fill.
func marker(kind string, e, u scene.Point, start bool, c *scene.Color) []scene.Path {
	n := scene.Point{X: -u.Y, Y: u.X}
	at := func(along, across float64) scene.Point {
		return scene.Point{X: e.X + along*u.X + across*n.X, Y: e.Y + along*u.Y + across*n.Y}
	}
	fill := paint(c, scene.LineColor)
	switch kind {
	case "arrow_point":
		// pointEnd-margin: viewBox 11.5 by 14 in 10.5 by 14, the tip at the end;
		// pointStart-margin: unscaled, the tip 1 beyond the start
		tip, base, half := at(0, 0), -10.5, 7*10.5/11.5
		if start {
			tip, base, half = at(-1, 0), 10.5, 7
		}
		b1, b2 := at(base, -half), at(base, half)
		p := (&scene.Path{}).MoveTo(tip.X, tip.Y).LineTo(b1.X, b1.Y).LineTo(b2.X, b2.Y).Close()
		if start {
			fill = scene.Use(scene.LineColor) // a <polygon>, which the colored copy leaves as it is
		}
		return []scene.Path{{Segs: p.Segs, Fill: fill}}
	case "arrow_circle":
		// circleEnd-margin and circleStart-margin: a radius of 5 scaled by 1.4
		o := at(-10.15, 0)
		if start {
			o = at(9.8, 0)
		}
		return []scene.Path{{Segs: scene.Ellipse(o, 7, 7), Fill: scene.Use(scene.LineColor)}}
	case "arrow_cross":
		// crossEnd-margin and crossStart-margin: viewBox 15 by 15 in 12 by 12, arms 2.5 wide
		near, far := -13.36, -2.96
		if start {
			near, far = 3.6, 14
		}
		a1, a2, b1, b2 := at(near, -5.2), at(far, 5.2), at(near, 5.2), at(far, -5.2)
		p := (&scene.Path{}).MoveTo(a1.X, a1.Y).LineTo(a2.X, a2.Y).MoveTo(b1.X, b1.Y).LineTo(b2.X, b2.Y)
		return []scene.Path{{Segs: p.Segs, Stroke: scene.Stroke{Paint: fill, Width: 2}}}
	}
	return nil
}

// drawEdge is e's item: what shows of its line, and its markers. Its box is the
// whole line's, as getBBox gives a path's.
func drawEdge(e Edge, pts []elk.Vec) scene.Item {
	it := scene.Item{Kind: scene.Edge, ID: e.ID, From: e.Start, To: e.End}
	segs := line(pts, e.StartMarker, e.EndMarker)
	if len(segs) == 0 {
		return it
	}
	it.Box = bounds(segs)
	it.Paths = strokes(e, segs, edgeStyling(e))
	c := markerColor(e)
	if u, ok := heading(segs, true); ok {
		it.Paths = append(it.Paths, marker(e.StartMarker, segs[0].Pts[0], u, true, c)...)
	}
	if u, ok := heading(segs, false); ok {
		// rounded ends on a line to the last point
		it.Paths = append(it.Paths, marker(e.EndMarker, segs[len(segs)-1].Pts[0], u, false, c)...)
	}
	return it
}

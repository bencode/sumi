// Package line draws the lines of a diagram as Mermaid's neo look does: the path of an
// edge with its corners rounded and its hops, what shows of it under a dash array, and
// the markers at its ends (edges.js, markers.js, lineJump.ts). It knows nothing of any
// diagram: a line here is its points and its markers.
package line

import (
	"math"
	"slices"

	"github.com/bencode/sumi/scene"
)

const cornerRadius = 5 // generateRoundedPath's radius

// pullBack is markerOffsets: how far a marker moves its end of the line in.
var pullBack = map[string]float64{"arrow_point": 4, "arrow_barb_neo": 5.5}

// Gap is markerOffsets2: how much of the line the neo look leaves undrawn at an end,
// under its marker, with stroke-dasharray.
var Gap = map[string]float64{"arrow_point": 4, "arrow_circle": 12.5, "arrow_cross": 12.5}

// Line is an edge's path as Mermaid writes its d: the layout's points, their ends
// moved in for the markers, the corners rounded, with the hops of cs.
func Line(pts []scene.Point, start, end string, cs []Crossing) []scene.Seg {
	ps := slices.Clone(pts)
	if n := len(ps); n >= 2 {
		ps[0] = moveIn(ps[0], ps[1], pullBack[start])
		ps[n-1] = moveIn(ps[n-1], ps[n-2], pullBack[end])
	}
	return rounded(ps, cornerRadius, cs)
}

// moveIn is applyMarkerOffsetsToPoints for one end: p moved by d toward q.
func moveIn(p, q scene.Point, d float64) scene.Point {
	if d == 0 {
		return p
	}
	a := math.Atan2(q.Y-p.Y, q.X-p.X)
	return scene.Point{X: p.X + d*math.Cos(a), Y: p.Y + d*math.Sin(a)}
}

// cornerCut is how far back along either side generateRoundedPath's curve about cur
// starts: no more than half way, and 0 where it draws no curve.
func cornerCut(prev, cur, next scene.Point, radius float64) float64 {
	const epsilon = 1e-5
	dx1, dy1 := cur.X-prev.X, cur.Y-prev.Y
	dx2, dy2 := next.X-cur.X, next.Y-cur.Y
	len1, len2 := math.Hypot(dx1, dy1), math.Hypot(dx2, dy2)
	if len1 < epsilon || len2 < epsilon {
		return 0
	}
	angle := math.Acos(max(-1, min(1, (dx1*dx2+dy1*dy2)/(len1*len2))))
	if angle < epsilon || math.Abs(math.Pi-angle) < epsilon {
		return 0
	}
	return min(radius/math.Sin(angle/2), len1/2, len2/2)
}

// rounded is generateRoundedPath, as rewriteEdgePath also draws it: a line through
// pts with a quadratic curve about each corner, and on each segment the hops of cs
// there is room for.
func rounded(pts []scene.Point, radius float64, cs []Crossing) []scene.Seg {
	if len(pts) < 2 {
		return nil
	}
	cut := make([]float64, len(pts))
	for i := 1; i < len(pts)-1; i++ {
		cut[i] = cornerCut(pts[i-1], pts[i], pts[i+1], radius)
	}
	way := func(a, b scene.Point) (scene.Point, float64) {
		l := math.Hypot(b.X-a.X, b.Y-a.Y)
		return scene.Point{X: (b.X - a.X) / l, Y: (b.Y - a.Y) / l}, l
	}
	p := (&scene.Path{}).MoveTo(pts[0].X, pts[0].Y)
	for i := 0; i+1 < len(pts); i++ {
		b := pts[i+1]
		u, l := way(pts[i], b)
		for _, h := range fit(cs, i, l, cut[i], cut[i+1]) {
			arcOver(p, h, u)
		}
		if cut[i+1] == 0 {
			p.LineTo(b.X, b.Y)
			continue
		}
		v, _ := way(b, pts[i+2])
		p.LineTo(b.X-u.X*cut[i+1], b.Y-u.Y*cut[i+1])
		p.QuadTo(b.X, b.Y, b.X+v.X*cut[i+1], b.Y+v.Y*cut[i+1])
	}
	return p.Segs
}

// Trimmed is segs without the first a and the last b of its length: what the neo
// look's dash array "0 a l-a-b b" draws. When a and b add up to more than plain, the
// line's length without its hops, the array has a negative length, which Chrome does
// not take, and draws all of it.
func Trimmed(segs []scene.Seg, a, b, plain float64) []scene.Seg {
	if a == 0 && b == 0 || plain-a-b < 0 {
		return segs
	}
	return Span(segs, a, ArcLength(segs)-b)
}

// Dotted is a dotted line as the neo look draws it: generateDashArray's
// "0 a 2 2 … 2 2 b", as many 2 2 as fit between a and b, which SVG takes twice for
// its odd length, of plain, the line's length without its hops. That leaves the first
// a undrawn, then dots 2 on and 2 off, then b and the rest solid; with no a, the rest
// after a gap of 2. A dot here is a dash of none under the round cap, set at the
// middle of Chrome's.
func Dotted(segs []scene.Seg, a, b, plain float64, st scene.Stroke) []scene.Path {
	total := ArcLength(segs)
	n := max(0, math.Floor((plain-a-b)/4))
	end := a + 4*n
	var out []scene.Path
	if n > 0 {
		dots := st
		dots.Dash = []float64{0, 4}
		out = append(out, scene.Path{Segs: Span(segs, a+1, end-1), Stroke: dots})
	}
	solid := [][2]float64{{end, total}}
	if a == 0 {
		solid = [][2]float64{{end, min(end+b, total)}, {end + b + 2, total}}
	}
	for _, s := range solid {
		if s[1] > s[0] {
			out = append(out, scene.Path{Segs: Span(segs, s[0], s[1]), Stroke: st})
		}
	}
	return out
}

// Heading is the way a path points at its end, from its last bend that has a
// length, or at its start, into the path, when start is set. That bend is a line:
// before rounded's first curve or hop, and after its last, it draws only lines.
func Heading(segs []scene.Seg, start bool) (scene.Point, bool) {
	ps := Bends(segs)
	if !start {
		slices.Reverse(ps)
	}
	for _, p := range ps {
		from, to := p[0], p.End()
		if d := math.Hypot(to.X-from.X, to.Y-from.Y); d > 0 {
			return scene.Point{X: (to.X - from.X) / d, Y: (to.Y - from.Y) / d}, true
		}
	}
	return scene.Point{}, false
}

// Marker draws an end's marker of a line: at e, where the path ends, with u the
// way the path points there (orient="auto"). Sizes are the -margin markers' in
// user space: the viewBox scaled to markerWidth and markerHeight, refX and refY at e.
// fill is the paint addEdgeMarker gives the marker's stroke, and an arrow's fill; a
// circle's stroke is no width, so it keeps the theme's fill.
func Marker(kind string, e, u scene.Point, start bool, fill scene.Paint) []scene.Path {
	n := scene.Point{X: -u.Y, Y: u.X}
	at := func(along, across float64) scene.Point {
		return scene.Point{X: e.X + along*u.X + across*n.X, Y: e.Y + along*u.Y + across*n.Y}
	}
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
	case "arrow_barb_neo":
		// barbEnd-margin: in user space, with no viewBox, refX 17 and refY 7
		p := (&scene.Path{}).MoveTo(at(2, 0).X, at(2, 0).Y).LineTo(at(-6, 7).X, at(-6, 7).Y).
			LineTo(at(-4, 0).X, at(-4, 0).Y).LineTo(at(-6, -7).X, at(-6, -7).Y).Close()
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

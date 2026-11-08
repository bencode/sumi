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

// A bend is a piece of a path, a line or a quadratic curve, from its start.
type bend struct {
	from, ctrl, to scene.Point
	curve          bool
}

func bends(segs []scene.Seg) []bend {
	var out []bend
	var at scene.Point
	for _, s := range segs {
		switch s.Op {
		case scene.LineTo:
			out = append(out, bend{from: at, to: s.Pts[0]})
			at = s.Pts[0]
		case scene.QuadTo:
			out = append(out, bend{from: at, ctrl: s.Pts[0], to: s.Pts[1], curve: true})
			at = s.Pts[1]
		default:
			at = s.Pts[0]
		}
	}
	return out
}

func (p bend) at(t float64) scene.Point {
	if !p.curve {
		return lerp(p.from, p.to, t)
	}
	return lerp(lerp(p.from, p.ctrl, t), lerp(p.ctrl, p.to, t), t)
}

func lerp(a, b scene.Point, t float64) scene.Point {
	return scene.Point{X: a.X + (b.X-a.X)*t, Y: a.Y + (b.Y-a.Y)*t}
}

// gauss are the nodes and weights of 8-point Gauss–Legendre quadrature on [0, 1].
var gauss = [8][2]float64{
	{0.0198550717512319, 0.0506142681451881}, {0.1016667612931866, 0.1111905172266872},
	{0.2372337950418355, 0.1568533229389436}, {0.4082826787521751, 0.1813418916891810},
	{0.5917173212478249, 0.1813418916891810}, {0.7627662049581645, 0.1568533229389436},
	{0.8983332387068134, 0.1111905172266872}, {0.9801449282487681, 0.0506142681451881},
}

// length is the arc length of p from 0 to t.
func (p bend) length(t float64) float64 {
	if !p.curve {
		return t * math.Hypot(p.to.X-p.from.X, p.to.Y-p.from.Y)
	}
	l := 0.0
	for _, g := range gauss {
		u := g[0] * t
		// B'(u) = 2((1-u)(c-p0) + u(p1-c))
		dx := 2 * ((1-u)*(p.ctrl.X-p.from.X) + u*(p.to.X-p.ctrl.X))
		dy := 2 * ((1-u)*(p.ctrl.Y-p.from.Y) + u*(p.to.Y-p.ctrl.Y))
		l += g[1] * math.Hypot(dx, dy)
	}
	return l * t
}

// param is where along p the arc length s is reached.
func (p bend) param(s float64) float64 {
	total := p.length(1)
	if total == 0 {
		return 0
	}
	if !p.curve {
		return s / total
	}
	lo, hi := 0.0, 1.0
	for range 50 {
		mid := (lo + hi) / 2
		if p.length(mid) < s {
			lo = mid
		} else {
			hi = mid
		}
	}
	return (lo + hi) / 2
}

// sub is the part of p from t0 to t1. A quadratic curve's part has its control
// point at the blossom of t0 and t1.
func (p bend) sub(t0, t1 float64) bend {
	q := bend{from: p.at(t0), to: p.at(t1), curve: p.curve}
	if p.curve {
		q.ctrl = lerp(lerp(p.from, p.ctrl, t0), lerp(p.ctrl, p.to, t0), t1)
	}
	return q
}

// arcLength is how long segs is.
func arcLength(segs []scene.Seg) float64 {
	l := 0.0
	for _, p := range bends(segs) {
		l += p.length(1)
	}
	return l
}

// span is the part of segs from length from to length to along it.
func span(segs []scene.Seg, from, to float64) []scene.Seg {
	path := &scene.Path{}
	at := 0.0
	for _, p := range bends(segs) {
		l := p.length(1)
		lo, hi := max(from-at, 0), min(to-at, l)
		at += l
		if lo > hi || l == 0 {
			continue
		}
		q := p.sub(p.param(lo), p.param(hi))
		if len(path.Segs) == 0 {
			path.MoveTo(q.from.X, q.from.Y)
		}
		if q.curve {
			path.QuadTo(q.ctrl.X, q.ctrl.Y, q.to.X, q.to.Y)
		} else {
			path.LineTo(q.to.X, q.to.Y)
		}
	}
	return path.Segs
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

// strokes are the parts of e's line, segs, that show: all but the gaps under its
// markers, or the dots and solid stretches of a dotted line; none when invisible.
func strokes(e Edge, segs []scene.Seg) []scene.Path {
	w, ok := widths[e.Stroke]
	if !ok {
		w = 2
	}
	if w == 0 {
		return nil
	}
	st := scene.Stroke{Paint: scene.Use(scene.LineColor), Width: w}
	a, b := gap[e.StartMarker], gap[e.EndMarker]
	if e.Stroke == "dotted" {
		return dotted(segs, a, b, st)
	}
	return []scene.Path{{Segs: trimmed(segs, a, b), Stroke: st}}
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
		if d := math.Hypot(p.to.X-p.from.X, p.to.Y-p.from.Y); d > 0 {
			return scene.Point{X: (p.to.X - p.from.X) / d, Y: (p.to.Y - p.from.Y) / d}, true
		}
	}
	return scene.Point{}, false
}

// marker draws an end's marker of a line: at e, where the path ends, with u the
// way the path points there (orient="auto"). Sizes are the -margin markers' in
// user space: the viewBox scaled to markerWidth and markerHeight, refX and refY at e.
func marker(kind string, e, u scene.Point, start bool) []scene.Path {
	n := scene.Point{X: -u.Y, Y: u.X}
	at := func(along, across float64) scene.Point {
		return scene.Point{X: e.X + along*u.X + across*n.X, Y: e.Y + along*u.Y + across*n.Y}
	}
	fill := scene.Use(scene.LineColor)
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
		return []scene.Path{{Segs: p.Segs, Fill: fill}}
	case "arrow_circle":
		// circleEnd-margin and circleStart-margin: a radius of 5 scaled by 1.4
		c := at(-10.15, 0)
		if start {
			c = at(9.8, 0)
		}
		return []scene.Path{{Segs: scene.Ellipse(c, 7, 7), Fill: fill}}
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
	it.Paths = strokes(e, segs)
	if len(it.Paths) == 0 {
		return it
	}
	if u, ok := heading(segs, true); ok {
		it.Paths = append(it.Paths, marker(e.StartMarker, segs[0].Pts[0], u, true)...)
	}
	if u, ok := heading(segs, false); ok {
		// rounded ends on a line to the last point
		it.Paths = append(it.Paths, marker(e.EndMarker, segs[len(segs)-1].Pts[0], u, false)...)
	}
	return it
}

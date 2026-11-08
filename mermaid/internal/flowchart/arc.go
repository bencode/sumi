package flowchart

import (
	"math"

	"github.com/bencode/sumi/scene"
)

// Lengths along paths, and the parts of paths between two lengths: what Chrome's
// getTotalLength and stroke-dasharray go by.

// A bend is a piece of a path from its start: a line, a quadratic or a cubic curve,
// as its two, three or four points.
type bend []scene.Point

// bends are the pieces of one subpath, segs; Close is the line back to its start.
func bends(segs []scene.Seg) []bend {
	var out []bend
	var at, start scene.Point
	for _, s := range segs {
		switch s.Op {
		case scene.MoveTo:
			at, start = s.Pts[0], s.Pts[0]
			continue
		case scene.LineTo:
			out = append(out, bend{at, s.Pts[0]})
		case scene.QuadTo:
			out = append(out, bend{at, s.Pts[0], s.Pts[1]})
		case scene.CubeTo:
			out = append(out, bend{at, s.Pts[0], s.Pts[1], s.Pts[2]})
		case scene.Close:
			out = append(out, bend{at, start})
		}
		at = out[len(out)-1].end()
	}
	return out
}

// subpaths are segs cut at each MoveTo.
func subpaths(segs []scene.Seg) [][]scene.Seg {
	var out [][]scene.Seg
	for i, s := range segs {
		if s.Op == scene.MoveTo || i == 0 {
			out = append(out, nil)
		}
		out[len(out)-1] = append(out[len(out)-1], s)
	}
	return out
}

func (p bend) end() scene.Point { return p[len(p)-1] }

// blossom is p's polar form at ts, one for each of its degrees: with every t the
// same, the point at t; with t0s then t1s, a control point of p's part from t0 to t1.
func (p bend) blossom(ts ...float64) scene.Point {
	q := append(bend{}, p...)
	for _, t := range ts {
		for i := range len(q) - 1 {
			q[i] = lerp(q[i], q[i+1], t)
		}
		q = q[:len(q)-1]
	}
	return q[0]
}

func (p bend) at(t float64) scene.Point {
	ts := make([]float64, len(p)-1)
	for i := range ts {
		ts[i] = t
	}
	return p.blossom(ts...)
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
	if len(p) == 2 {
		return t * math.Hypot(p[1].X-p[0].X, p[1].Y-p[0].Y)
	}
	n := float64(len(p) - 1)
	d := make(bend, len(p)-1)
	for i := range d {
		d[i] = scene.Point{X: n * (p[i+1].X - p[i].X), Y: n * (p[i+1].Y - p[i].Y)}
	}
	l := 0.0
	for _, g := range gauss {
		v := scene.Point{X: d[0].X, Y: d[0].Y}
		if len(d) > 1 {
			v = d.at(g[0] * t)
		}
		l += g[1] * math.Hypot(v.X, v.Y)
	}
	return l * t
}

// param is where along p the arc length s is reached.
func (p bend) param(s float64) float64 {
	total := p.length(1)
	if total == 0 {
		return 0
	}
	if len(p) == 2 {
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

// sub is the part of p from t0 to t1, its control points p's blossoms.
func (p bend) sub(t0, t1 float64) bend {
	n := len(p) - 1
	q := make(bend, len(p))
	for i := range q {
		ts := make([]float64, n)
		for j := range ts {
			ts[j] = t0
			if j >= n-i {
				ts[j] = t1
			}
		}
		q[i] = p.blossom(ts...)
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

// span is the part of one subpath, segs, from length from to length to along it.
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
			path.MoveTo(q[0].X, q[0].Y)
		}
		switch len(q) {
		case 2:
			path.LineTo(q[1].X, q[1].Y)
		case 3:
			path.QuadTo(q[1].X, q[1].Y, q[2].X, q[2].Y)
		default:
			path.CubeTo(q[1].X, q[1].Y, q[2].X, q[2].Y, q[3].X, q[3].Y)
		}
	}
	return path.Segs
}

// dashed is the line of segs, stroked by st, dashed by dash as Chrome dashes it with
// butt ends, drawn with st's round ones: each subpath takes the array from its start
// again, an odd one twice, and each dash is drawn in from its ends by as far as the
// round cap reaches out, so that it covers what Chrome's covers. A dash shorter than
// the stroke is wide is a dot at its middle; one of no length draws nothing. So is
// what the subpath's end leaves of a dash when that is shorter than the stroke is
// wide, as it is all along roughjs's short pieces of a curve. turn is how much of each
// subpath's length Chrome's starts past where segs's does.
func dashed(segs []scene.Seg, dash []float64, st scene.Stroke, turn float64) []scene.Path {
	if len(dash)%2 == 1 {
		dash = append(append([]float64{}, dash...), dash...)
	}
	sum := 0.0
	for _, d := range dash {
		sum += d
	}
	if sum == 0 {
		return []scene.Path{{Segs: segs, Stroke: st}}
	}
	w := st.Width
	ons, whole := periodDashes(dash, w)
	if len(ons) == 0 {
		return nil
	}
	d := st
	d.Dash = pattern(ons, sum)
	var out []scene.Path
	for _, sp := range subpaths(segs) {
		l := arcLength(sp)
		if start := turn * l; start > 0 && start < l {
			sp = append(span(sp, start, l), span(sp, 0, start)[1:]...)
		}
		if s := span(sp, ons[0][0], l-w/2); len(s) > 0 {
			out = append(out, scene.Path{Segs: s, Stroke: d})
		}
		if m, ok := lastDot(ons, whole, sum, l, w); ok {
			out = append(out, scene.Path{Segs: span(sp, m, m), Stroke: st})
		}
	}
	return out
}

// periodDashes are where the dashes of a period of dash start and end: ons in the
// terms of round caps w wide, whole as Chrome draws them with butt ends.
func periodDashes(dash []float64, w float64) (ons, whole [][2]float64) {
	at := 0.0
	for i := 0; i < len(dash); i += 2 {
		if d := dash[i]; d > 0 {
			on := max(d-w, 0)
			from := at + (d-on)/2
			ons = append(ons, [2]float64{from, from + on})
			whole = append(whole, [2]float64{at, at + d})
		}
		at += dash[i] + dash[i+1]
	}
	return ons, whole
}

// pattern is the array that draws ons, in a period of sum, from the first of them.
func pattern(ons [][2]float64, sum float64) []float64 {
	var out []float64
	for i, on := range ons {
		next := ons[(i+1)%len(ons)][0]
		if i == len(ons)-1 {
			next += sum
		}
		out = append(out, on[1]-on[0], next-on[1])
	}
	return out
}

// lastDot is the middle of the last dash Chrome draws of a subpath l long, when the
// array stops short of it, at l-w/2: ons are where a period's dashes are drawn from
// and to, whole where Chrome's are, and sum the period.
func lastDot(ons, whole [][2]float64, sum, l, w float64) (float64, bool) {
	n := math.Floor(l / sum)
	for _, k := range []float64{n, n - 1} {
		for i := len(whole) - 1; i >= 0 && k >= 0; i-- {
			from := k*sum + whole[i][0]
			if from < l-1e-9 { // a dash that starts at the end, within a float, is none
				return (from + min(k*sum+whole[i][1], l)) / 2, k*sum+ons[i][0] > l-w/2
			}
		}
	}
	return 0, false
}

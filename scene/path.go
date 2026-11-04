package scene

import "math"

// An Op is a command of a path.
type Op uint8

const (
	MoveTo Op = iota // start a subpath at Pts[0]
	LineTo           // a line to Pts[0]
	QuadTo           // a quadratic curve about Pts[0] to Pts[1]
	CubeTo           // a cubic curve about Pts[0] and Pts[1] to Pts[2]
	Close            // a line back to the subpath's start
)

// A Seg is one command of a path and the points it takes.
type Seg struct {
	Op  Op
	Pts [3]Point
}

// A Path is a shape: its outline, how it is filled, and how it is stroked.
// Every shape of a scene is a path; circles and rounded corners are cubic
// curves (see Ellipse and RoundRect).
type Path struct {
	Segs   []Seg
	Fill   Paint
	Stroke Stroke
}

// A Stroke is how a path's outline is drawn. Its ends and corners are round.
type Stroke struct {
	Paint Paint
	Width float64
	Dash  []float64 // lengths of dashes and gaps in turn; nil draws a solid line
}

func (p *Path) add(op Op, pts ...Point) *Path {
	s := Seg{Op: op}
	copy(s.Pts[:], pts)
	p.Segs = append(p.Segs, s)
	return p
}

func (p *Path) MoveTo(x, y float64) *Path { return p.add(MoveTo, Point{x, y}) }
func (p *Path) LineTo(x, y float64) *Path { return p.add(LineTo, Point{x, y}) }
func (p *Path) Close() *Path              { return p.add(Close) }

func (p *Path) QuadTo(cx, cy, x, y float64) *Path {
	return p.add(QuadTo, Point{cx, cy}, Point{x, y})
}

func (p *Path) CubeTo(c1x, c1y, c2x, c2y, x, y float64) *Path {
	return p.add(CubeTo, Point{c1x, c1y}, Point{c2x, c2y}, Point{x, y})
}

// kappa places the control points of a cubic curve that draws a quarter of a
// circle: off by less than 0.03% of the radius.
const kappa = 0.5523

// Ellipse is the outline of an ellipse about c, four cubic curves clockwise
// from its top.
func Ellipse(c Point, rx, ry float64) []Seg {
	kx, ky := rx*kappa, ry*kappa
	p := (&Path{}).
		MoveTo(c.X, c.Y-ry).
		CubeTo(c.X+kx, c.Y-ry, c.X+rx, c.Y-ky, c.X+rx, c.Y).
		CubeTo(c.X+rx, c.Y+ky, c.X+kx, c.Y+ry, c.X, c.Y+ry).
		CubeTo(c.X-kx, c.Y+ry, c.X-rx, c.Y+ky, c.X-rx, c.Y).
		CubeTo(c.X-rx, c.Y-ky, c.X-kx, c.Y-ry, c.X, c.Y-ry).
		Close()
	return p.Segs
}

// RoundRect is the outline of r with its corners rounded by radius, which is
// held to half its shorter side. A radius of 0 gives square corners.
func RoundRect(r Rect, radius float64) []Seg {
	x0, y0, x1, y1 := r.X, r.Y, r.X+r.W, r.Y+r.H
	rad := max(min(radius, r.W/2, r.H/2), 0)
	p := &Path{}
	if rad == 0 {
		return p.MoveTo(x0, y0).LineTo(x1, y0).LineTo(x1, y1).LineTo(x0, y1).Close().Segs
	}
	k := rad * (1 - kappa)
	return p.MoveTo(x0+rad, y0).
		LineTo(x1-rad, y0).CubeTo(x1-k, y0, x1, y0+k, x1, y0+rad).
		LineTo(x1, y1-rad).CubeTo(x1, y1-k, x1-k, y1, x1-rad, y1).
		LineTo(x0+rad, y1).CubeTo(x0+k, y1, x0, y1-k, x0, y1-rad).
		LineTo(x0, y0+rad).CubeTo(x0, y0+k, x0+k, y0, x0+rad, y0).
		Close().Segs
}

// Arrowhead is a filled arrowhead size long pointing at tip from the way of
// from, and end, where a line from from should stop so that it does not poke
// through the head's point: the middle of the head's base.
func Arrowhead(from, tip Point, size float64) (head []Seg, end Point) {
	dx, dy := tip.X-from.X, tip.Y-from.Y
	n := math.Hypot(dx, dy)
	if n == 0 {
		return nil, tip
	}
	ux, uy := dx/n, dy/n                   // along the line
	bx, by := tip.X-ux*size, tip.Y-uy*size // middle of the base
	w := size * 0.4                        // half the base's width
	p := (&Path{}).
		MoveTo(tip.X, tip.Y).
		LineTo(bx-uy*w, by+ux*w).
		LineTo(bx+uy*w, by-ux*w).
		Close()
	return p.Segs, Point{bx, by}
}

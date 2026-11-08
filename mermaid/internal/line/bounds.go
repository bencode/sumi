package line

import (
	"math"

	"github.com/bencode/sumi/scene"
)

// Bounds is the tight box of a path: its ends, and where its curves turn.
func Bounds(segs []scene.Seg) scene.Rect {
	x0, y0, x1, y1 := math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1)
	add := func(p scene.Point) {
		x0, y0, x1, y1 = min(x0, p.X), min(y0, p.Y), max(x1, p.X), max(y1, p.Y)
	}
	var at scene.Point
	for _, s := range segs {
		switch s.Op {
		case scene.MoveTo, scene.LineTo:
			at = s.Pts[0]
		case scene.QuadTo:
			for _, t := range quadTurns(at, s.Pts[0], s.Pts[1]) {
				add(Bend{at, s.Pts[0], s.Pts[1]}.At(t))
			}
			at = s.Pts[1]
		case scene.CubeTo:
			for _, t := range cubeTurns(at, s.Pts[0], s.Pts[1], s.Pts[2]) {
				add(Bend{at, s.Pts[0], s.Pts[1], s.Pts[2]}.At(t))
			}
			at = s.Pts[2]
		default:
			continue
		}
		add(at)
	}
	if x0 > x1 {
		return scene.Rect{}
	}
	return scene.Rect{X: x0, Y: y0, W: x1 - x0, H: y1 - y0}
}

// quadTurns are where a quadratic curve's x or y turns, inside (0, 1).
func quadTurns(p0, c, p1 scene.Point) []float64 {
	var ts []float64
	for _, v := range [][3]float64{{p0.X, c.X, p1.X}, {p0.Y, c.Y, p1.Y}} {
		if d := v[0] - 2*v[1] + v[2]; d != 0 {
			if t := (v[0] - v[1]) / d; t > 0 && t < 1 {
				ts = append(ts, t)
			}
		}
	}
	return ts
}

// cubeTurns are where a cubic curve's x or y turns, inside (0, 1): the roots of
// its derivative, a·t² + b·t + c.
func cubeTurns(p0, c1, c2, p1 scene.Point) []float64 {
	var ts []float64
	for _, v := range [][4]float64{{p0.X, c1.X, c2.X, p1.X}, {p0.Y, c1.Y, c2.Y, p1.Y}} {
		a := -v[0] + 3*v[1] - 3*v[2] + v[3]
		b := 2 * (v[0] - 2*v[1] + v[2])
		c := v[1] - v[0]
		var roots []float64
		switch disc := b*b - 4*a*c; {
		case math.Abs(a) < 1e-12 && b != 0:
			roots = []float64{-c / b}
		case math.Abs(a) >= 1e-12 && disc >= 0:
			q := math.Sqrt(disc)
			roots = []float64{(-b + q) / (2 * a), (-b - q) / (2 * a)}
		}
		for _, t := range roots {
			if t > 0 && t < 1 {
				ts = append(ts, t)
			}
		}
	}
	return ts
}

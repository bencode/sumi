package shape

import (
	"math"
	"slices"

	"github.com/bencode/sumi/scene"
)

// polygon is a shape drawn through pts, moved by off about the node's center:
// insertPolygonShape's polygon, or roughjs's group of two paths through the
// points, whose box is the points' too.
func polygon(pts []scene.Point, off scene.Point) Node {
	l, t, r, b := pointsBox(pts)
	return Node{
		W:         r - l,
		H:         b - t,
		Outline:   outline(pts, off),
		Intersect: meetPolygon(pts),
	}
}

// inserted places pts as insertPolygonShape does, by the box w by h they are
// drawn about.
func inserted(w, h float64, pts []scene.Point) Node {
	return polygon(pts, scene.Point{X: -w / 2, Y: h / 2})
}

// pointsBox is the box of pts: its left, top, right and bottom.
func pointsBox(pts []scene.Point) (l, t, r, b float64) {
	l, t, r, b = math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1)
	for _, p := range pts {
		l, t, r, b = math.Min(l, p.X), math.Min(t, p.Y), math.Max(r, p.X), math.Max(b, p.Y)
	}
	return l, t, r, b
}

func outline(pts []scene.Point, off scene.Point) []scene.Seg {
	p := &scene.Path{}
	p.MoveTo(pts[0].X+off.X, pts[0].Y+off.Y)
	for _, q := range pts[1:] {
		p.LineTo(q.X+off.X, q.Y+off.Y)
	}
	return p.Close().Segs
}

// meetPolygon is intersect-polygon.js for pts: it places them in the node's
// box by their least x and y, and takes the meeting nearest the point.
func meetPolygon(pts []scene.Point) func(scene.Point, float64, float64, scene.Point) scene.Point {
	minX, minY, _, _ := pointsBox(pts)
	return func(c scene.Point, w, h float64, toward scene.Point) scene.Point {
		left, top := c.X-w/2-minX, c.Y-h/2-minY
		var hits []scene.Point
		for i, p := range pts {
			q := pts[(i+1)%len(pts)]
			if m, ok := meetLine(c, toward, scene.Point{X: left + p.X, Y: top + p.Y}, scene.Point{X: left + q.X, Y: top + q.Y}); ok {
				hits = append(hits, m)
			}
		}
		if len(hits) == 0 {
			return c
		}
		dist := func(m scene.Point) float64 {
			dx, dy := m.X-toward.X, m.Y-toward.Y
			return math.Hypot(dx, dy)
		}
		slices.SortStableFunc(hits, func(a, b scene.Point) int {
			da, db := dist(a), dist(b)
			if da < db {
				return -1
			}
			if da == db {
				return 0
			}
			return 1
		})
		return hits[0]
	}
}

// meetLine is intersect-line.js: where segment p1 p2 meets segment q1 q2.
func meetLine(p1, p2, q1, q2 scene.Point) (scene.Point, bool) {
	a1, b1 := p2.Y-p1.Y, p1.X-p2.X
	c1 := p2.X*p1.Y - p1.X*p2.Y
	r3 := a1*q1.X + b1*q1.Y + c1
	r4 := a1*q2.X + b1*q2.Y + c1
	if r3 != 0 && r4 != 0 && r3*r4 > 0 {
		return scene.Point{}, false
	}
	a2, b2 := q2.Y-q1.Y, q1.X-q2.X
	c2 := q2.X*q1.Y - q1.X*q2.Y
	r1 := a2*p1.X + b2*p1.Y + c2
	r2 := a2*p2.X + b2*p2.Y + c2
	const epsilon = 1e-6
	if math.Abs(r1) < epsilon && math.Abs(r2) < epsilon && r1*r2 > 0 {
		return scene.Point{}, false
	}
	denom := a1*b2 - a2*b1
	if denom == 0 {
		return scene.Point{}, false
	}
	return scene.Point{
		X: (b1*c2 - b2*c1) / denom,
		Y: (a2*c1 - a1*c2) / denom,
	}, true
}

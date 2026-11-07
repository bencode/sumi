package shape

import (
	"math"
	"slices"

	"github.com/bencode/sumi/scene"
)

// polygon is a shape drawn through pts, moved by off about the node's center:
// insertPolygonShape's polygon, or roughjs's group of two paths through the
// points. A group's box is the union of its paths', which here is the box of
// either: the shapes roughjs draws, stadium and odd, have boxes that sit evenly
// about 0 or numbers on a 1/64 grid, so the union adds nothing.
func polygon(pts []scene.Point, off scene.Point) Node {
	r := pointsRect(pts)
	return Node{
		W:         float64(r.w),
		H:         float64(r.h),
		Outline:   outline(pts, off),
		Intersect: meetPolygon(pts),
	}
}

// pointsRect is the getBBox of a polygon or path through pts.
func pointsRect(pts []scene.Point) rectF {
	b := newBox(pt{svg(pts[0].X), svg(pts[0].Y)})
	for _, p := range pts[1:] {
		b.add(pt{svg(p.X), svg(p.Y)})
	}
	return boundingRect(b)
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
	minX, minY := math.Inf(1), math.Inf(1)
	for _, p := range pts {
		minX, minY = math.Min(minX, p.X), math.Min(minY, p.Y)
	}
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
			return math.Sqrt(float64(dx*dx) + float64(dy*dy))
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
// Products are converted one by one so that Go does not fuse them, as V8 does not.
func meetLine(p1, p2, q1, q2 scene.Point) (scene.Point, bool) {
	a1, b1 := p2.Y-p1.Y, p1.X-p2.X
	c1 := float64(p2.X*p1.Y) - float64(p1.X*p2.Y)
	r3 := float64(a1*q1.X) + float64(b1*q1.Y) + c1
	r4 := float64(a1*q2.X) + float64(b1*q2.Y) + c1
	if r3 != 0 && r4 != 0 && r3*r4 > 0 {
		return scene.Point{}, false
	}
	a2, b2 := q2.Y-q1.Y, q1.X-q2.X
	c2 := float64(q2.X*q1.Y) - float64(q1.X*q2.Y)
	r1 := float64(a2*p1.X) + float64(b2*p1.Y) + c2
	r2 := float64(a2*p2.X) + float64(b2*p2.Y) + c2
	const epsilon = 1e-6
	if math.Abs(r1) < epsilon && math.Abs(r2) < epsilon && r1*r2 > 0 {
		return scene.Point{}, false
	}
	denom := float64(a1*b2) - float64(a2*b1)
	if denom == 0 {
		return scene.Point{}, false
	}
	return scene.Point{
		X: (float64(b1*c2) - float64(b2*c1)) / denom,
		Y: (float64(a2*c1) - float64(a1*c2)) / denom,
	}, true
}

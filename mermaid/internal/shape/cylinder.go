package shape

import (
	"math"

	"github.com/bencode/sumi/scene"
)

// cylinder is cylinder.ts: a box with an ellipse for its top and half of one for
// its bottom. The label sits padding/1.5 below the center, and lines meet the
// ellipses, not the box.
func cylinder(lw, lh, padding float64) Node {
	const pad = 24
	w := lw + pad
	rx := w / 2
	ry := rx / (2.5 + w/50)
	h := lh + pad + ry
	r := cylinderRect(w, h, rx, ry)
	return Node{
		W:         float64(r.w),
		H:         float64(r.h),
		Outline:   cylinderOutline(w, h, rx, ry),
		Label:     scene.Point{Y: padding / 1.5},
		Intersect: meetCylinder(rx, ry),
	}
}

// cylinderRect is the getBBox of the path createCylinderPathD draws at 0, 0:
// its numbers as Blink reads them and its arcs as Skia's conics.
func cylinderRect(w, h, rx, ry float64) rectF {
	w32, rx32, ry32 := svg(w), svg(rx), svg(ry)
	at := pt{0, ry32}
	b := newBox(at)
	arcBy := func(dx float32) {
		to := at.add(pt{dx, 0})
		for _, c := range arc(at, to, rx32, ry32, false, false) {
			b.addConic(c)
		}
		b.add(to) // the arc's end, also when arc gives no conics for a line
		at = to
	}
	lineBy := func(dy float32) {
		at = at.add(pt{0, dy})
		b.add(at)
	}
	arcBy(w32)
	arcBy(svg(-w))
	lineBy(svg(h))
	arcBy(w32)
	lineBy(svg(-h))
	return boundingRect(b)
}

// cylinderOutline is createCylinderPathD's path about the node's center: the
// top ellipse, its lower half first, then down the left side, along the lower
// half of the bottom ellipse and up the right side. The arcs are the quarters
// of scene.Ellipse.
func cylinderOutline(w, h, rx, ry float64) []scene.Seg {
	top := scene.Ellipse(scene.Point{Y: -h / 2}, rx, ry)
	bottom := scene.Ellipse(scene.Point{Y: h / 2}, rx, ry)
	p := (&scene.Path{}).MoveTo(-w/2, -h/2)
	p.Segs = append(p.Segs, back(top, 2), back(top, 1), back(top, 0), back(top, 3))
	p.LineTo(-w/2, h/2)
	p.Segs = append(p.Segs, back(bottom, 2), back(bottom, 1))
	return p.LineTo(w/2, -h/2).Segs
}

// back is quarter q of an ellipse scene.Ellipse drew, from its end to its
// start; the quarters run clockwise from the top.
func back(e []scene.Seg, q int) scene.Seg {
	from := e[q].Pts[2]
	if q == 0 {
		from = e[0].Pts[0]
	}
	c := e[q+1].Pts
	return scene.Seg{Op: scene.CubeTo, Pts: [3]scene.Point{c[1], c[0], from}}
}

// meetCylinder is the cylinder's intersect: intersect-rect.js, then moved onto
// the ellipse a line meets above or below the box.
func meetCylinder(rx, ry float64) func(scene.Point, float64, float64, scene.Point) scene.Point {
	return func(c scene.Point, w, h float64, toward scene.Point) scene.Point {
		pos := meetRect(c, w, h, toward)
		x := pos.X - c.X
		if rx != 0 && (math.Abs(x) < w/2 || (math.Abs(x) == w/2 && math.Abs(pos.Y-c.Y) > h/2-ry)) {
			y := ry * ry * (1 - x*x/(rx*rx))
			if y > 0 {
				y = math.Sqrt(y)
			}
			y = ry - y
			if toward.Y-c.Y > 0 {
				y = -y
			}
			pos.Y += y
		}
		return pos
	}
}

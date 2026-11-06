package graphlayout

import (
	"math"

	"github.com/bencode/sumi/mermaid/internal/elk"
)

// These go by their names in geometry.ts, unless a comment gives another. Rounding
// counts: a product is converted before it is added, which keeps Go from fusing the
// two where JavaScript does not.

const (
	borderTolerance = 0.5  // onBorder's default
	nearBorder      = 1    // EPS
	pushOut         = 10   // PUSH_OUT
	outlineSteps    = 20   // OUTLINE_RAY_STEPS
	axisTolerance   = 1e-6 // DEPARTURE_AXIS_EPS
)

func onBorder(b Box, p elk.Vec, tolerance float64) bool {
	left, right := b.X-b.Width/2, b.X+b.Width/2
	top, bottom := b.Y-b.Height/2, b.Y+b.Height/2
	withinX := p.X >= left-tolerance && p.X <= right+tolerance
	withinY := p.Y >= top-tolerance && p.Y <= bottom+tolerance
	return (math.Abs(p.X-left) <= tolerance || math.Abs(p.X-right) <= tolerance) && withinY ||
		(math.Abs(p.Y-top) <= tolerance || math.Abs(p.Y-bottom) <= tolerance) && withinX
}

func outsideNode(b Box, p elk.Vec) bool {
	return math.Abs(p.X-b.X) >= b.Width/2 || math.Abs(p.Y-b.Y) >= b.Height/2
}

// sign is Math.sign.
func sign(x float64) float64 {
	switch {
	case x > 0:
		return 1
	case x < 0:
		return -1
	}
	return 0
}

// intersectRect is where the line from b's center to p leaves b (intersectRect,
// rendering-elements/intersect/intersect-rect.js).
func intersectRect(b Box, p elk.Vec) elk.Vec {
	dx, dy := p.X-b.X, p.Y-b.Y
	w, h := b.Width/2, b.Height/2
	if math.Abs(dy)*w > math.Abs(dx)*h {
		if dy < 0 {
			h = -h
		}
		return elk.Vec{X: b.X + h*dx/dy, Y: b.Y + h}
	}
	if dx < 0 {
		w = -w
	}
	if dx == 0 {
		return elk.Vec{X: b.X + w, Y: b.Y}
	}
	return elk.Vec{X: b.X + w, Y: b.Y + w*dy/dx}
}

// intersection gives where the segment from inside to outside meets b's border.
func intersection(b Box, outside, inside elk.Vec) elk.Vec {
	w, h := b.Width/2, b.Height/2
	q, r := math.Abs(outside.Y-inside.Y), math.Abs(outside.X-inside.X)
	right, down := inside.X < outside.X, inside.Y < outside.Y
	var p elk.Vec
	if math.Abs(b.Y-outside.Y)*w > math.Abs(b.X-outside.X)*h {
		a := b.Y - h - outside.Y
		if down {
			a = outside.Y - h - b.Y
		}
		along := r * a / q
		p = elk.Vec{X: inside.X - r + along, Y: inside.Y - q + a}
		if right {
			p.X = inside.X + along
		}
		if down {
			p.Y = inside.Y + q - a
		}
	} else {
		a := b.X - w - outside.X
		if right {
			a = outside.X - w - b.X
		}
		along := q * a / r
		p = elk.Vec{X: inside.X - r + a, Y: inside.Y - along}
		if right {
			p.X = inside.X + r - a
		}
		if down {
			p.Y = inside.Y + along
		}
	}
	if r == 0 {
		p.X = outside.X
	}
	if q == 0 {
		p.Y = outside.Y
	}
	return p
}

// ensureTrulyOutside pushes a point on b's border out along the ray from its center.
func ensureTrulyOutside(b Box, p elk.Vec) elk.Vec {
	if math.Abs(math.Abs(p.X-b.X)-b.Width/2) >= nearBorder && math.Abs(math.Abs(p.Y-b.Y)-b.Height/2) >= nearBorder {
		return p
	}
	dx, dy := p.X-b.X, p.Y-b.Y
	length := math.Sqrt(float64(dx*dx) + float64(dy*dy))
	if length <= 0 {
		return p
	}
	return elk.Vec{X: b.X + float64(dx/length*(length+pushOut)), Y: b.Y + float64(dy/length*(length+pushOut))}
}

func makeInsidePoint(b Box, outside, center elk.Vec) elk.Vec {
	p := elk.Vec{X: outside.X, Y: center.Y}
	if math.Abs(outside.X-b.X) >= nearBorder {
		p.X = b.X + b.Width/4
		if outside.X < b.X {
			p.X = b.X - b.Width/4
		}
	}
	if math.Abs(outside.Y-b.Y) < nearBorder {
		p.Y = outside.Y
	}
	return p
}

// shapeIntersect is a node's own answer, unless it comes out on the wrong side or on
// top of the point (tryNodeIntersect). Subgraphs have none while Mermaid lays out.
func shapeIntersect(f frame, b Box, outside elk.Vec) (elk.Vec, bool) {
	if f.group {
		return elk.Vec{}, false
	}
	p := f.outline(outside)
	wrongSide := outside.X < b.X && p.X > b.X || outside.X > b.X && p.X < b.X
	if wrongSide || math.Hypot(outside.X-p.X, outside.Y-p.Y) <= nearBorder {
		return elk.Vec{}, false
	}
	return p, true
}

// outline gives where the line from f's center to p leaves its shape.
func (f frame) outline(p elk.Vec) elk.Vec {
	c := f.center()
	b := Box{c.X, c.Y, f.width, f.height}
	if f.intersect == nil {
		return intersectRect(b, p)
	}
	return f.intersect(b, p)
}

// nodeIntersection gives where the way from outside into f meets its outline, or
// else b's border (computeNodeIntersection, fallbackIntersection).
func nodeIntersection(f frame, b Box, outside, center elk.Vec) elk.Vec {
	pushed := ensureTrulyOutside(b, outside)
	if p, ok := shapeIntersect(f, b, pushed); ok {
		return p
	}
	return intersection(b, pushed, makeInsidePoint(b, pushed, center))
}

// insideOutline tells whether probe is in f's outline: no further from its center
// than the outline is that way. It takes the outline to be star-shaped.
func insideOutline(f frame, center, probe elk.Vec) bool {
	crossing := f.outline(probe)
	return math.Hypot(probe.X-center.X, probe.Y-center.Y) <= math.Hypot(crossing.X-center.X, crossing.Y-center.Y)+1e-9
}

// outlineAttachPoint gives where f's outline meets the ray into it from port, against
// the way the edge leaves. ELK routes to the bounding box and leaves it square;
// going along that axis, rather than toward the center, keeps the edge straight. It
// has no answer for a departure off the axes (outlineAttachPoint).
func outlineAttachPoint(f frame, b Box, port, next elk.Vec) (elk.Vec, bool) {
	dx, dy := next.X-port.X, next.Y-port.Y
	if dx == 0 && dy == 0 || math.Abs(dx) > axisTolerance && math.Abs(dy) > axisTolerance {
		return elk.Vec{}, false
	}
	center := elk.Vec{X: b.X, Y: b.Y}
	level := math.Abs(dx) > math.Abs(dy)
	along := func(t float64) elk.Vec {
		if level {
			return elk.Vec{X: t, Y: port.Y}
		}
		return elk.Vec{X: port.X, Y: t}
	}
	inner, outer := center.Y, port.Y
	if level {
		inner, outer = center.X, port.X
	}
	if !insideOutline(f, center, along(inner)) {
		return elk.Vec{}, false
	}
	if insideOutline(f, center, along(outer)) {
		return port, true
	}
	for range outlineSteps {
		mid := (inner + outer) / 2
		if insideOutline(f, center, along(mid)) {
			inner = mid
		} else {
			outer = mid
		}
	}
	return along(inner), true
}

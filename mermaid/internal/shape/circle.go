package shape

import (
	"math"

	"github.com/bencode/sumi/scene"
)

// circleGap is how far doublecircle.ts sets the outer circle from the inner in
// the neo look, and circlePadding how far the inner one is from the label's corners.
const (
	circleGap     = 12
	circlePadding = 16
)

// circle is circle.ts: a circle through points 16 beyond the label's corners.
func circle(lw, lh, _ float64) Node {
	r := labelRadius(lw, lh) + circlePadding
	return Node{
		W:         2 * r,
		H:         2 * r,
		Outline:   scene.Ellipse(scene.Point{}, r, r),
		Intersect: meetEllipse(r, r),
	}
}

// doubleCircle is doublecircle.ts: circle's circle and one 12 beyond it, in a
// group whose box is the outer circle's.
func doubleCircle(lw, lh, _ float64) Node {
	inner := labelRadius(lw, lh) + circlePadding
	outer := inner + circleGap
	return Node{
		W:         2 * outer,
		H:         2 * outer,
		Outline:   append(scene.Ellipse(scene.Point{}, outer, outer), scene.Ellipse(scene.Point{}, inner, inner)...),
		Intersect: meetEllipse(outer, outer),
	}
}

func labelRadius(lw, lh float64) float64 {
	return math.Hypot(lw, lh) / 2
}

// meetEllipse is intersect-ellipse.js for an ellipse of radii rx, ry.
func meetEllipse(rx, ry float64) func(scene.Point, float64, float64, scene.Point) scene.Point {
	return func(c scene.Point, _, _ float64, toward scene.Point) scene.Point {
		px, py := c.X-toward.X, c.Y-toward.Y
		det := math.Sqrt(rx*rx*py*py + ry*ry*px*px)
		dx := math.Abs(rx * ry * px / det)
		if toward.X < c.X {
			dx = -dx
		}
		dy := math.Abs(rx * ry * py / det)
		if toward.Y < c.Y {
			dy = -dy
		}
		return scene.Point{X: c.X + dx, Y: c.Y + dy}
	}
}

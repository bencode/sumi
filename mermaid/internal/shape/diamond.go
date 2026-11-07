package shape

import "github.com/bencode/sumi/scene"

// diamond is question.ts: a square on its corner, as wide as the label is wide
// and high together, set half a pixel right of the node's center.
func diamond(lw, lh, padding float64) Node {
	s := lw + padding + (lh + padding)
	n := polygon(diamondPoints(s), scene.Point{X: -s/2 + 0.5, Y: s / 2})
	// calcIntersect sets the square anew from the node's laid out width.
	n.Intersect = func(c scene.Point, w, h float64, toward scene.Point) scene.Point {
		return meetPolygon(diamondPoints(w))(c, w, h, toward)
	}
	return n
}

func diamondPoints(s float64) []scene.Point {
	return []scene.Point{{X: s / 2, Y: 0}, {X: s, Y: -s / 2}, {X: s / 2, Y: -s}, {X: 0, Y: -s / 2}}
}

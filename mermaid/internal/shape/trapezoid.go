package shape

import "github.com/bencode/sumi/scene"

// The four slanted boxes: trapezoid.ts, invertedTrapezoid.ts, leanRight.ts and
// leanLeft.ts. Each slants by h/2 at the side it widens.

func trapezoid(lw, lh, padding float64) Node {
	h, w := lh+padding, lw+padding*2
	return slanted(w, h, []scene.Point{{X: -3 * h / 6, Y: 0}, {X: w + 3*h/6, Y: 0}, {X: w, Y: -h}, {X: 0, Y: -h}})
}

func invTrapezoid(lw, lh, padding float64) Node {
	h, w := lh+padding*2, lw+padding*2*2
	return slanted(w, h, []scene.Point{{X: 0, Y: 0}, {X: w, Y: 0}, {X: w + 3*h/6, Y: -h}, {X: -3 * h / 6, Y: -h}})
}

func leanRight(lw, lh, padding float64) Node {
	h, w := lh+padding, lw+padding*2
	return slanted(w, h, []scene.Point{{X: -3 * h / 6, Y: 0}, {X: w, Y: 0}, {X: w + 3*h/6, Y: -h}, {X: 0, Y: -h}})
}

func leanLeft(lw, lh, padding float64) Node {
	h, w := lh+padding, lw+padding*2
	return slanted(w, h, []scene.Point{{X: 0, Y: 0}, {X: w + 3*h/6, Y: 0}, {X: w, Y: -h}, {X: -3 * h / 6, Y: -h}})
}

// slanted places pts as insertPolygonShape does, by the box w by h they slant from.
func slanted(w, h float64, pts []scene.Point) Node {
	return polygon(pts, scene.Point{X: -w / 2, Y: h / 2})
}

package shape

import "github.com/bencode/sumi/scene"

// The four slanted boxes: trapezoid.ts, invertedTrapezoid.ts, leanRight.ts and
// leanLeft.ts. Each slants by h/2 at the side it widens.

func trapezoid(lw, lh, padding float64) Node {
	h, w := lh+padding, lw+padding*2
	return inserted(w, h, []scene.Point{{X: -3 * h / 6, Y: 0}, {X: w + 3*h/6, Y: 0}, {X: w, Y: -h}, {X: 0, Y: -h}})
}

func invTrapezoid(lw, lh, padding float64) Node {
	h, w := lh+padding*2, lw+padding*2*2
	return inserted(w, h, []scene.Point{{X: 0, Y: 0}, {X: w, Y: 0}, {X: w + 3*h/6, Y: -h}, {X: -3 * h / 6, Y: -h}})
}

func leanRight(lw, lh, padding float64) Node {
	h, w := lh+padding, lw+padding*2
	return inserted(w, h, []scene.Point{{X: -3 * h / 6, Y: 0}, {X: w, Y: 0}, {X: w + 3*h/6, Y: -h}, {X: 0, Y: -h}})
}

func leanLeft(lw, lh, padding float64) Node {
	h, w := lh+padding, lw+padding*2
	return inserted(w, h, []scene.Point{{X: 0, Y: 0}, {X: w + 3*h/6, Y: 0}, {X: w, Y: -h}, {X: -3 * h / 6, Y: -h}})
}

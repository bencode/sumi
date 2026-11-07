package shape

import "github.com/bencode/sumi/scene"

// hexagon is hexagon.ts: a box with points to the left and right, each h/3.5
// deep in the neo look.
func hexagon(lw, lh, _ float64) Node {
	const f, padH, padW = 3.5, 70, 32 // Mermaid's labelPaddingX goes to the height
	h := lh + padH
	m := h / f
	w := lw + 2*m + padW
	return polygon([]scene.Point{
		{X: m, Y: 0}, {X: w - m, Y: 0}, {X: w, Y: -h / 2}, {X: w - m, Y: -h}, {X: m, Y: -h}, {X: 0, Y: -h / 2},
	}, scene.Point{X: -w / 2, Y: h / 2})
}

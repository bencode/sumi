package shape

import "github.com/bencode/sumi/scene"

// subroutine is subroutine.ts: a box with a frame 8 wide down each side, drawn
// as one polygon that walks the inner box and then the outer.
func subroutine(lw, lh, _ float64) Node {
	const frame, padX, padY = 8, 28, 12
	w := lw + 2*frame + padX - 2*frame
	h := lh + padY
	return polygon([]scene.Point{
		{X: 0, Y: 0}, {X: w, Y: 0}, {X: w, Y: -h}, {X: 0, Y: -h}, {X: 0, Y: 0},
		{X: -frame, Y: 0}, {X: w + frame, Y: 0}, {X: w + frame, Y: -h}, {X: -frame, Y: -h}, {X: -frame, Y: 0},
	}, scene.Point{X: -w / 2, Y: h / 2})
}

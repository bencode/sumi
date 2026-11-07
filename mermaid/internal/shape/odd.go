package shape

import "github.com/bencode/sumi/scene"

// odd is rect_left_inv_arrow.ts, the flowchart's >odd] node: a flag with a
// notch h/4 deep in its left side. The outline and the label move right by
// half the notch, which centers the outline on the node.
func odd(lw, lh, _ float64) Node {
	const padX, padY = 21, 12
	w := lw + padX*2
	h := lh + padY*2
	x, y := -w/2, -h/2
	notch := y / 2
	n := polygon([]scene.Point{
		{X: x + notch, Y: y}, {X: x, Y: 0}, {X: x + notch, Y: -y}, {X: -x, Y: -y}, {X: -x, Y: y},
	}, scene.Point{X: -notch / 2})
	n.Label = scene.Point{X: -notch / 2}
	return n
}

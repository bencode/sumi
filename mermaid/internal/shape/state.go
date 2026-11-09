package shape

import "github.com/bencode/sumi/scene"

// The shapes only a state diagram draws. What layout gets is the getBBox
// insertMeasuredNode takes of the whole node, whatever size the shape set itself.

const (
	startRadius  = 7
	endInner     = 2.5
	barLength    = 70
	barWidth     = 10
	choiceRadius = 14

	// endWidth is the width of stateEnd's circle: roughjs's curve through its points
	// goes that little past the circle on the right, and not at all below it. It is
	// always 14 across when it is measured, so this is what Chrome gave.
	endWidth = 14.013293266296387
)

// stateStart is stateStart.ts: a circle.
func stateStart(_, _, _ float64) Node {
	return Node{
		W:         2 * startRadius,
		H:         2 * startRadius,
		Outline:   scene.Ellipse(scene.Point{}, startRadius, startRadius),
		Intersect: meetEllipse(startRadius, startRadius),
	}
}

// stateEnd is stateEnd.ts: a circle around a smaller one.
func stateEnd(_, _, _ float64) Node {
	return Node{
		W:         endWidth,
		H:         2 * startRadius,
		Outline:   append(scene.Ellipse(scene.Point{}, startRadius, startRadius), scene.Ellipse(scene.Point{}, endInner, endInner)...),
		Intersect: meetEllipse(endWidth/2, endWidth/2),
	}
}

// fork is forkJoin.ts for a flow top to bottom or bottom to top: a bar across it.
func fork(_, _, _ float64) Node { return rectNode(barLength, barWidth, 0) }

// forkSide is forkJoin.ts for a flow left to right or right to left.
func forkSide(_, _, _ float64) Node { return rectNode(barWidth, barLength, 0) }

// choice is choice.ts: a square on its corner.
func choice(_, _, _ float64) Node {
	return polygon([]scene.Point{{X: 0, Y: choiceRadius}, {X: choiceRadius, Y: 0}, {X: 0, Y: -choiceRadius}, {X: -choiceRadius, Y: 0}}, scene.Point{})
}

// note is note.ts: padding all round the label.
func note(w, h, padding float64) Node {
	return rectNode(w+padding*2, h+padding*2, 0)
}

// Titled is rectWithTitle.ts's node: two labels, a title over a description, with a
// line between them.
type Titled struct {
	Node
	Title, Description scene.Point // the boxes' centers, from the node's center
	Line               [2]scene.Point
}

// titleGap is what rectWithTitle.ts adds, beyond half the padding, between the title
// and the description, and titleRaise how far it moves the two down from the middle.
const (
	titleGap   = 5
	titleRaise = 3
)

// RectWithTitle sets a title box tw by th and a description box dw by dh.
func RectWithTitle(tw, th, dw, dh, padding float64) Titled {
	half := padding / 2
	bw, bh := max(tw, dw), th+half+titleGap+dh
	top := -bh/2 - half + titleRaise
	lineY := -bh/2 + th
	return Titled{
		Node:        rectNode(bw+padding, bh+padding, 0),
		Title:       scene.Point{Y: top + th/2},
		Description: scene.Point{Y: top + th + half + titleGap + dh/2},
		Line:        [2]scene.Point{{X: -bw/2 - half, Y: lineY}, {X: bw/2 + half, Y: lineY}},
	}
}

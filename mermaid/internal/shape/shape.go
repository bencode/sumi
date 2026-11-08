// Package shape sets a label box in the shapes Mermaid draws a node in, as its
// neo look draws them (rendering-elements/shapes/*.ts in Mermaid): the node's
// size, its outline, where its label goes, and where a line toward the node
// meets its outline.
//
// Mermaid takes most shapes' sizes from the getBBox of the element it draws, so
// such a size is a float32 Chrome worked out from the strings JS wrote: Set
// follows those steps, so that layout gets the very size Mermaid's does.
package shape

import "github.com/bencode/sumi/scene"

// A Kind is a shape a node is drawn in. The comments name Mermaid's functions.
type Kind uint8

const (
	Rect         Kind = iota // squareRect
	Rounded                  // roundedRect
	Stadium                  // stadium
	Subroutine               // subroutine
	Cylinder                 // cylinder
	Circle                   // circle
	DoubleCircle             // doublecircle
	Diamond                  // question
	Hexagon                  // hexagon
	Odd                      // rect_left_inv_arrow
	Trapezoid                // trapezoid
	InvTrapezoid             // inv_trapezoid
	LeanRight                // lean_right
	LeanLeft                 // lean_left
	StateStart               // stateStart
	StateEnd                 // stateEnd
	Fork                     // forkJoin, the flow up or down
	ForkSide                 // forkJoin, the flow left or right
	Choice                   // choice
	Note                     // note
)

// A Node is a label box set in a shape.
type Node struct {
	W, H    float64     // the size, as Chrome's getBBox gives it
	Outline []scene.Seg // about the node's center
	Label   scene.Point // the label box's center, from the node's center

	// Intersect is where a line from the node's center toward a point meets the
	// outline, for the node laid out at center with size w by h.
	Intersect func(center scene.Point, w, h float64, toward scene.Point) scene.Point
}

// Set sets a label box w by h in a shape of kind k; padding is the diagram's
// node padding (flowchart.padding).
func Set(k Kind, w, h, padding float64) Node {
	return setters[k](w, h, padding)
}

var setters = [...]func(w, h, padding float64) Node{
	Rect:         rect,
	Rounded:      rounded,
	Stadium:      stadium,
	Subroutine:   subroutine,
	Cylinder:     cylinder,
	Circle:       circle,
	DoubleCircle: doubleCircle,
	Diamond:      diamond,
	Hexagon:      hexagon,
	Odd:          odd,
	Trapezoid:    trapezoid,
	InvTrapezoid: invTrapezoid,
	LeanRight:    leanRight,
	LeanLeft:     leanLeft,
	StateStart:   stateStart,
	StateEnd:     stateEnd,
	Fork:         fork,
	ForkSide:     forkSide,
	Choice:       choice,
	Note:         note,
}

// WidensLabel reports whether a label in k is widened to the diagram's least
// width. Circles are not, as they size from both sides of the label
// (labelHelper's ignoreMinWidth). An empty label is not widened in any shape,
// which is the caller's to check.
func (k Kind) WidensLabel() bool {
	return k != Circle && k != DoubleCircle
}

// Package scene is a diagram laid out: a list of items, each made of paths
// and lines of text at their final places, drawn in order. Coordinates are in
// DIPs (one CSS px), with the origin at the top left and y pointing down.
//
// A scene says what to draw, not in which colors: a paint names a token of the
// theme, or a color the diagram's author set, and a backend settles tokens
// with a Palette when it draws. So a scene is drawn in any theme without being
// laid out again.
package scene

// A Point is a position, or a vector.
type Point struct{ X, Y float64 }

// A Rect is a box: its top-left corner and its size.
type Rect struct{ X, Y, W, H float64 }

// A Scene is a laid-out diagram. Its content starts at (0, 0).
type Scene struct {
	Width, Height float64
	Items         []Item // in the order they are drawn
}

// An Item is a thing of a diagram. A backend draws its Paths and then its
// Texts, so its text lies on its shape. The other fields say what the item is,
// for tests and for SVG; drawing ignores them.
type Item struct {
	Kind     Kind
	ID       string // the id the diagram's source gave it, if any
	Parent   string // the id of the group it is in; empty at the top
	From, To string // for an edge: the ids of its ends
	Box      Rect   // where the item lies
	Paths    []Path
	Texts    []Text
}

// A Kind is the sort of thing an item is.
type Kind uint8

const (
	Node  Kind = iota + 1 // a shape with a label: a flowchart's node, an actor
	Edge                  // a line from one thing to another, with its label
	Group                 // a box that holds others: a subgraph, a loop
	Label                 // text on its own: a title
	Mark                  // a shape with no label: a lifeline, an activation
)

var kindNames = [...]string{Node: "node", Edge: "edge", Group: "group", Label: "label", Mark: "mark"}

// String names the kind in lower case, as "node".
func (k Kind) String() string {
	if int(k) < len(kindNames) && kindNames[k] != "" {
		return kindNames[k]
	}
	return "unknown"
}

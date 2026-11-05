// SPDX-License-Identifier: EPL-2.0 OR GPL-3.0-or-later

package elk

// Direction is the way edges go (core Direction.java).
type Direction int

// The directions Mermaid asks for.
const (
	Down Direction = iota
	Right
	Up
	Left
)

// Size is a width and a height.
type Size struct{ Width, Height float64 }

// InputNode is a node to place.
type InputNode struct {
	ID            string
	Width, Height float64
}

// InputEdge is an edge to route. Label is nil when the edge has no text: ELK
// drops a label with empty text (ElkGraphImporter.java line 1203).
type InputEdge struct {
	ID, Source, Target string
	Label              *Size
}

// Input is what Mermaid gives ELK. The order of Nodes and Edges counts:
// layout prefers it when choices are otherwise equal.
type Input struct {
	Direction Direction
	Nodes     []InputNode
	Edges     []InputEdge
}

// PlacedNode is a node with its top left corner.
type PlacedNode struct {
	ID                  string
	X, Y, Width, Height float64
}

// PlacedLabel is a label with its top left corner.
type PlacedLabel struct{ X, Y, Width, Height float64 }

// PlacedEdge is an edge with its path, from start through bends to end.
type PlacedEdge struct {
	ID     string
	Points []Vec
	Labels []PlacedLabel
}

// Result is a finished layout, in the order of the Input.
type Result struct {
	Width, Height float64
	Nodes         []PlacedNode
	Edges         []PlacedEdge
}

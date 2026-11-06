// SPDX-License-Identifier: EPL-2.0 OR GPL-3.0-or-later

package elk

// Direction is the way edges go (core Direction.java).
type Direction int

// The directions Mermaid asks for. A graph with none goes right, the way ELK
// takes for a wide aspect ratio (LGraphUtil.getDirection, graph/LGraphUtil.java
// line 1134).
const (
	Undefined Direction = iota
	Down
	Right
	Up
	Left
)

// Size is a width and a height.
type Size struct{ Width, Height float64 }

// Hierarchy is hierarchyHandling: how a subgraph's layout relates to its parent's.
type Hierarchy int

const (
	Inherit  Hierarchy = iota // as its parent
	Include                   // in one layout with its parent
	Separate                  // laid out first, then a node of fixed size
)

// InputNode is a node to place, or a subgraph when it has Children. Direction
// counts only where a layout starts, as the nodes laid out with a parent go its
// way. Title is nil when the subgraph has no text. MinWidth is nodeSize.minimum's
// x; Mermaid leaves its y at 0.
type InputNode struct {
	ID            string
	Width, Height float64
	Children      []InputNode
	Hierarchy     Hierarchy
	Direction     Direction
	Title         *Size
	MinWidth      float64
}

// InputEdge is an edge to route. Label is nil when the edge has no text: ELK
// drops a label with empty text (ElkGraphImporter.java line 1203).
type InputEdge struct {
	ID, Source, Target string
	Label              *Size
}

// Input is what Mermaid gives ELK, its edges all here whatever nodes they join.
// The order of Nodes and Edges counts: layout prefers it when choices are
// otherwise equal.
type Input struct {
	Direction Direction
	Nodes     []InputNode
	Edges     []InputEdge
}

// PlacedNode is a node with its top left corner, relative to its parent. A
// Title is relative to the node.
type PlacedNode struct {
	ID                  string
	X, Y, Width, Height float64
	Title               *PlacedLabel
	Children            []PlacedNode
}

// PlacedLabel is a label with its top left corner.
type PlacedLabel struct{ X, Y, Width, Height float64 }

// PlacedEdge is an edge with its path, from start through bends to end.
// Container is the subgraph its coordinates are relative to, empty for the root.
type PlacedEdge struct {
	ID        string
	Container string
	Points    []Vec
	Labels    []PlacedLabel
}

// Result is a finished layout, in the order of the Input.
type Result struct {
	Width, Height float64
	Nodes         []PlacedNode
	Edges         []PlacedEdge
}

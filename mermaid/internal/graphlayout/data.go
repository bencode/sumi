package graphlayout

import "github.com/bencode/sumi/mermaid/internal/elk"

// Data is what a diagram hands to layout (LayoutData). Parents must form a tree:
// a diagram has to turn down subgraphs nested in each other, which Mermaid accepts
// and, with an edge on them, never returns from.
type Data struct {
	Direction elk.Direction // Undefined reads as Down
	Nodes     []Node        // in the order of the diagram
	Edges     []Edge
}

// Box is a node's box by its center.
type Box struct{ X, Y, Width, Height float64 }

// Intersect gives where the line from b's center to point leaves a shape's outline.
type Intersect func(b Box, point elk.Vec) elk.Vec

// Node is a node, or a subgraph when Group is set. A subgraph leaves its size 0.
type Node struct {
	ID, Parent    string // Parent is empty at the root
	Group         bool
	Width, Height float64
	Intersect     Intersect     // nil reads as a rectangle
	Direction     elk.Direction // a subgraph's own; Undefined when it has none
	Title         *elk.Size     // a subgraph's measured title; nil without text
	Untitled      bool          // a Title no one paints (clusterPaintsTitle), which asks for no size
	Padding       float64       // a subgraph's padding
}

// Edge is an edge. Label is its measured text, nil without text. The markers are
// Mermaid's arrowTypeStart and arrowTypeEnd.
type Edge struct {
	ID, Start, End         string
	Label                  *elk.Size
	StartMarker, EndMarker string
}

// Result is a finished layout, its nodes in the order to paint them and its edges
// in the order of the Data.
type Result struct {
	Nodes []PlacedNode
	Edges []PlacedEdge
}

// PlacedNode is a node's place.
type PlacedNode struct {
	ID string
	Box
}

// PlacedEdge is an edge's path, from start through bends to end. Label is the
// center of its text, nil without text.
type PlacedEdge struct {
	ID     string
	Points []elk.Vec
	Label  *elk.Vec
}

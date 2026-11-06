package graphlayout

import "github.com/bencode/sumi/mermaid/internal/elk"

// Data is what a diagram hands to layout (LayoutData).
type Data struct {
	Direction elk.Direction // Undefined reads as Down
	Nodes     []Node        // in the order of the diagram
	Edges     []Edge
}

// Node is a node, or a subgraph when others name it as their Parent. A subgraph
// leaves its size 0.
type Node struct {
	ID, Parent    string // Parent is empty at the root
	Width, Height float64
	Direction     elk.Direction // a subgraph's own; Undefined when it has none
	Title         *elk.Size     // a subgraph's measured title; nil without text
	Padding       float64       // a subgraph's padding
}

// Edge is an edge. Label is its measured text, nil without text.
type Edge struct {
	ID, Start, End string
	Label          *elk.Size
}

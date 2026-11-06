// Package graphlayout places what flowcharts, state, class and ER diagrams draw,
// as Mermaid v12.3.0 places it with its default configuration. It is written after
// Mermaid's layout-elk, and file names in comments are those of layout-elk.
//
// Mermaid does more than call ELK. Before ELK it builds a tree of what diagrams
// give as a flat list, decides which subgraphs ELK lays out on their own, and
// turns round or moves edges that ELK would otherwise route badly (input.go,
// feedback.go, nesting.go). After ELK it mends what ELK gives, in this order, which
// counts:
//
//	placeNodes       ELK's relative places to absolute ones      frames.go
//	evenFrames       subgraph frames in around what they hold    frames.go
//	alignSmallNodes  nodes too small for ports onto their lines  frames.go
//	routeEdges       paths clipped to the outlines of nodes      route.go, clip.go
//	straightenEdges  the step beside a port out of a path        straighten.go
//	separateLabels   texts of opposite edges apart               labels.go
//
// Nothing calls this package yet.
package graphlayout

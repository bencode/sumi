package dagre

// Options are what Mermaid sets on dagre's graph. A field left at 0 is 0,
// not dagre's default.
type Options struct {
	RankDir          string // "TB", "BT", "LR" or "RL"; "" is TB
	NodeSep, RankSep float64
	MarginX, MarginY float64
}

type Point struct{ X, Y float64 }

// A PlacedNode is a node's center and size. A subgraph's are those of the
// box around what it holds.
type PlacedNode struct {
	ID                  string
	X, Y, Width, Height float64
}

// A PlacedEdge is an edge's path, from V to W, and the center of its label
// when HasLabel.
type PlacedEdge struct {
	V, W, Name string
	Points     []Point
	X, Y       float64
	HasLabel   bool
}

// A Result is where dagre puts in's nodes and edges, in in's order, and
// the size it gives the graph. Like dagre's, the size leaves out edges'
// bends, which can go outside it, and is -Inf for an empty graph.
type Result struct {
	Nodes         []PlacedNode
	Edges         []PlacedEdge
	Width, Height float64
}

// Layout lays in out as dagre does (layout.js line 16).
func Layout(in Graph, opts Options) Result { return layout(in, opts, &counter{}) }

// layout runs dagre's steps (layout.js lines 26-52) and reads off the
// result, as updateInputGraph does (line 61). c names what it adds.
func layout(in Graph, opts Options, c *counter) Result {
	g, chains := orderGraph(in, c)
	adjustCoordinates(g, opts.RankDir)
	position(g, opts.NodeSep, opts.RankSep/2) // makeSpaceForEdgeLabels
	removeBorderNodes(g)
	denormalize(g, chains)
	undoCoordinates(g, opts.RankDir)
	width, height := translateGraph(g, opts.MarginX, opts.MarginY)
	assignNodeIntersects(g)
	reversePoints(g)
	undoAcyclic(g)

	r := Result{Width: width, Height: height}
	for _, n := range in.Nodes {
		l := g.node(n.ID)
		r.Nodes = append(r.Nodes, PlacedNode{n.ID, l.x, l.y, l.width, l.height})
	}
	for _, e := range in.Edges {
		l := g.edge(e.V, e.W, e.Name)
		r.Edges = append(r.Edges, PlacedEdge{e.V, e.W, e.Name, l.points, l.x, l.y, l.hasXY})
	}
	return r
}

// SPDX-License-Identifier: EPL-2.0 OR GPL-3.0-or-later

package elk

import "slices"

// Vec is a point or a size (KVector).
type Vec struct{ X, Y float64 }

func (v Vec) add(w Vec) Vec { return Vec{v.X + w.X, v.Y + w.Y} }

// Spacing is the room around a graph's content (LPadding) or a node (LMargin).
type Spacing struct{ Top, Right, Bottom, Left float64 }

// PortSide is the side of its node a port sits on (core PortSide.java).
type PortSide int

const (
	sideUndefined PortSide = iota
	sideNorth
	sideEast
	sideSouth
	sideWest
)

// sideOf gives the side edges leave a node by, going in a direction
// (fromDirection, core PortSide.java line 181).
var sideOf = map[Direction]PortSide{Up: sideNorth, Right: sideEast, Down: sideSouth, Left: sideWest}

// opposed gives the side across from a side (core PortSide.java line 146).
var opposed = map[PortSide]PortSide{
	sideNorth: sideSouth, sideSouth: sideNorth, sideEast: sideWest, sideWest: sideEast,
}

// NodeType tells a real node from the ones steps add (LNode.NodeType).
type NodeType int

const (
	typeNormal NodeType = iota
	typeLongEdge
	typeLabel
)

// Graph is a graph being laid out (graph/LGraph.java). Nodes are in Layerless
// until they are given a layer, and back there when layout ends.
type Graph struct {
	Size, Offset Vec
	MinSize      Vec // nodeSize.minimum, never turned with the graph
	Padding      Spacing
	Layerless    []*Node
	Layers       []*Layer
	Direction    Direction
	Settings     settings
	Hierarchical bool    // hierarchyHandling is INCLUDE_CHILDREN
	Parent       *Node   // the node holding this graph, nil on the root
	SelfLoops    bool    // GraphProperties.SELF_LOOPS
	CenterLabels bool    // GraphProperties.CENTER_LABELS
	Random       *random // InternalProperties.RANDOM
}

// Layer is one layer's nodes, in order (graph/Layer.java).
type Layer struct {
	Nodes []*Node
	Width float64
}

// Node is a node (graph/LNode.java). Pos is its top left corner. A subgraph has
// its graph in Nested, Included when that takes part in the same layout, which
// is what ELK's LNode.getNestedGraph means. ELK has no LGraph at all for one
// laid out on its own, and keeps it on the ElkNode.
type Node struct {
	ID         string
	Type       NodeType
	Pos, Size  Vec
	Margin     Spacing
	Ports      []*Port
	Loops      []selfLoop // InternalProperties.SELF_LOOP_HOLDER
	ModelOrder int        // its index among its siblings
	Nested     *Graph
	Included   bool
	Title      *Label
}

// Port is where an edge meets a node (graph/LPort.java). Pos is relative to
// the node.
type Port struct {
	Owner   *Node
	Side    PortSide
	Pos     Vec
	In, Out []*Edge
}

// Edge is an edge (graph/LEdge.java). Bends hold the points between its ends.
type Edge struct {
	ID             string
	Source, Target *Port
	Bends          []Vec
	Labels         []*Label
	ModelOrder     int  // its index in Input.Edges
	Reversed       bool // InternalProperties.REVERSED
}

// Label is an edge's label or a subgraph's title (graph/LLabel.java).
type Label struct{ Pos, Size Vec }

// actualSize is g's size with its padding (graph/LGraph.java line 98).
func actualSize(g *Graph) Vec {
	return Vec{g.Size.X + g.Padding.Left + g.Padding.Right, g.Size.Y + g.Padding.Top + g.Padding.Bottom}
}

// nodes gives g's nodes, those without a layer first
// (intermediate/GraphTransformer.java lines 77-80).
func nodes(g *Graph) []*Node {
	all := slices.Clone(g.Layerless)
	for _, l := range g.Layers {
		all = append(all, l.Nodes...)
	}
	return all
}

// incoming gives the edges coming to n, port by port (graph/LNode.java line 348).
func incoming(n *Node) []*Edge {
	var all []*Edge
	for _, p := range n.Ports {
		all = append(all, p.In...)
	}
	return all
}

// outgoing gives the edges leaving n, port by port (graph/LNode.java line 362).
func outgoing(n *Node) []*Edge {
	var all []*Edge
	for _, p := range n.Ports {
		all = append(all, p.Out...)
	}
	return all
}

// setSource moves e's start to p, last among p's edges; a nil p leaves e
// without a start (graph/LEdge.java line 112).
func setSource(e *Edge, p *Port) {
	if e.Source != nil {
		e.Source.Out = without(e.Source.Out, e)
	}
	e.Source = p
	if p != nil {
		p.Out = append(p.Out, e)
	}
}

// setTarget moves e's end to p, as setSource moves its start
// (graph/LEdge.java line 142).
func setTarget(e *Edge, p *Port) {
	if e.Target != nil {
		e.Target.In = without(e.Target.In, e)
	}
	e.Target = p
	if p != nil {
		p.In = append(p.In, e)
	}
}

// without gives all with item taken out, keeping the order.
func without[T comparable](all []T, item T) []T {
	i := slices.Index(all, item)
	return slices.Delete(all, i, i+1)
}

// edges gives g's edges, each once, from the ports they leave.
func edges(g *Graph) []*Edge {
	var all []*Edge
	for _, n := range nodes(g) {
		all = append(all, outgoing(n)...)
	}
	return all
}

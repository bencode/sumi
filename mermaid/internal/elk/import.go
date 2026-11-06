// SPDX-License-Identifier: EPL-2.0 OR GPL-3.0-or-later

package elk

import "fmt"

// titleGap is nodeLabels.padding's default, the room around a subgraph's title
// on every side (core Core.melk line 524).
const titleGap = 5.0

// located is a node and the graph it sits in.
type located struct {
	node  *Node
	graph *Graph
}

// importGraph turns in into a Graph, all nodes first, as importHierarchicalGraph
// does (graph/transform/ElkGraphImporter.java lines 96, 309). The tree comes
// whole, where ELK imports each layout when it starts: a graph's layout reads
// nothing of another's, so the two are the same.
func importGraph(in Input) *Graph {
	g := newGraph(rootSettings(), layoutDirection(in.Direction))
	g.Hierarchical = true
	byID := map[string]located{}
	importNodes(g, in.Nodes, Include, byID)
	importEdges(g, in.Edges, byID)
	return g
}

// newGraph is the part of createLGraph that every graph shares (ElkGraphImporter
// line 592), with its random numbers from ELK's default seed
// (GraphConfigurator.java line 121).
func newGraph(s settings, d Direction) *Graph {
	p := s.padding
	return &Graph{Settings: s, Direction: d, Padding: Spacing{p, p, p, p}, Random: newRandom(1)}
}

// layoutDirection gives a layout's direction (createLGraph, ElkGraphImporter line
// 592, and LGraphUtil.getDirection): Mermaid leaves the aspect ratio wide.
func layoutDirection(d Direction) Direction {
	if d == Undefined {
		return Right
	}
	return d
}

// importNodes adds all to g, whose hierarchy is the one given.
func importNodes(g *Graph, all []InputNode, hierarchy Hierarchy, byID map[string]located) {
	for i, n := range all {
		node := &Node{ID: n.ID, Size: Vec{n.Width, n.Height}, ModelOrder: i, Graph: g}
		if n.Title != nil {
			node.Title = &Label{Size: Vec{n.Title.Width, n.Title.Height}}
		}
		g.Layerless = append(g.Layerless, node)
		byID[n.ID] = located{node, g}
		if len(n.Children) > 0 {
			importSubgraph(g, node, n, hierarchy, byID)
		}
	}
}

// importSubgraph gives node, in g, the graph of n's children. A node inherits its
// parent's hierarchy (evaluateHierarchyHandlingInheritance,
// RecursiveGraphLayoutEngine.java line 512). Its graph joins the parent's layout
// when both include (layoutRecursively; importHierarchicalGraph, ElkGraphImporter
// line 309), and then goes the parent's way.
func importSubgraph(g *Graph, node *Node, n InputNode, hierarchy Hierarchy, byID map[string]located) {
	own := n.Hierarchy
	if own == Inherit {
		own = hierarchy
	}
	node.Included = own == Include && hierarchy == Include
	direction := g.Direction
	if !node.Included {
		direction = layoutDirection(n.Direction)
	}
	nested := newGraph(subgraphSettings(), direction)
	nested.Hierarchical = own == Include
	nested.Parent = node
	nested.MinSize = minimumSize(n)
	if n.Title != nil {
		nested.Padding.Top += n.Title.Height + titleGap
	}
	node.Nested = nested
	importNodes(nested, n.Children, own, byID)
}

// minimumSize gives the size a subgraph keeps at least: what its title takes with
// the room around it, and what MinWidth asks (calculateMinimumGraphSize,
// ElkGraphImporter line 193; getMinimumNodeOrClientAreaSize,
// nodespacing/internal/algorithm/NodeLabelAndSizeUtilities.java line 224). The
// cell grid of a node's labels is symmetrical, so a title's top row counts twice
// (minRowHeights, nodespacing/cellsystem/GridContainerCell.java line 344). ELK's
// default of 20 for a side left unset never tells: padding alone comes to more.
func minimumSize(n InputNode) Vec {
	if n.Title == nil {
		return Vec{n.MinWidth, 0}
	}
	return Vec{max(n.MinWidth, n.Title.Width+2*titleGap), n.Title.Height + n.Title.Height + 2*titleGap}
}

// importEdges adds all edges to the graphs holding them, breadth first and in
// input order within a graph, which sets the order of a node's ports
// (importHierarchicalGraph, graph/transform/ElkGraphImporter.java line 309).
func importEdges(root *Graph, all []InputEdge, byID map[string]located) {
	held := map[*Graph][]int{}
	for i, e := range all {
		source, sourceKnown := byID[e.Source]
		target, targetKnown := byID[e.Target]
		if !sourceKnown || !targetKnown {
			panic(fmt.Sprintf("elk: edge %s has an unknown end", e.ID))
		}
		if layoutRoot(source.graph) != layoutRoot(target.graph) {
			panic(fmt.Sprintf("elk: edge %s crosses the border of a subgraph laid out on its own", e.ID))
		}
		g := edgeContainer(source, target)
		held[g] = append(held[g], i)
	}
	for queue := []*Graph{root}; len(queue) > 0; queue = queue[1:] {
		g := queue[0]
		for _, i := range held[g] {
			importEdge(g, byID[all[i].Source].node, byID[all[i].Target].node, all[i], i)
		}
		for _, n := range g.Layerless {
			if n.Nested != nil {
				queue = append(queue, n.Nested)
			}
		}
	}
}

// layoutRoot gives the graph at the top of g's layout.
func layoutRoot(g *Graph) *Graph {
	for included(g) {
		g = g.Parent.Graph
	}
	return g
}

// edgeContainer gives the graph holding an edge between two nodes: the lowest
// that holds both, or the inside of one that holds the other
// (findBestEdgeContainment, org.eclipse.elk.graph util/ElkGraphUtil.java line 334,
// which the JSON importer applies to every edge).
func edgeContainer(source, target located) *Graph {
	switch {
	case isDescendant(target.node, source.node):
		return source.node.Nested
	case isDescendant(source.node, target.node):
		return target.node.Nested
	}
	for g := source.graph; ; g = g.Parent.Graph {
		for h := target.graph; ; h = h.Parent.Graph {
			if g == h {
				return g
			}
			if h.Parent == nil {
				break
			}
		}
	}
}

// importEdge adds e with a new port at each end, the source's first (transformEdge,
// ElkGraphImporter.java line 1084). The edge may cross borders: splitCrossEdges
// takes care of that.
func importEdge(g *Graph, source, target *Node, e InputEdge, order int) {
	side := sideOf[g.Direction]
	edge := &Edge{ID: e.ID, ModelOrder: order, Container: g}
	edge.Source, edge.Target = newPort(source, side), newPort(target, opposed[side])
	edge.Source.Out = append(edge.Source.Out, edge)
	edge.Target.In = append(edge.Target.In, edge)
	if e.Label != nil {
		edge.Labels = []*Label{{Size: Vec{e.Label.Width, e.Label.Height}}}
	}
}

// newPort adds a port on one side of n. Ports are never shared, since Mermaid
// sets mergeEdges to false (createPort, graph/LGraphUtil.java line 433).
func newPort(n *Node, side PortSide) *Port {
	p := &Port{Owner: n, Side: side}
	n.Ports = append(n.Ports, p)
	return p
}

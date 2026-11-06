// SPDX-License-Identifier: EPL-2.0 OR GPL-3.0-or-later

package elk

import "slices"

// export reads the finished layout out of g, whose nodes are all back in
// Layerless (applyLayout, graph/transform/ElkGraphLayoutTransferrer.java line
// 67). ELK writes onto the input graph's objects; here ModelOrder gives each
// its place in the Result.
func export(g *Graph) Result {
	size := actualSize(g)
	byOrder := map[int]PlacedEdge{}
	r := Result{Width: size.X, Height: size.Y, Nodes: exportNodes(g, byOrder)}
	r.Edges = make([]PlacedEdge, len(byOrder))
	for i, e := range byOrder {
		r.Edges[i] = e
	}
	return r
}

// exportNodes gives the nodes of g, and adds its edges and those of the graphs
// below it to byOrder.
func exportNodes(g *Graph, byOrder map[int]PlacedEdge) []PlacedNode {
	offset := Vec{g.Offset.X + g.Padding.Left, g.Offset.Y + g.Padding.Top}
	for _, e := range edges(g) {
		byOrder[e.ModelOrder] = exportEdge(e, offset, containerID(g))
	}
	for _, c := range g.Crossing {
		byOrder[c.edge.ModelOrder] = c.placed
	}
	own := slices.DeleteFunc(slices.Clone(g.Layerless), func(n *Node) bool { return n.Type != typeNormal })
	placed := make([]PlacedNode, len(own))
	for _, n := range own {
		placed[n.ModelOrder] = exportNode(n, offset, byOrder)
	}
	return placed
}

// containerID names the node holding g, the root having none.
func containerID(g *Graph) string {
	if g.Parent == nil {
		return ""
	}
	return g.Parent.ID
}

// exportNode is applyNodeLayout (ElkGraphLayoutTransferrer.java line 159).
func exportNode(n *Node, offset Vec, byOrder map[int]PlacedEdge) PlacedNode {
	pos := n.Pos.add(offset)
	placed := PlacedNode{ID: n.ID, X: pos.X, Y: pos.Y, Width: n.Size.X, Height: n.Size.Y}
	if n.Title != nil {
		t := n.Title
		placed.Title = &PlacedLabel{t.Pos.X, t.Pos.Y, t.Size.X, t.Size.Y}
	}
	if n.Nested != nil {
		placed.Children = exportNodes(n.Nested, byOrder)
	}
	return placed
}

// exportEdge is applyEdgeLayout (ElkGraphLayoutTransferrer.java line 233).
func exportEdge(e *Edge, offset Vec, container string) PlacedEdge {
	points := make([]Vec, 0, len(e.Bends)+2)
	points = append(points, absolute(e.Source).add(offset))
	for _, b := range e.Bends {
		points = append(points, b.add(offset))
	}
	points = append(points, absolute(e.Target).add(offset))
	labels := make([]PlacedLabel, len(e.Labels))
	for i, l := range e.Labels {
		pos := l.Pos.add(offset)
		labels[i] = PlacedLabel{pos.X, pos.Y, l.Size.X, l.Size.Y}
	}
	return PlacedEdge{ID: e.ID, Container: container, Points: points, Labels: labels}
}

// absolute gives where p sits in g (getAbsoluteAnchor, graph/LPort.java line
// 198).
func absolute(p *Port) Vec { return p.Owner.Pos.add(p.Pos) }

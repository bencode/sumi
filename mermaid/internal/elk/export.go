// SPDX-License-Identifier: EPL-2.0 OR GPL-3.0-or-later

package elk

// export reads the finished layout out of g, whose nodes are all back in
// Layerless (applyLayout, graph/transform/ElkGraphLayoutTransferrer.java line
// 67). ELK writes onto the input graph's objects; here ModelOrder gives each
// its place in the Result.
func export(g *Graph) Result {
	offset := Vec{g.Offset.X + g.Padding.Left, g.Offset.Y + g.Padding.Top}
	all := edges(g)
	r := Result{
		Width:  g.Size.X + g.Padding.Left + g.Padding.Right,
		Height: g.Size.Y + g.Padding.Top + g.Padding.Bottom,
		Nodes:  make([]PlacedNode, len(g.Layerless)),
		Edges:  make([]PlacedEdge, len(all)),
	}
	for _, n := range g.Layerless {
		r.Nodes[n.ModelOrder] = exportNode(n, offset)
	}
	for _, e := range all {
		r.Edges[e.ModelOrder] = exportEdge(e, offset)
	}
	return r
}

// exportNode is applyNodeLayout (ElkGraphLayoutTransferrer.java line 159).
func exportNode(n *Node, offset Vec) PlacedNode {
	pos := n.Pos.add(offset)
	return PlacedNode{ID: n.ID, X: pos.X, Y: pos.Y, Width: n.Size.X, Height: n.Size.Y}
}

// exportEdge is applyEdgeLayout (ElkGraphLayoutTransferrer.java line 233).
func exportEdge(e *Edge, offset Vec) PlacedEdge {
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
	return PlacedEdge{ID: e.ID, Points: points, Labels: labels}
}

// absolute gives where p sits in g (getAbsoluteAnchor, graph/LPort.java line
// 198).
func absolute(p *Port) Vec { return p.Owner.Pos.add(p.Pos) }

// SPDX-License-Identifier: EPL-2.0 OR GPL-3.0-or-later

package elk

import "slices"

// reverseConstrainedEdges gives g's external port dummies their layers and turns
// their edges to fit (process, intermediate/EdgeAndLayerConstraintEdgeReverser.java
// line 71). A dummy on the west goes first and on the east last, which the
// direction's turn has made hold for all directions (GraphTransformer.java lines
// 369 and 795). A dummy has one edge, which leads to no other dummy, so ELK's
// canReverseOutgoingEdge and canReverseIncomingEdge (lines 281 and 322) always
// agree. A subgraph runs the way its parent does, so its ports are never all
// against the flow, and ELK's turn of such a node (line 174) never comes up.
func reverseConstrainedEdges(g *Graph) {
	for _, n := range g.Layerless {
		switch {
		case n.Type != typeExternalPort:
		case n.ExtSide == sideWest:
			n.Constraint = firstSeparate
			reverseAll(incoming(n))
		default:
			n.Constraint = lastSeparate
			reverseAll(outgoing(n))
		}
	}
}

func reverseAll(all []*Edge) {
	for _, e := range all {
		reverse(e)
	}
}

// separateLinks tells which separate layers a node was joined to.
type separateLinks struct{ first, last bool }

// hideSeparateNodes takes the nodes of separate layers off the graph before
// layering, which their edges would stretch. A node left with no edge goes next
// to the layer it was joined to, if there is only one (process and hide,
// intermediate/LayerConstraintPreprocessor.java lines 61 and 104;
// updateOppositeNodeLayerConstraints, line 185).
func hideSeparateNodes(g *Graph) {
	links := map[*Node]separateLinks{}
	hidden := func(n *Node) bool { return n.Constraint == firstSeparate || n.Constraint == lastSeparate }
	for _, n := range g.Layerless {
		if hidden(n) {
			hide(n, links)
			g.Hidden = append(g.Hidden, n)
		}
	}
	g.Layerless = slices.DeleteFunc(g.Layerless, hidden)
}

func hide(n *Node, links map[*Node]separateLinks) {
	for _, e := range slices.Concat(incoming(n), outgoing(n)) {
		if e.Source.Owner == n {
			e.Opposite = e.Target
			setTarget(e, nil)
		} else {
			e.Opposite = e.Source
			setSource(e, nil)
		}
		settle(e.Opposite.Owner, n.Constraint, links)
	}
}

func settle(n *Node, separate layerConstraint, links map[*Node]separateLinks) {
	if n.Constraint != noConstraint {
		return
	}
	l := links[n]
	l.first = l.first || separate == firstSeparate
	l.last = l.last || separate == lastSeparate
	links[n] = l
	if len(incoming(n))+len(outgoing(n)) > 0 {
		return
	}
	switch {
	case l.first && !l.last:
		n.Constraint = firstLayer
	case l.last && !l.first:
		n.Constraint = lastLayer
	}
}

// placeConstrainedNodes moves the nodes hideSeparateNodes tied to the first or last
// layer there and restores the hidden ones in layers of their own (process,
// intermediate/LayerConstraintPostprocessor.java line 63). A move goes to the end of
// a layer, even within it. Such a node has no edges, so no label dummy moves
// with it. No layer empties: such nodes start in the first layer, and that holds
// nothing else only if no node has an edge, which leaves it the one layer.
func placeConstrainedNodes(g *Graph) {
	if len(g.Layers) > 0 {
		first, last := g.Layers[0], g.Layers[len(g.Layers)-1]
		for _, l := range g.Layers {
			for _, n := range slices.Clone(l.Nodes) {
				switch n.Constraint {
				case firstLayer:
					moveToLayer(n, l, first)
				case lastLayer:
					moveToLayer(n, l, last)
				}
			}
		}
	}
	before, after := &Layer{}, &Layer{}
	for _, n := range g.Hidden {
		if n.Constraint == firstSeparate {
			before.Nodes = append(before.Nodes, n)
		} else {
			after.Nodes = append(after.Nodes, n)
		}
		restore(n)
	}
	g.Hidden = nil
	if len(before.Nodes) > 0 {
		g.Layers = slices.Insert(g.Layers, 0, before)
	}
	if len(after.Nodes) > 0 {
		g.Layers = append(g.Layers, after)
	}
}

func moveToLayer(n *Node, from, to *Layer) {
	from.Nodes = without(from.Nodes, n)
	to.Nodes = append(to.Nodes, n)
}

// restore puts n's edges back on the ports they left (restoreHiddenNodes,
// LayerConstraintPostprocessor.java line 181).
func restore(n *Node) {
	for _, p := range n.Ports {
		for _, e := range slices.Clone(p.In) {
			setSource(e, e.Opposite)
		}
		for _, e := range slices.Clone(p.Out) {
			setTarget(e, e.Opposite)
		}
	}
}

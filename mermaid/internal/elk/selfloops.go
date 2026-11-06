// SPDX-License-Identifier: EPL-2.0 OR GPL-3.0-or-later

package elk

import "slices"

// selfLoop is a self loop out of the graph, with the ports it left. Each edge
// has ports of its own, so ELK's hyper loop of edges sharing ports is one edge.
type selfLoop struct {
	edge           *Edge
	source, target *Port
}

// hideSelfLoops takes self loops and their ports off their nodes, to keep them
// out of the way of layering and ordering (process and hidePorts,
// intermediate/SelfLoopPreProcessor.java lines 51, 88).
func hideSelfLoops(g *Graph) {
	for _, n := range g.Layerless {
		for _, e := range outgoing(n) {
			if e.Target.Owner == n {
				n.Loops = append(n.Loops, selfLoop{e, e.Source, e.Target})
				n.Ports = slices.DeleteFunc(n.Ports, func(p *Port) bool { return p == e.Source || p == e.Target })
				setSource(e, nil)
				setTarget(e, nil)
			}
		}
	}
}

// restoreLoopPorts puts ports back on the north side, each later loop around the
// ones before it, sources to the left. Mermaid sets selfLoopDistribution on the
// root, but ELK reads it from nodes, so it stays NORTH; the ordering stays
// STACKED (process, intermediate/SelfLoopPortRestorer.java line 48;
// assignToNorthSide, intermediate/loops/ordering/PortSideAssigner.java line 68;
// processOneSideLoops, intermediate/loops/ordering/PortRestorer.java line 120).
// North ports come first on a node, which has none of its own.
func restoreLoopPorts(g *Graph) {
	for _, n := range nodes(g) {
		var sources, targets []*Port
		for _, l := range n.Loops {
			l.source.Side, l.target.Side = sideNorth, sideNorth
			sources = append(sources, l.source)
			targets = append(targets, l.target)
		}
		slices.Reverse(sources)
		n.Ports = slices.Concat(sources, targets, n.Ports)
	}
}

// routeSelfLoops runs each loop above its node, inner loops nearest, and gives
// the node margins to hold them (routeSelfLoops,
// intermediate/loops/routing/OrthogonalSelfLoopRouter.java line 51). Loops nest,
// so an outer loop can only go above an inner one without crossing it. That
// leaves intermediate/loops/routing/RoutingSlotAssigner.java no cycle to break
// and no slot to share: a node's i-th loop gets slot i, and no random number is
// drawn. Bends and labels are relative to the node until restoreSelfLoops.
func routeSelfLoops(g *Graph) {
	s := g.Settings
	for _, n := range nodes(g) {
		margin := n.Margin
		y := -n.Margin.Top - s.selfLoop
		for _, l := range n.Loops {
			l.edge.Bends = []Vec{{l.source.Pos.X, y}, {l.target.Pos.X, y}}
			for _, b := range l.edge.Bends {
				growMargin(&margin, n.Size, b)
			}
			y -= placeLoopLabel(l.edge, n, y, s.edgeLabel, &margin) + s.edgeEdge
		}
		n.Margin = margin
	}
}

// placeLoopLabel puts e's label above the loop at y, centered on n, and gives
// the room the label takes (placeLabels and computePositions,
// OrthogonalSelfLoopRouter.java lines 139, 261; computeCoordinates,
// intermediate/loops/routing/LabelPlacer.java line 372). A loop on one side
// drops the label's inline option (LabelPlacer.java line 147).
func placeLoopLabel(e *Edge, n *Node, y, labelGap float64, margin *Spacing) float64 {
	if len(e.Labels) == 0 {
		return 0
	}
	label := e.Labels[0]
	label.Pos = Vec{(n.Size.X - label.Size.X) / 2, y - (labelGap + label.Size.Y)}
	growMargin(margin, n.Size, label.Pos)
	growMargin(margin, n.Size, label.Pos.add(label.Size))
	if label.Size.Y > 0 {
		return label.Size.Y + labelGap
	}
	return 0
}

// growMargin makes m reach p, relative to a node of a size, p being above the
// node's bottom (updateNewNodeMargins, OrthogonalSelfLoopRouter.java line 187).
func growMargin(m *Spacing, size, p Vec) {
	m.Left = max(m.Left, -p.X)
	m.Right = max(m.Right, p.X-size.X)
	m.Top = max(m.Top, -p.Y)
}

// restoreSelfLoops joins self loops to their ports again, with bends and labels
// placed in the graph (process, intermediate/SelfLoopPostProcessor.java line
// 46). Going down or up, ELK aligns a label to its bottom, which adds and takes
// off its height (applyPlacementVerticalForVerticalLayout,
// intermediate/loops/SelfHyperLoopLabels.java line 179) and can change the last
// bit.
func restoreSelfLoops(g *Graph) {
	for _, n := range nodes(g) {
		for _, l := range n.Loops {
			setSource(l.edge, l.source)
			setTarget(l.edge, l.target)
			for i, b := range l.edge.Bends {
				l.edge.Bends[i] = b.add(n.Pos)
			}
			for _, label := range l.edge.Labels {
				if vertical(g.Direction) {
					label.Pos.Y = label.Pos.Y + label.Size.Y - label.Size.Y
				}
				label.Pos = label.Pos.add(n.Pos)
			}
		}
	}
}

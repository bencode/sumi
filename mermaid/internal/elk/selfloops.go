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
// intermediate/SelfLoopPreProcessor.java lines 51, 88). A subgraph with external
// ports keeps the ports, which have dummy nodes inside it.
func hideSelfLoops(g *Graph) {
	for _, n := range g.Layerless {
		for _, e := range outgoing(n) {
			if e.Target.Owner == n {
				n.Loops = append(n.Loops, selfLoop{e, e.Source, e.Target})
				if !n.FixedPorts {
					n.Ports = slices.DeleteFunc(n.Ports, func(p *Port) bool { return p == e.Source || p == e.Target })
				}
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
		if n.FixedPorts {
			continue
		}
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
			y -= placeLoopLabels(l.edge, n, y, s.edgeLabel, vertical(g.Direction), &margin) + s.edgeEdge
		}
		n.Margin = margin
	}
}

// placeLoopLabels puts all of e's labels above the loop at y, as one block
// centered on n, and gives the room it takes (placeLabels and computePositions,
// OrthogonalSelfLoopRouter.java lines 139, 261; computeCoordinates,
// intermediate/loops/routing/LabelPlacer.java line 372). The block's labels run
// along the layers, each after the last (applyPlacement,
// intermediate/loops/SelfHyperLoopLabels.java line 147). Going down or up, they
// are aligned to the block's bottom, otherwise centered. A loop on one side drops
// its labels' inline option (LabelPlacer.java line 147).
func placeLoopLabels(e *Edge, n *Node, y, labelGap float64, alongX bool, margin *Spacing) float64 {
	if len(e.Labels) == 0 {
		return 0
	}
	size := loopLabelsSize(e.Labels, alongX)
	at := Vec{(n.Size.X - size.X) / 2, y - (labelGap + size.Y)}
	next := at
	for _, l := range e.Labels {
		if alongX {
			l.Pos = Vec{next.X, at.Y + size.Y - l.Size.Y}
			next.X += l.Size.X
		} else {
			l.Pos = Vec{at.X + (size.X-l.Size.X)/2, next.Y}
			next.Y += l.Size.Y
		}
	}
	growMargin(margin, n.Size, at)
	growMargin(margin, n.Size, at.add(size))
	if size.Y > 0 {
		return size.Y + labelGap
	}
	return 0
}

// loopLabelsSize gives the size of a block of labels (updateSize,
// SelfHyperLoopLabels.java line 109). spacing.labelLabel, which would part them,
// is 0 at Mermaid's base value.
func loopLabelsSize(labels []*Label, alongX bool) Vec {
	var size Vec
	for _, l := range labels {
		if alongX {
			size = Vec{size.X + l.Size.X, max(size.Y, l.Size.Y)}
		} else {
			size = Vec{max(size.X, l.Size.X), size.Y + l.Size.Y}
		}
	}
	return size
}

// growMargin makes m reach p, relative to a node of a size
// (updateNewNodeMargins, OrthogonalSelfLoopRouter.java line 187).
func growMargin(m *Spacing, size, p Vec) {
	m.Left = max(m.Left, -p.X)
	m.Right = max(m.Right, p.X-size.X)
	m.Top = max(m.Top, -p.Y)
	m.Bottom = max(m.Bottom, p.Y-size.Y)
}

// restoreSelfLoops joins self loops to their ports again, with bends and labels
// placed in the graph (process, intermediate/SelfLoopPostProcessor.java line
// 46).
func restoreSelfLoops(g *Graph) {
	for _, n := range nodes(g) {
		for _, l := range n.Loops {
			setSource(l.edge, l.source)
			setTarget(l.edge, l.target)
			for i, b := range l.edge.Bends {
				l.edge.Bends[i] = b.add(n.Pos)
			}
			for _, label := range l.edge.Labels {
				label.Pos = label.Pos.add(n.Pos)
			}
		}
	}
}

// SPDX-License-Identifier: EPL-2.0 OR GPL-3.0-or-later

package elk

import (
	"math"
	"slices"
)

func vertical(d Direction) bool { return d == Down || d == Up }

// insertLabelDummies cuts each edge with a label in two around a dummy node the
// size of the label, so that the label takes room in the layers (process,
// intermediate/LabelDummyInserter.java line 65). Input's labels are center ones,
// the only kind this package takes, and self loops are out of the way by now.
// ELK moves a label to the dummy and back to the edge; here it stays on the edge,
// as that half is the one to survive when the dummy goes.
func insertLabelDummies(g *Graph) {
	var dummies []*Node
	for _, n := range g.Layerless {
		for _, e := range outgoing(n) {
			if len(e.Labels) > 0 {
				dummy := &Node{Type: typeLabel}
				splitEdge(e, dummy)
				dummy.Size = labelRoom(e.Labels[0].Size, g.Direction, g.Settings.edgeLabel)
				dummies = append(dummies, dummy)
			}
		}
	}
	g.Layerless = append(g.Layerless, dummies...)
}

// labelRoom gives the size of the dummy for a label of a size: the height a
// dummy starts with, then what LabelDummyInserter.java adds in lines 94-114.
// Going down or up, labels sit beside the edge instead of on top of it, so the
// starting height is only a minimum. The sum runs in ELK's order, which can
// change its last bit. spacing.labelLabel is 0 at Mermaid's base value, and
// Input holds one label per edge, so nothing separates labels.
func labelRoom(size Vec, d Direction, labelGap float64) Vec {
	if vertical(d) {
		return Vec{size.X, max(edgeThickness, size.Y) + (labelGap + edgeThickness)}
	}
	return Vec{size.X, edgeThickness + size.Y + (labelGap + edgeThickness)}
}

// switchLabelDummies moves each label dummy to the lower median of the layers
// its long edge crosses, Mermaid leaving the strategy at MEDIAN_LAYER (process,
// intermediate/LabelDummySwitcher.java line 99, and findMedianLayerTargetId,
// line 329). An edge has one dummy at most, so moving one leaves the others'
// long edges alone.
func switchLabelDummies(g *Graph) {
	layerOf := map[*Node]*Layer{}
	var dummies []*Node
	for _, l := range g.Layers {
		for _, n := range l.Nodes {
			layerOf[n] = l
			if n.Type == typeLabel {
				dummies = append(dummies, n)
			}
		}
	}
	for _, n := range dummies {
		chain := longEdgeThrough(n)
		if median := chain[(len(chain)-1)/2]; median != n {
			swapNodes(n, median, layerOf[n], layerOf[median])
		}
	}
}

// longEdgeThrough gives the dummy nodes of n's long edge from left to right,
// one per layer, n among them (LabelDummyInfo, LabelDummySwitcher.java line 665).
func longEdgeThrough(n *Node) []*Node {
	chain := []*Node{n}
	for m := incoming(n)[0].Source.Owner; m.Type == typeLongEdge; m = incoming(m)[0].Source.Owner {
		chain = append(chain, m)
	}
	slices.Reverse(chain)
	for m := outgoing(n)[0].Target.Owner; m.Type == typeLongEdge; m = outgoing(m)[0].Target.Owner {
		chain = append(chain, m)
	}
	return chain
}

// swapNodes trades the places of a, in layer la, and b, in layer lb, with the
// edges at them (LabelDummySwitcher.java line 556). Both have one edge in and
// one out, and when they are neighbours the edge between them turns around.
func swapNodes(a, b *Node, la, lb *Layer) {
	la.Nodes[slices.Index(la.Nodes, a)], lb.Nodes[slices.Index(lb.Nodes, b)] = b, a
	aIn, aOut, bIn, bOut := portOn(a, sideWest), portOn(a, sideEast), portOn(b, sideWest), portOn(b, sideEast)
	intoA, outOfA, intoB, outOfB := aIn.In[0], aOut.Out[0], bIn.In[0], bOut.Out[0]
	setTarget(intoB, aIn)
	setSource(outOfB, aOut)
	setTarget(intoA, bIn)
	setSource(outOfA, bOut)
}

// sizeLabelDummies takes off the room meant for the edge, as Mermaid's labels
// sit on their edges, and runs the edge through the middle of the dummy
// (applyLabelSide, intermediate/LabelSideSelector.java line 373). Inline labels
// skip the choice of side that gives the step its name.
func sizeLabelDummies(g *Graph) {
	for _, n := range nodes(g) {
		if n.Type == typeLabel {
			y := math.Ceil(n.Size.Y-g.Settings.edgeLabel-edgeThickness) / 2
			n.Size.Y -= g.Settings.edgeLabel
			n.Size.Y -= edgeThickness
			for _, p := range n.Ports {
				p.Pos.Y = y
			}
		}
	}
}

// removeLabelDummies gives each label the place of its dummy and joins the edge
// again (process and placeLabelsForVerticalLayout,
// intermediate/LabelDummyRemover.java lines 63, 145). Going down or up, a dummy
// can be taller than its label, which is then centered; a dummy is never wider.
// Long edges are joined by now, so the edge into the dummy is the first half,
// holding the label. A dummy leaves no bend, as ELK adds one only for polyline
// routing.
func removeLabelDummies(g *Graph) {
	isDummy := func(n *Node) bool { return n.Type == typeLabel }
	for _, l := range g.Layers {
		for _, n := range l.Nodes {
			if isDummy(n) {
				label := portOn(n, sideWest).In[0].Labels[0]
				label.Pos = n.Pos
				if vertical(g.Direction) {
					label.Pos.Y += (n.Size.Y - label.Size.Y) / 2
				}
				joinAt(n, false)
			}
		}
		l.Nodes = slices.DeleteFunc(l.Nodes, isDummy)
	}
}

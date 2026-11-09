// SPDX-License-Identifier: EPL-2.0 OR GPL-3.0-or-later

package elk

import (
	"cmp"
	"slices"
)

// placeEndLabels puts each head and tail label beside its edge, just off its
// node, and grows the node's margins around them, which is what pushes the nodes
// apart (process, intermediate/EndLabelPreprocessor.java line 76). A label's Pos
// is relative to its node until applyEndLabels. Labels only come on edges between
// leaf nodes, whose ports are on the east and west sides here. The labels' sides
// are only chosen later, by LabelSideSelector.java, so the test for the upper
// side in placeLabels (line 285) always fails, and a label goes below its edge. A
// port holds one edge, as Mermaid sets mergeEdges to false, so it holds one such
// label, and the sorting of intermediate/EndLabelSorter.java has nothing to do.
func placeEndLabels(g *Graph) {
	spacing := g.Settings.edgeLabel
	for _, n := range nodes(g) {
		east := n.Size.X + n.Margin.Right + spacing
		west := -n.Margin.Left - spacing
		sides := map[PortSide][]*Label{}
		for _, p := range n.Ports {
			if l := endLabel(p); l != nil {
				x := east
				if p.Side != sideEast {
					x = west - l.Size.X
				}
				l.Pos = Vec{x, p.Pos.Y + edgeThickness + spacing}
				sides[p.Side] = append(sides[p.Side], l)
			}
		}
		if vertical(g.Direction) {
			stack(sides[sideEast], func(l *Label, depth float64) { l.Pos.X = east + depth })
			stack(sides[sideWest], func(l *Label, depth float64) { l.Pos.X = west - l.Size.X - depth })
		}
		for _, labels := range sides {
			for _, l := range labels {
				growMargin(&n.Margin, n.Size, l.Pos)
				growMargin(&n.Margin, n.Size, l.Pos.add(l.Size))
			}
		}
	}
}

// endLabel gives the label at p's end of its edge, if there is one
// (gatherLabels, EndLabelPreprocessor.java line 169).
func endLabel(p *Port) *Label {
	for _, e := range p.Out {
		if l := labelOf(e, tailLabel); l != nil {
			return l
		}
	}
	for _, e := range p.In {
		if l := labelOf(e, headLabel); l != nil {
			return l
		}
	}
	return nil
}

// stack moves apart the labels that overlap along a side of a node, the first
// nearest the node, and hands each its distance from there. Going down or up, the
// ports on one side are side by side, so their labels can collide
// (removeLabelOverlaps, EndLabelPreprocessor.java line 365; alg.common
// overlaps/RectangleStripOverlapRemover.java and
// GreedyRectangleStripOverlapRemover.java). The sweep and the greedy pass are one
// loop here, as only the labels before a label can have been placed. ELK keeps
// the labels under the sweep in a set ordered by right border, so of two with the
// same right border only the first stays. spacing.labelLabel, which would part
// them, is 0 at Mermaid's base value.
func stack(labels []*Label, place func(*Label, float64)) {
	type placed struct{ right, size, depth float64 }
	slices.SortStableFunc(labels, func(a, b *Label) int { return cmp.Compare(a.Pos.Y, b.Pos.Y) })
	var under []placed
	for _, l := range labels {
		right, size, depth := l.Pos.Y+l.Size.Y, l.Size.X, 0.0
		under = slices.DeleteFunc(under, func(o placed) bool { return o.right < l.Pos.Y })
		slices.SortFunc(under, func(a, b placed) int { return cmp.Compare(a.depth, b.depth) })
		for _, o := range under {
			if depth < o.depth+o.size && depth+size > o.depth {
				depth = o.depth + o.size
			}
		}
		if !slices.ContainsFunc(under, func(o placed) bool { return o.right == right }) {
			under = append(under, placed{right, size, depth})
		}
		place(l, depth)
	}
}

// applyEndLabels gives the labels their places, now their nodes have theirs
// (process, intermediate/EndLabelPostprocessor.java line 53).
func applyEndLabels(g *Graph) {
	for _, n := range nodes(g) {
		for _, p := range n.Ports {
			if l := endLabel(p); l != nil {
				l.Pos = l.Pos.add(n.Pos)
			}
		}
	}
}

// SPDX-License-Identifier: EPL-2.0 OR GPL-3.0-or-later

package elk

import (
	"cmp"
	"slices"
)

// endBox is the room of a head or tail label, relative to its node until the
// node has its place (a LabelCell in InternalProperties.END_LABELS). A port holds
// one edge, as Mermaid sets mergeEdges to false, so it holds one such label, and
// the sorting of intermediate/EndLabelSorter.java has nothing to do.
type endBox struct {
	label *Label
	side  PortSide
	pos   Vec
}

// placeEndLabels puts each head and tail label beside its edge, just off its
// node, and grows the node's margins around them, which is what pushes the nodes
// apart (process, intermediate/EndLabelPreprocessor.java line 76). Labels only
// come on edges between leaf nodes, whose ports are on the east and west sides
// here. The labels' sides are only chosen later, by LabelSideSelector.java, so
// the test for the upper side in placeLabels (line 285) always fails, and a
// label goes below its edge.
func placeEndLabels(g *Graph) {
	spacing := g.Settings.edgeLabel
	for _, n := range nodes(g) {
		east := n.Size.X + n.Margin.Right + spacing
		west := -n.Margin.Left - spacing
		for _, p := range n.Ports {
			if l := endLabel(p); l != nil {
				x := east
				if p.Side != sideEast {
					x = west - l.Size.X
				}
				n.endBoxes = append(n.endBoxes, &endBox{l, p.Side, Vec{x, p.Pos.Y + edgeThickness + spacing}})
			}
		}
		if len(n.endBoxes) == 0 {
			continue
		}
		if vertical(g.Direction) {
			stack(sideBoxes(n, sideEast), func(b *endBox, depth float64) { b.pos.X = east + depth })
			stack(sideBoxes(n, sideWest), func(b *endBox, depth float64) { b.pos.X = west - b.label.Size.X - depth })
		}
		growMarginToBoxes(n)
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

func sideBoxes(n *Node, side PortSide) []*endBox {
	var boxes []*endBox
	for _, b := range n.endBoxes {
		if b.side == side {
			boxes = append(boxes, b)
		}
	}
	return boxes
}

// strip is a box in a strip running along a node's side: it spans left to right
// along the side, and depth away from it.
type strip struct {
	box         *endBox
	left, right float64
	size, depth float64
	placed      bool
	overlaps    []*strip
}

// stack moves apart the boxes that overlap along a side, the first nearest the
// node, and hands each its distance from there. Going down or up, the ports on one
// side of a node are side by side, so their labels can collide (removeLabelOverlaps,
// EndLabelPreprocessor.java line 365; alg.common
// overlaps/RectangleStripOverlapRemover.java and
// GreedyRectangleStripOverlapRemover.java). spacing.labelLabel, which would part
// them, is 0 at Mermaid's base value.
func stack(boxes []*endBox, place func(*endBox, float64)) {
	strips := make([]*strip, len(boxes))
	for i, b := range boxes {
		strips[i] = &strip{box: b, left: b.pos.Y, right: b.pos.Y + b.label.Size.Y, size: b.label.Size.X}
	}
	slices.SortStableFunc(strips, func(a, b *strip) int { return cmp.Compare(a.left, b.left) })
	findOverlaps(strips)
	for _, s := range strips {
		slices.SortStableFunc(s.overlaps, func(a, b *strip) int { return cmp.Compare(a.depth, b.depth) })
		for _, o := range s.overlaps {
			if o.placed && s.depth < o.depth+o.size && s.depth+s.size > o.depth {
				s.depth = o.depth + o.size
			}
		}
		s.placed = true
		place(s.box, s.depth)
	}
}

// findOverlaps sweeps along the strip, which is sorted by left border. ELK keeps
// the strips under the sweep in a set ordered by right border, so of two with the
// same right border only the first joins it (computeOverlaps,
// RectangleStripOverlapRemover.java line 263).
func findOverlaps(strips []*strip) {
	var active []*strip
	for _, s := range strips {
		for len(active) > 0 && active[0].right < s.left {
			active = active[1:]
		}
		for _, a := range active {
			a.overlaps = append(a.overlaps, s)
			s.overlaps = append(s.overlaps, a)
		}
		byRight := func(a *strip, right float64) int { return cmp.Compare(a.right, right) }
		if i, found := slices.BinarySearchFunc(active, s.right, byRight); !found {
			active = slices.Insert(active, i, s)
		}
	}
}

// growMarginToBoxes unites n's margin box with its end boxes (updateNodeMargins,
// EndLabelPreprocessor.java line 420).
func growMarginToBoxes(n *Node) {
	m := n.Margin
	x, y := -m.Left, -m.Top
	width, height := m.Left+n.Size.X+m.Right, m.Top+n.Size.Y+m.Bottom
	for _, b := range n.endBoxes {
		x1, y1 := min(x, b.pos.X), min(y, b.pos.Y)
		x2 := max(x+width, b.pos.X+b.label.Size.X)
		y2 := max(y+height, b.pos.Y+b.label.Size.Y)
		x, y, width, height = x1, y1, x2-x1, y2-y1
	}
	left, top := -x, -y
	n.Margin = Spacing{Left: left, Top: top, Right: width - left - n.Size.X, Bottom: height - top - n.Size.Y}
}

// applyEndLabels gives the labels their places, now their nodes have theirs
// (process, intermediate/EndLabelPostprocessor.java line 53).
func applyEndLabels(g *Graph) {
	for _, n := range nodes(g) {
		for _, b := range n.endBoxes {
			b.label.Pos = b.pos.add(n.Pos)
		}
		n.endBoxes = nil
	}
}

// SPDX-License-Identifier: EPL-2.0 OR GPL-3.0-or-later

package elk

import "math"

// stitchEdges puts the segments of root's cross edges back together and takes them
// off the graphs (process, compound/CompoundGraphPostprocessor.java line 75). The
// result goes straight to what export gives, as nothing reads these edges after.
func stitchEdges(root *Graph) {
	for _, c := range root.Crossing {
		reference := referenceGraph(c)
		bends, targetOffset, labels := joinSegments(c, reference, root.Settings.keepBends)
		c.placed = placeCrossEdge(c, reference, bends, targetOffset, labels)
		for _, s := range c.segments {
			setSource(s.edge, nil)
			setTarget(s.edge, nil)
		}
	}
}

// referenceGraph gives the graph c's bends are worked out in: the one its source
// sits in, or the inside of the source if the edge runs down into it
// (CompoundGraphPostprocessor.java line 103).
func referenceGraph(c *crossEdge) *Graph {
	if isDescendant(c.target.Owner, c.source.Owner) {
		return c.source.Owner.Nested
	}
	return c.source.Owner.Graph
}

// joinSegments gives c's bends in reference, where its target sits relative to
// that, and its labels. Where two segments meet, the point between them becomes a
// bend if the line would not otherwise run through it
// (CompoundGraphPostprocessor.java lines 119 to 177).
func joinSegments(c *crossEdge, reference *Graph, keepBends bool) ([]Vec, Vec, []*Label) {
	var bends []Vec
	var labels []*Label
	var last *Vec
	var targetOffset Vec
	for i, s := range c.segments {
		offset := changeCoords(Vec{}, s.graph, reference)
		points := make([]Vec, len(s.edge.Bends))
		for j, b := range s.edge.Bends {
			points[j] = b.add(offset)
		}
		source, target := absolute(s.edge.Source).add(offset), absolute(s.edge.Target).add(offset)
		if last != nil {
			next := target
			if len(points) > 0 {
				next = points[0]
			}
			if apart(*last, next, keepBends) {
				bends = append(bends, source)
			}
		}
		bends = append(bends, points...)
		last = &source
		if len(points) > 0 {
			last = &points[len(points)-1]
		}
		if i == len(c.segments)-1 {
			targetOffset = offset
			if c.target.Owner.Graph != s.graph {
				targetOffset = changeCoords(Vec{}, c.target.Owner.Graph, reference)
			}
		}
		for _, l := range s.edge.Labels {
			l.Pos = changeCoords(l.Pos, s.graph, reference)
			labels = append(labels, l)
		}
	}
	return bends, targetOffset, labels
}

// apart tells whether two points call for a bend between them: with keepBends if
// they differ at all, and otherwise only if a straight line would be slanted.
func apart(a, b Vec, keepBends bool) bool {
	x, y := math.Abs(a.X-b.X) > tolerance, math.Abs(a.Y-b.Y) > tolerance
	if keepBends {
		return x || y
	}
	return x && y
}

// changeCoords moves v from from's coordinates to to's, by way of the top of the
// layout (changeCoordSystem, graph/LGraphUtil.java line 1058).
func changeCoords(v Vec, from, to *Graph) Vec {
	if from == to {
		return v
	}
	for g := from; ; g = g.Parent.Graph {
		v = v.add(g.Offset)
		if !included(g) {
			break
		}
		v = v.add(Vec{g.Padding.Left, g.Padding.Top}).add(g.Parent.Pos)
	}
	for g := to; ; g = g.Parent.Graph {
		v = v.sub(g.Offset)
		if !included(g) {
			break
		}
		v = v.sub(Vec{g.Padding.Left, g.Padding.Top}).sub(g.Parent.Pos)
	}
	return v
}

// placeCrossEdge writes c out in the coordinates of the graph holding it
// (applyEdgeLayout, graph/transform/ElkGraphLayoutTransferrer.java line 233). An edge
// from a subgraph into it needs its source point taken out of the offset, as
// a port is relative to its node. Other edges rise from reference to their
// container (calculateHierarchicalOffset, line 326; findCoordinateSystemOrigin,
// ElkGraphImporter.java line 535).
func placeCrossEdge(c *crossEdge, reference *Graph, bends []Vec, targetOffset Vec, labels []*Label) PlacedEdge {
	offset := Vec{reference.Offset.X + reference.Padding.Left, reference.Offset.Y + reference.Padding.Top}
	edgeOffset := offset
	source := absolute(c.source)
	if isDescendant(c.target.Owner, c.source.Owner) {
		source = c.source.Pos.sub(offset)
	} else {
		edgeOffset = offset.add(riseTo(reference, c.edge.Container))
	}
	points := make([]Vec, 0, len(bends)+2)
	points = append(points, source.add(edgeOffset))
	for _, b := range bends {
		points = append(points, b.add(edgeOffset))
	}
	points = append(points, absolute(c.target).add(targetOffset).add(edgeOffset))
	center, tail, head := placedLabels(labels, edgeOffset)
	return PlacedEdge{ID: c.edge.ID, Container: containerID(c.edge.Container), Points: points, Labels: center, Tail: tail, Head: head}
}

// riseTo adds up what lies between g's coordinates and top's.
func riseTo(g, top *Graph) Vec {
	var total Vec
	for g != top {
		node := g.Parent
		g = node.Graph
		total = total.add(node.Pos).add(g.Offset).add(Vec{g.Padding.Left, g.Padding.Top})
	}
	return total
}

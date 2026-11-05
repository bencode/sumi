// SPDX-License-Identifier: EPL-2.0 OR GPL-3.0-or-later

package elk

import "math"

// Spacing between layers, at Mermaid's base value.
const (
	layerGap = 40.0 // spacing.nodeNodeBetweenLayers
	slotGap  = 20.0 // spacing.edgeEdgeBetweenLayers
	sideGap  = 20.0 // spacing.edgeNodeBetweenLayers
)

// measureLayers gives each layer its width and g its height, moving the top of
// the drawing to where the graph starts
// (intermediate/LayerSizeAndGraphHeightCalculator.java line 49).
func measureLayers(g *Graph) {
	if len(g.Layers) == 0 {
		return
	}
	low, high := math.Inf(1), math.Inf(-1)
	for _, l := range g.Layers {
		l.Width = 0
		for _, n := range l.Nodes {
			l.Width = max(l.Width, n.Size.X+n.Margin.Left+n.Margin.Right)
		}
		first, last := l.Nodes[0], l.Nodes[len(l.Nodes)-1]
		low = min(low, first.Pos.Y-first.Margin.Top)
		high = max(high, last.Pos.Y+last.Size.Y+last.Margin.Bottom)
	}
	g.Size.Y = high - low
	g.Offset.Y -= low
}

// routeEdges places the layers from left to right and routes the edges between
// each two, which take as much room as they need (process,
// p5edges/OrthogonalEdgeRouter.java line 217). With no ports outside the
// layers, nothing routes before the first layer or after the last, and two
// side gaps are never less than ELK's minimum of a layer gap.
func routeEdges(g *Graph) {
	r := &router{random: g.Random}
	x := 0.0
	for i, left := range g.Layers {
		placeHorizontally(left, x)
		x += left.Width
		if i == len(g.Layers)-1 {
			break
		}
		slots := r.route(left.Nodes, g.Layers[i+1].Nodes, x+sideGap)
		gap := layerGap
		if slots > 0 {
			gap = float64(slots-1)*slotGap + sideGap + sideGap
		}
		x += gap
	}
	g.Size.X = x
}

// placeHorizontally gives l's nodes their x, each nearer the side with more of
// its edges (placeNodesHorizontally, graph/LGraphUtil.java line 223).
func placeHorizontally(l *Layer, offset float64) {
	left, right := 0.0, 0.0
	for _, n := range l.Nodes {
		left = max(left, n.Margin.Left)
		right = max(right, n.Margin.Right)
	}
	for _, n := range l.Nodes {
		ratio := portRatio(n)
		x := float64((l.Width - n.Size.X) * ratio)
		if ratio > 0.5 {
			x -= float64(right * 2 * (ratio - 0.5))
		} else if ratio < 0.5 {
			x += float64(left * 2 * (0.5 - ratio))
		}
		x = max(x, n.Margin.Left)
		if limit := l.Width - n.Margin.Right - n.Size.X; x > limit {
			x = limit
		}
		n.Pos.X = offset + x
	}
}

// portRatio is the share of n's ports with edges that lead out.
func portRatio(n *Node) float64 {
	in, out := 0, 0
	for _, p := range n.Ports {
		if len(p.In) > 0 {
			in++
		}
		if len(p.Out) > 0 {
			out++
		}
	}
	if in+out == 0 {
		return 0.5
	}
	return float64(out) / float64(in+out)
}

// collectNodes puts all nodes back in Layerless (process,
// intermediate/HierarchicalNodeResizingProcessor.java line 54). Mermaid sets no
// minimum size, so a graph stays as big as it is, but ELK still adds the padding
// and takes it off again (line 225), which can change the last bit of its size.
func collectNodes(g *Graph) {
	for _, l := range g.Layers {
		g.Layerless = append(g.Layerless, l.Nodes...)
	}
	g.Layers = nil
	p := g.Padding
	g.Size.X = g.Size.X + p.Left + p.Right - p.Left - p.Right
	g.Size.Y = g.Size.Y + p.Top + p.Bottom - p.Top - p.Bottom
}

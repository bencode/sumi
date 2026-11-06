// SPDX-License-Identifier: EPL-2.0 OR GPL-3.0-or-later

package elk

import (
	"math"
	"slices"
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
		top, bottom := first.Pos.Y-first.Margin.Top, last.Pos.Y+last.Size.Y+last.Margin.Bottom
		if first.Type == typeExternalPort {
			top -= g.Settings.portsSurrounding
		}
		if last.Type == typeExternalPort {
			bottom += g.Settings.portsSurrounding
		}
		low = min(low, top)
		high = max(high, bottom)
	}
	g.Size.Y = high - low
	g.Offset.Y -= low
}

// routeEdges places the layers from left to right and routes the edges between
// each two, which take as much room as they need, and those before the first
// layer and after the last, where inverted ports go round (process,
// p5edges/OrthogonalEdgeRouter.java line 217). Two side gaps are never less than
// ELK's minimum of a layer gap.
func routeEdges(g *Graph) {
	s := g.Settings
	r := &router{random: g.Random, slotGap: s.edgeEdgeLayers}
	x := 0.0
	var left *Layer
	for i := 0; i <= len(g.Layers); i++ {
		var right *Layer
		if i < len(g.Layers) {
			right = g.Layers[i]
		}
		start := x
		if left != nil {
			placeHorizontally(left, x)
			x += left.Width
			start = x + s.edgeNodeLayers
		}
		x += routingWidth(s, left, right, r.route(nodesOf(left), nodesOf(right), start))
		left = right
	}
	g.Size.X = x
}

// routingWidth is the room between two layers with slots slots. Either may be
// missing, on the outside of the graph.
func routingWidth(s settings, left, right *Layer, slots int) float64 {
	if slots == 0 {
		if externalLayer(left) || externalLayer(right) {
			return 0
		}
		return s.nodeLayers
	}
	width := float64(slots-1) * s.edgeEdgeLayers
	if left != nil {
		width += s.edgeNodeLayers
	}
	if right != nil {
		width += s.edgeNodeLayers
	}
	return width
}

func nodesOf(l *Layer) []*Node {
	if l == nil {
		return nil
	}
	return l.Nodes
}

// externalLayer tells whether l holds external port dummies alone, a missing
// layer counting as such (PRED_EXTERNAL_WEST_OR_EAST_PORT,
// p5edges/PolylineEdgeRouter.java line 65).
func externalLayer(l *Layer) bool {
	return l == nil || !slices.ContainsFunc(l.Nodes, func(n *Node) bool { return n.Type != typeExternalPort })
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
// intermediate/HierarchicalNodeResizingProcessor.java line 54).
func collectNodes(g *Graph) {
	for _, l := range g.Layers {
		g.Layerless = append(g.Layerless, l.Nodes...)
	}
	g.Layers = nil
}

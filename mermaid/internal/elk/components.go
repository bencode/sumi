// SPDX-License-Identifier: EPL-2.0 OR GPL-3.0-or-later

package elk

import (
	"cmp"
	"math"
	"slices"
)

// aspectRatio is the option's default, a float (Layered.melk line 125).
const aspectRatio = float64(float32(1.6))

// splitComponents gives a graph for each connected component of g, nodes in the
// order a depth first search meets them (split and dfs,
// components/ComponentsProcessor.java lines 91 and 185). The components share g's
// random numbers, each carrying on where the one before stopped.
func splitComponents(g *Graph) []*Graph {
	seen := map[*Node]bool{}
	var all []*Graph
	for _, n := range g.Layerless {
		if !seen[n] {
			part := &Graph{Padding: g.Padding, Direction: g.Direction, Settings: g.Settings, Random: g.Random}
			collectComponent(n, part, seen)
			all = append(all, part)
		}
	}
	return all
}

// collectComponent adds what n connects to, port by port (getConnectedPorts,
// graph/LPort.java line 352).
func collectComponent(n *Node, part *Graph, seen map[*Node]bool) {
	if seen[n] {
		return
	}
	seen[n] = true
	part.Layerless = append(part.Layerless, n)
	for _, p := range n.Ports {
		for _, e := range p.In {
			collectComponent(e.Source.Owner, part, seen)
		}
		for _, e := range p.Out {
			collectComponent(e.Target.Owner, part, seen)
		}
	}
}

// combineComponents puts all back in g, smallest first in rows as wide as the
// aspect ratio asks (combine, components/SimpleRowGraphPlacer.java line 43). A
// lone component keeps its offset, which the rows reset; Mermaid sets no priority
// and leaves compaction off.
func combineComponents(all []*Graph, g *Graph) {
	g.Layerless = nil
	if len(all) == 1 {
		moveNodes(g, all[0], all[0].Offset)
		g.Size = all[0].Size
		return
	}
	slices.SortStableFunc(all, func(a, b *Graph) int { return cmp.Compare(area(a), area(b)) })
	widest, total := 0.0, 0.0
	for _, part := range all {
		widest = max(widest, part.Size.X)
		total += area(part)
	}
	rowWidth := max(widest, math.Sqrt(total)*aspectRatio)
	placeComponents(all, g, rowWidth, g.Settings.component)
}

// area keeps Go from fusing its product into a sum, which JavaScript rounds apart.
func area(g *Graph) float64 { return float64(g.Size.X * g.Size.Y) }

// placeComponents fills rows from the left and sizes g around them
// (placeComponents, SimpleRowGraphPlacer.java line 142).
func placeComponents(all []*Graph, g *Graph, rowWidth, spacing float64) {
	x, y, highest, broadest := 0.0, 0.0, 0.0, spacing
	for _, part := range all {
		if x+part.Size.X > rowWidth {
			x = 0
			y += highest + spacing
			highest = 0
		}
		moveNodes(g, part, Vec{x + part.Offset.X, y + part.Offset.Y})
		broadest = max(broadest, x+part.Size.X)
		highest = max(highest, part.Size.Y)
		x += part.Size.X + spacing
	}
	g.Size = Vec{broadest, y + highest}
}

// moveNodes moves part's nodes to g, by offset, with their edges
// (moveGraph and offsetGraph, components/AbstractGraphPlacer.java lines 75 and 124).
func moveNodes(g, part *Graph, offset Vec) {
	for _, n := range part.Layerless {
		n.Pos = n.Pos.add(offset)
		for _, e := range outgoing(n) {
			for i := range e.Bends {
				e.Bends[i] = e.Bends[i].add(offset)
			}
			for _, l := range e.Labels {
				l.Pos = l.Pos.add(offset)
			}
		}
		g.Layerless = append(g.Layerless, n)
	}
}

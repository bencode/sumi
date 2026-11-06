// SPDX-License-Identifier: EPL-2.0 OR GPL-3.0-or-later

package elk

// toLTR turns g so that edges go left to right, the only way the steps after
// it lay graphs out. It flips rather than rotates, which keeps reading order:
// Mermaid leaves directionCongruency at READING_DIRECTION
// (intermediate/GraphTransformer.java line 72).
//
// ELK's mode names run the other way round: the step before the phases is
// built with TO_INPUT_DIRECTION, the step after them with TO_INTERNAL_LTR
// (intermediate/IntermediateProcessorStrategy.java lines 211-215). So going
// in, Up mirrors first.
func toLTR(g *Graph) {
	if g.Direction == Left {
		mirrorX(g)
	}
	if g.Direction == Up {
		mirrorY(g)
	}
	if g.Direction == Down || g.Direction == Up {
		transpose(g)
	}
}

// fromLTR undoes toLTR (GraphTransformer.java line 72).
func fromLTR(g *Graph) {
	if g.Direction == Left {
		mirrorX(g)
	}
	if g.Direction == Down || g.Direction == Up {
		transpose(g)
	}
	if g.Direction == Up {
		mirrorY(g)
	}
}

// mirrorX flips g left to right (mirrorAllX and mirrorX, GraphTransformer.java
// lines 157, 187).
func mirrorX(g *Graph) {
	far := func(m Spacing) float64 { return m.Right }
	mirror(g, func(v *Vec) *float64 { return &v.X }, far, map[PortSide]PortSide{sideEast: sideWest, sideWest: sideEast})
	g.Padding.Left, g.Padding.Right = g.Padding.Right, g.Padding.Left
}

// mirrorY flips g top to bottom (mirrorAllY and mirrorY, GraphTransformer.java
// lines 163, 399).
func mirrorY(g *Graph) {
	far := func(m Spacing) float64 { return m.Bottom }
	mirror(g, func(v *Vec) *float64 { return &v.Y }, far, map[PortSide]PortSide{sideNorth: sideSouth, sideSouth: sideNorth})
	g.Padding.Top, g.Padding.Bottom = g.Padding.Bottom, g.Padding.Top
}

// mirror flips g along the axis that at picks. far is the margin on the far side
// of a node, and swapped gives the sides that trade places. ELK also mirrors the placement of node labels; here that is the
// direction titlePosition looks at.
func mirror(g *Graph, at func(*Vec) *float64, far func(Spacing) float64, swapped map[PortSide]PortSide) {
	all := nodes(g)
	offset := mirrorOffset(g, all, at, far)
	flip := func(v *Vec, across float64) { *at(v) = across - *at(v) }
	for _, n := range all {
		flip(&n.Pos, offset-*at(&n.Size))
		if side, ok := swapped[n.ExtSide]; ok {
			n.ExtSide = side
		}
		if n.Title != nil {
			flip(&n.Title.Pos, *at(&n.Size)-*at(&n.Title.Size))
		}
		for _, p := range n.Ports {
			flip(&p.Pos, *at(&n.Size))
			if side, ok := swapped[p.Side]; ok {
				p.Side = side
			}
			for _, e := range p.Out {
				for i := range e.Bends {
					flip(&e.Bends[i], offset)
				}
				for _, l := range e.Labels {
					flip(&l.Pos, offset-*at(&l.Size))
				}
			}
		}
	}
}

// mirrorOffset gives the coordinate g flips across. With no size yet that is
// the far edge of the nodes, as on the way in, or on the way out when all nodes
// have no size. Once sized, ELK subtracts the offset twice (GraphTransformer.java
// lines 199-212). ELK never turns margins, so they keep the sides they got.
func mirrorOffset(g *Graph, all []*Node, at func(*Vec) *float64, far func(Spacing) float64) float64 {
	offset := 0.0
	if *at(&g.Size) == 0 {
		for _, n := range all {
			offset = max(offset, *at(&n.Pos)+*at(&n.Size)+far(n.Margin))
		}
	} else {
		offset = *at(&g.Size) - *at(&g.Offset)
	}
	return offset - *at(&g.Offset)
}

// transpose swaps x and y everywhere (transposeAll and transpose,
// GraphTransformer.java lines 169, 589).
func transpose(g *Graph) {
	for _, n := range nodes(g) {
		swap(&n.Pos)
		swap(&n.Size)
		n.ExtSide = transposedSide[n.ExtSide]
		if n.Title != nil {
			swap(&n.Title.Pos)
			swap(&n.Title.Size)
		}
		for _, p := range n.Ports {
			swap(&p.Pos)
			p.Side = transposedSide[p.Side]
			for _, e := range p.Out {
				for i := range e.Bends {
					swap(&e.Bends[i])
				}
				for _, l := range e.Labels {
					swap(&l.Pos)
					swap(&l.Size)
				}
			}
		}
	}
	swap(&g.Offset)
	swap(&g.Size)
	pad := g.Padding
	g.Padding = Spacing{Top: pad.Left, Bottom: pad.Right, Left: pad.Top, Right: pad.Bottom}
}

func swap(v *Vec) { v.X, v.Y = v.Y, v.X }

// transposedSide is transposePortSide (GraphTransformer.java line 745).
var transposedSide = map[PortSide]PortSide{
	sideNorth: sideWest, sideWest: sideNorth, sideSouth: sideEast, sideEast: sideSouth,
}

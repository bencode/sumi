// SPDX-License-Identifier: EPL-2.0 OR GPL-3.0-or-later

package elk

import (
	"cmp"
	"math"
	"slices"
)

// Spacing between neighbors in a layer: spacing.nodeNode, and spacing.edgeNode
// and spacing.edgeEdge at Mermaid's base value, which come to the same.
const (
	nodeGap = 40.0
	edgeGap = 20.0
)

// alignment is one of Brandes and Köpf's four layouts
// (p4nodes/bk/BKAlignedLayout.java). A block is a ring of nodes, each with its
// next, around a root. Sweeping left means from the last layer back, sweeping up
// from the bottom of each layer.
type alignment struct {
	up, left   bool
	root, next map[*Node]*Node
	inner      map[*Node]float64 // innerShift, from the top of the block
	size       map[*Node]float64 // blockSize, by root
	sink       map[*Node]*Node
	shift      map[*Node]float64
	y          map[*Node]float64 // only placed nodes have one
	used       map[*Node]bool    // su, by root
}

// neighbors is what placement looks up about the order of a graph
// (p4nodes/bk/NeighborhoodInformation.java).
type neighbors struct {
	index       map[*Node]int
	layer       map[*Node][]*Node
	left, right map[*Node][]neighbor
}

// neighbor is a node at the other end of an edge.
type neighbor struct {
	node *Node
	edge *Edge
}

func newAlignment(up, left bool) *alignment {
	return &alignment{
		up: up, left: left,
		root: map[*Node]*Node{}, next: map[*Node]*Node{},
		inner: map[*Node]float64{}, size: map[*Node]float64{},
		sink: map[*Node]*Node{}, shift: map[*Node]float64{},
		y: map[*Node]float64{}, used: map[*Node]bool{},
	}
}

// placeNodes gives each node its y, in four layouts and their balance
// (process, p4nodes/bk/BKNodePlacer.java line 148). This is Brandes and Köpf's
// algorithm with their x as y, as ELK's layers run left to right.
func placeNodes(g *Graph) {
	near := neighborsOf(g)
	marked := markConflicts(g, near)
	all := []*alignment{
		newAlignment(false, false), newAlignment(true, false),
		newAlignment(false, true), newAlignment(true, true),
	}
	for _, a := range all {
		align(g, a, near, marked)
		shiftInside(g, a)
	}
	for _, a := range all {
		compact(g, a, near)
	}
	chosen := choose(g, all)
	for _, n := range nodes(g) {
		n.Pos.Y = chosen.y[n] + chosen.inner[n]
	}
}

// layers gives g's layers in the order of the sweep.
func (a *alignment) layers(g *Graph) []*Layer {
	if a.left {
		return reversed(g.Layers)
	}
	return g.Layers
}

// order gives l's nodes in the order of the sweep.
func (a *alignment) order(l *Layer) []*Node {
	if a.up {
		return reversed(l.Nodes)
	}
	return l.Nodes
}

// block gives the nodes of n's block, around the ring from n.
func (a *alignment) block(n *Node) []*Node {
	all := []*Node{n}
	for current := a.next[n]; current != n; current = a.next[current] {
		all = append(all, current)
	}
	return all
}

func reversed[T any](all []T) []T {
	r := slices.Clone(all)
	slices.Reverse(r)
	return r
}

// neighborsOf looks up the order of g's layers and each node's neighbors, top
// to bottom (buildFor, p4nodes/bk/NeighborhoodInformation.java line 89, and
// lines 131 and 164). Edges only join neighboring layers at this point.
func neighborsOf(g *Graph) *neighbors {
	near := &neighbors{
		index: map[*Node]int{}, layer: map[*Node][]*Node{},
		left: map[*Node][]neighbor{}, right: map[*Node][]neighbor{},
	}
	for _, l := range g.Layers {
		for i, n := range l.Nodes {
			near.index[n], near.layer[n] = i, l.Nodes
		}
	}
	sorted := func(all []neighbor) []neighbor {
		slices.SortStableFunc(all, func(a, b neighbor) int {
			return cmp.Compare(near.index[a.node], near.index[b.node])
		})
		return all
	}
	for _, n := range nodes(g) {
		var left, right []neighbor
		for _, e := range incoming(n) {
			left = append(left, neighbor{e.Source.Owner, e})
		}
		for _, e := range outgoing(n) {
			right = append(right, neighbor{e.Target.Owner, e})
		}
		near.left[n], near.right[n] = sorted(left), sorted(right)
	}
	return near
}

func (near *neighbors) above(n *Node) *Node {
	if i := near.index[n]; i > 0 {
		return near.layer[n][i-1]
	}
	return nil
}

func (near *neighbors) below(n *Node) *Node {
	if i := near.index[n]; i < len(near.layer[n])-1 {
		return near.layer[n][i+1]
	}
	return nil
}

// markConflicts marks the edges crossing an inner segment, the middle of a long
// edge, which stays straight (markConflicts, p4nodes/bk/BKNodePlacer.java line
// 318).
func markConflicts(g *Graph, near *neighbors) map[*Edge]bool {
	marked := map[*Edge]bool{}
	for i := 2; i < len(g.Layers); i++ {
		layer := g.Layers[i].Nodes
		from, next := 0, 0
		for at, v := range layer {
			if at != len(layer)-1 && !innerSegment(v) {
				continue
			}
			to := len(g.Layers[i-1].Nodes) - 1
			if innerSegment(v) {
				to = near.index[near.left[v][0].node]
			}
			for ; next <= at; next++ {
				if innerSegment(layer[next]) {
					continue
				}
				for _, upper := range near.left[layer[next]] {
					if k := near.index[upper.node]; k < from || k > to {
						marked[upper.edge] = true
					}
				}
			}
			from = to
		}
	}
	return marked
}

// innerSegment tells whether an edge from a dummy node of a long edge comes to
// n, itself such a node (incidentToInnerSegment,
// p4nodes/bk/BKNodePlacer.java line 478).
func innerSegment(n *Node) bool {
	return n.Type == typeLongEdge && slices.ContainsFunc(incoming(n), func(e *Edge) bool {
		return e.Source.Owner.Type == typeLongEdge
	})
}

// choose takes the balance of the layouts or, should that overlap, the smallest
// sound layout, or else the first (process, p4nodes/bk/BKNodePlacer.java lines
// 239 to 267).
func choose(g *Graph, all []*alignment) *alignment {
	if balanced := balance(g, all); sound(g, balanced) {
		return balanced
	}
	var chosen *alignment
	for _, a := range all {
		if sound(g, a) && (chosen == nil || chosen.extent(g) > a.extent(g)) {
			chosen = a
		}
	}
	if chosen == nil {
		return all[0]
	}
	return chosen
}

// balance lines the layouts up with the narrowest, at the top those sweeping
// down and at the bottom those sweeping up, and puts each node between its
// middle two positions (createBalancedLayout,
// p4nodes/bk/BKNodePlacer.java line 400).
func balance(g *Graph, all []*alignment) *alignment {
	low, high, width := make([]float64, len(all)), make([]float64, len(all)), make([]float64, len(all))
	narrowest := 0
	for i, a := range all {
		low[i], high[i] = math.MaxInt32, math.MinInt32
		width[i] = a.extent(g)
		if width[narrowest] > width[i] {
			narrowest = i
		}
		for _, n := range nodes(g) {
			y := a.y[n] + a.inner[n]
			low[i] = min(low[i], y)
			high[i] = max(high[i], y+n.Size.Y)
		}
	}
	shift := make([]float64, len(all))
	for i, a := range all {
		shift[i] = low[narrowest] - low[i]
		if a.up {
			shift[i] = high[narrowest] - high[i]
		}
	}
	balanced := newAlignment(false, false)
	for _, n := range nodes(g) {
		ys := make([]float64, len(all))
		for i, a := range all {
			ys[i] = a.y[n] + a.inner[n] + shift[i]
		}
		slices.Sort(ys)
		balanced.y[n] = (ys[1] + ys[2]) / 2
	}
	return balanced
}

// sound tells whether a layout keeps each layer's nodes in order, apart
// (checkOrderConstraint, p4nodes/bk/BKNodePlacer.java line 577).
func sound(g *Graph, a *alignment) bool {
	for _, l := range g.Layers {
		position := math.Inf(-1)
		for _, n := range l.Nodes {
			y := a.y[n] + a.inner[n]
			if y-n.Margin.Top <= position {
				return false
			}
			position = y + n.Size.Y + n.Margin.Bottom
		}
	}
	return true
}

// extent is the height of a layout (layoutSize,
// p4nodes/bk/BKAlignedLayout.java line 117).
func (a *alignment) extent(g *Graph) float64 {
	low, high := math.Inf(1), math.Inf(-1)
	for _, n := range nodes(g) {
		low = min(low, a.y[n])
		high = max(high, a.y[n]+a.size[a.root[n]])
	}
	return high - low
}

// nodeSpacing is the room between neighbors in a layer (getVerticalSpacing,
// options/Spacings.java line 195, with the table from line 62).
func nodeSpacing(a, b *Node) float64 {
	if a.Type == typeLongEdge || b.Type == typeLongEdge || a.Type == typeLabel && b.Type == typeLabel {
		return edgeGap
	}
	return nodeGap
}

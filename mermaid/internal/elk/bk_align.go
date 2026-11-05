// SPDX-License-Identifier: EPL-2.0 OR GPL-3.0-or-later

package elk

import (
	"math"
	"slices"
)

// align joins nodes into blocks, each to a median neighbor on the side the
// sweep comes from, without crossing an alignment made before or using a marked
// edge (verticalAlignment, p4nodes/bk/BKAligner.java line 70).
func align(g *Graph, a *alignment, near *neighbors, marked map[*Edge]bool) {
	for _, n := range nodes(g) {
		a.root[n], a.next[n] = n, n
	}
	for _, l := range a.layers(g) {
		last := -1
		if a.up {
			last = math.MaxInt
		}
		for _, v := range a.order(l) {
			for _, u := range a.medians(near, v) {
				at := near.index[u.node]
				if a.next[v] != v || marked[u.edge] || a.up && last <= at || !a.up && last >= at {
					continue
				}
				a.next[u.node] = v
				a.root[v] = a.root[u.node]
				a.next[v] = a.root[v]
				last = at
			}
		}
	}
}

// medians gives the one or two middle neighbors of v on the side the sweep
// comes from, in the order the sweep tries them (BKAligner.java lines 117 to
// 121).
func (a *alignment) medians(near *neighbors, v *Node) []neighbor {
	all := near.left[v]
	if a.left {
		all = near.right[v]
	}
	if len(all) == 0 {
		return nil
	}
	middle := all[(len(all)+1)/2-1 : len(all)/2+1]
	if a.up {
		return reversed(middle)
	}
	return middle
}

// shiftInside moves the nodes of each block so that the edges inside it are
// straight, and sizes the block (insideBlockShift,
// p4nodes/bk/BKAligner.java line 173).
func shiftInside(g *Graph, a *alignment) {
	for _, root := range nodes(g) {
		if a.root[root] != root {
			continue
		}
		above, below := root.Margin.Top, root.Size.Y+root.Margin.Bottom
		a.inner[root] = 0
		for n := root; a.next[n] != root; n = a.next[n] {
			next := a.next[n]
			a.inner[next] = a.inner[n] + a.portOffset(edgeBetween(n, next))
			above = max(above, next.Margin.Top-a.inner[next])
			below = max(below, a.inner[next]+next.Size.Y+next.Margin.Bottom)
		}
		for _, n := range a.block(root) {
			a.inner[n] += above
		}
		a.size[root] = above + below
	}
}

// portOffset is how far the next node of a block lies from the one before for
// e to be straight (BKAligner.java lines 199 to 206).
func (a *alignment) portOffset(e *Edge) float64 {
	if a.left {
		return e.Target.Pos.Y - e.Source.Pos.Y
	}
	return e.Source.Pos.Y - e.Target.Pos.Y
}

// edgeBetween gives the first edge of from that touches to, port by port and
// each port's incoming edges first (getEdge, p4nodes/bk/BKNodePlacer.java line
// 501, graph/LPort.java line 76). With several edges between two nodes, it need
// not be the edge that made them a block.
func edgeBetween(from, to *Node) *Edge {
	for _, p := range from.Ports {
		for _, e := range slices.Concat(p.In, p.Out) {
			if e.Target.Owner == to || e.Source.Owner == to {
				return e
			}
		}
	}
	return nil
}

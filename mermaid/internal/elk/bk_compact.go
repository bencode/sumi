// SPDX-License-Identifier: EPL-2.0 OR GPL-3.0-or-later

package elk

import "math"

// compaction is the placing of one layout's blocks
// (p4nodes/bk/BKCompactor.java).
type compaction struct {
	a       *alignment
	near    *neighbors
	classes map[*Node]*class // by sink
	line    *straightener
}

// class is a group of blocks placed relative to each other, known by its sink.
// Classes are placed last, by the room they need from each other (ClassNode,
// BKCompactor.java line 396).
type class struct {
	sink     *Node
	shift    float64
	placed   bool
	outgoing []classEdge
	indegree int
}

// classEdge is the room a class needs from another.
type classEdge struct {
	target     *class
	separation float64
}

// compact gives each block its position, top to bottom when sweeping down and
// the other way when sweeping up (horizontalCompaction,
// p4nodes/bk/BKCompactor.java line 88).
func compact(g *Graph, a *alignment, near *neighbors) {
	unset := math.Inf(1)
	if a.up {
		unset = math.Inf(-1)
	}
	for _, n := range nodes(g) {
		a.sink[n], a.shift[n] = n, unset
	}
	c := &compaction{a: a, near: near, classes: map[*Node]*class{}, line: newStraightener(a, near)}
	for _, l := range a.layers(g) {
		for _, v := range a.order(l) {
			if a.root[v] == v {
				c.placeBlock(v)
			}
		}
	}
	c.placeClasses()
	for _, l := range a.layers(g) {
		for _, v := range l.Nodes {
			a.y[v] = a.y[a.root[v]]
			if shift := a.shift[a.sink[v]]; a.root[v] == v && shift != unset {
				a.y[v] += shift
			}
		}
	}
	c.line.finish()
}

// placeBlock places root's block as close as it can to what lies before it,
// which it places first (placeBlock, p4nodes/bk/BKCompactor.java line 167).
func (c *compaction) placeBlock(root *Node) {
	a := c.a
	if _, placed := a.y[root]; placed {
		return
	}
	a.y[root] = 0
	first := true
	thresh := math.Inf(-1)
	if a.up {
		thresh = math.Inf(1)
	}
	for _, n := range a.block(root) {
		neighbor := c.near.above(n)
		if a.up {
			neighbor = c.near.below(n)
		}
		if neighbor != nil {
			c.placeBlock(a.root[neighbor])
		}
		// After the neighbor, as this may queue the block for later, and the order
		// of the queue matters.
		thresh = c.line.threshold(thresh, root, n)
		if neighbor != nil && c.keepClear(root, n, neighbor, thresh, first) {
			first = false
		}
	}
	c.line.finished[root] = true
}

// keepClear keeps n, of root's block, off its neighbor: in the same class by
// moving the block, which it tells by returning true, and otherwise by asking
// that of the classes (BKCompactor.java lines 217 to 322).
func (c *compaction) keepClear(root, n, neighbor *Node, thresh float64, first bool) bool {
	a := c.a
	other := a.root[neighbor]
	if a.sink[root] == root {
		a.sink[root] = a.sink[other]
	}
	if a.sink[root] != a.sink[other] {
		from, to := c.classOf(a.sink[root]), c.classOf(a.sink[other])
		to.indegree++
		from.outgoing = append(from.outgoing, classEdge{to, c.separation(root, n, neighbor)})
		return false
	}
	spacing := nodeSpacing(n, neighbor)
	if a.up {
		position := a.y[other] + a.inner[neighbor] - neighbor.Margin.Top - spacing -
			n.Margin.Bottom - n.Size.Y - a.inner[n]
		position = min(position, thresh)
		if !first {
			position = min(a.y[root], position)
		}
		a.y[root] = position
		return true
	}
	position := a.y[other] + a.inner[neighbor] + neighbor.Size.Y + neighbor.Margin.Bottom +
		spacing + n.Margin.Top - a.inner[n]
	position = max(position, thresh)
	if !first {
		position = max(a.y[root], position)
	}
	a.y[root] = position
	return true
}

// separation is the room n's class needs from its neighbor's, relative to their
// positions so far (BKCompactor.java lines 283 to 314).
func (c *compaction) separation(root, n, neighbor *Node) float64 {
	a := c.a
	other := a.root[neighbor]
	if a.up {
		return a.y[root] + a.inner[n] + n.Size.Y + n.Margin.Bottom + nodeGap -
			(a.y[other] + a.inner[neighbor] - neighbor.Margin.Top)
	}
	return a.y[root] + a.inner[n] - n.Margin.Top - a.y[other] - a.inner[neighbor] -
		neighbor.Size.Y - neighbor.Margin.Bottom - nodeGap
}

func (c *compaction) classOf(sink *Node) *class {
	if c.classes[sink] == nil {
		c.classes[sink] = &class{sink: sink}
	}
	return c.classes[sink]
}

// placeClasses shifts each class from the classes before it, in the order of
// the class graph (placeClasses, p4nodes/bk/BKCompactor.java line 337). A class
// takes the min or max over all before it, so where the walk starts does not
// matter.
func (c *compaction) placeClasses() {
	var queue []*class
	for _, n := range c.classes {
		if n.indegree == 0 {
			queue = append(queue, n)
		}
	}
	for len(queue) > 0 {
		n := queue[0]
		queue = queue[1:]
		for _, e := range n.outgoing {
			shift := n.shift + e.separation
			switch {
			case !e.target.placed:
				e.target.shift, e.target.placed = shift, true
			case c.a.up:
				e.target.shift = max(e.target.shift, shift)
			default:
				e.target.shift = min(e.target.shift, shift)
			}
			if e.target.indegree--; e.target.indegree == 0 {
				queue = append(queue, e.target)
			}
		}
	}
	for _, n := range c.classes {
		c.a.shift[n.sink] = n.shift
	}
}

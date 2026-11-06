// SPDX-License-Identifier: EPL-2.0 OR GPL-3.0-or-later

package elk

import (
	"math"
	"slices"
)

// straightener tries to make one more edge of each block straight, at the first
// or the last node of the block (SimpleThresholdStrategy,
// p4nodes/bk/ThresholdStrategy.java).
type straightener struct {
	a        *alignment
	near     *neighbors
	finished map[*Node]bool // blocks placed, by root
	queue    []*pending
}

// pending is an end of a block, its edge chosen later if at all
// (Postprocessable, ThresholdStrategy.java line 415).
type pending struct {
	free     *Node
	root     bool // free is the first node of its block, not the last
	hasEdges bool
	edge     *Edge
}

func newStraightener(a *alignment, near *neighbors) *straightener {
	return &straightener{a: a, near: near, finished: map[*Node]bool{}}
}

// threshold is the limit on root's position that makes an edge at n, either end
// of the block, straight (calculateThreshold,
// p4nodes/bk/ThresholdStrategy.java line 170, whose two directions are the
// same).
func (s *straightener) threshold(old float64, root, n *Node) float64 {
	isRoot, isLast := root == n, s.a.next[n] == root
	if !isRoot && !isLast {
		return old
	}
	t := old
	if isRoot {
		t = s.bound(root, true)
	}
	if math.IsInf(t, 0) && isLast {
		t = s.bound(n, false)
	}
	return t
}

// takesIncoming tells whether p looks at its incoming edges and not its outgoing
// ones: it looks at those on the outside of the block.
func (s *straightener) takesIncoming(p *pending) bool {
	return p.root != s.a.left
}

// pick chooses the first edge of p to a placed block, unless p's block has an
// edge made straight already (pickEdge, p4nodes/bk/ThresholdStrategy.java line
// 227).
func (s *straightener) pick(p *pending) {
	p.edge, p.hasEdges = nil, false
	if s.a.used[s.a.root[p.free]] {
		return
	}
	all, other := outgoing(p.free), func(e *Edge) *Node { return e.Target.Owner }
	if s.takesIncoming(p) {
		all, other = incoming(p.free), func(e *Edge) *Node { return e.Source.Owner }
	}
	for _, e := range all {
		p.hasEdges = true
		if s.finished[s.a.root[other(e)]] {
			p.edge = e
			return
		}
	}
}

// bound gives the position making an edge at n straight and marks the blocks at
// both its ends as used, or else gives no limit, queueing n when its edge may
// come later (getBound, p4nodes/bk/ThresholdStrategy.java line 277).
func (s *straightener) bound(n *Node, isRoot bool) float64 {
	a := s.a
	p := &pending{free: n, root: isRoot}
	s.pick(p)
	if p.edge == nil {
		if p.hasEdges {
			s.queue = append(s.queue, p)
		}
		if a.up {
			return math.Inf(1)
		}
		return math.Inf(-1)
	}
	own, other := p.edge.Source, p.edge.Target
	if s.takesIncoming(p) {
		own, other = other, own
	}
	threshold := a.y[a.root[other.Owner]] + a.inner[other.Owner] + other.Pos.Y -
		a.inner[own.Owner] - own.Pos.Y
	a.used[a.root[p.edge.Source.Owner]] = true
	a.used[a.root[p.edge.Target.Owner]] = true
	return threshold
}

// finish moves the queued blocks, in order and then back from the last that did
// not move (postProcess, p4nodes/bk/ThresholdStrategy.java line 336).
func (s *straightener) finish() {
	var stuck []*pending
	for _, p := range s.queue {
		s.pick(p)
		if p.edge != nil && !s.move(p) {
			stuck = append(stuck, p)
		}
	}
	for _, p := range slices.Backward(stuck) {
		s.move(p)
	}
}

// move shifts p's block as far towards a straight edge as there is room, and
// tells whether it did (process, p4nodes/bk/ThresholdStrategy.java line 372).
func (s *straightener) move(p *pending) bool {
	a := s.a
	fix, block := p.edge.Source, p.edge.Target
	if p.edge.Source.Owner == p.free {
		fix, block = block, fix
	}
	delta := a.portY(block) - a.portY(fix)
	var room float64
	switch {
	case delta > 0:
		room = a.spaceAbove(block.Owner, delta, s.near)
		a.shiftBlock(block.Owner, -room)
	case delta < 0:
		room = a.spaceBelow(block.Owner, -delta, s.near)
		a.shiftBlock(block.Owner, room)
	}
	return room > 0
}

// portY is where a port is in the layout (calculateDelta,
// p4nodes/bk/BKAlignedLayout.java line 147).
func (a *alignment) portY(p *Port) float64 {
	return a.y[p.Owner] + a.inner[p.Owner] + p.Pos.Y
}

// shiftBlock moves the block of n (shiftBlock,
// p4nodes/bk/BKAlignedLayout.java line 164).
func (a *alignment) shiftBlock(n *Node, delta float64) {
	for _, current := range a.block(n) {
		a.y[current] += delta
	}
}

// spaceAbove is how far n's block can move up, at most delta (checkSpaceAbove,
// p4nodes/bk/BKAlignedLayout.java line 184).
func (a *alignment) spaceAbove(n *Node, delta float64, near *neighbors) float64 {
	room := delta
	for _, current := range a.block(n) {
		if above := near.above(current); above != nil {
			room = min(room, a.top(current)-(a.bottom(above)+nodeSpacing(near.settings, current, above)))
		}
	}
	return room
}

// spaceBelow is how far n's block can move down, at most delta (checkSpaceBelow,
// p4nodes/bk/BKAlignedLayout.java line 221).
func (a *alignment) spaceBelow(n *Node, delta float64, near *neighbors) float64 {
	room := delta
	for _, current := range a.block(n) {
		if below := near.below(current); below != nil {
			room = min(room, a.top(below)-(a.bottom(current)+nodeSpacing(near.settings, current, below)))
		}
	}
	return room
}

// top is where n and its margin start (getMinY,
// p4nodes/bk/BKAlignedLayout.java line 256).
func (a *alignment) top(n *Node) float64 {
	return a.y[a.root[n]] + a.inner[n] - n.Margin.Top
}

// bottom is where n and its margin end (getMaxY,
// p4nodes/bk/BKAlignedLayout.java line 274).
func (a *alignment) bottom(n *Node) float64 {
	return a.y[a.root[n]] + a.inner[n] + n.Size.Y + n.Margin.Bottom
}

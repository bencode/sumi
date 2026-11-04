package sequence

import (
	"cmp"
	"fmt"
	"maps"
	"slices"

	"github.com/bencode/sumi/scene"
)

// A bar is an activation bar still open: where it stands and where it
// starts.
type bar struct {
	x, y       float64
	order      int
	id, parent string
}

// openBar opens a bar on id, two pixels below the cursor, over the bars
// open on it (newActivation, sequenceRenderer.ts). Mermaid stacks the bar
// where it opens, so it takes its order now. The state's map is copied,
// never changed.
func openBar(s state, id string, e env) state {
	s.n.bar++
	x := barX(id, len(s.bars[id]), e)
	b := bar{x: x, y: s.y + 2, order: s.order, id: fmt.Sprintf("activation-%d", s.n.bar), parent: e.parent}
	s.order++
	s.bars = withBars(s.bars, id, append(s.bars[id][:len(s.bars[id]):len(s.bars[id])], b))
	return s
}

// closeBar closes id's last bar at the cursor. A bar shorter than 18 is
// drawn from 6 above the cursor to 12 below it, without moving the cursor
// (activeEnd). The room it takes then reaches the bars still open.
func closeBar(s state, id string, c config) (*pBar, state) {
	open := s.bars[id]
	b := open[len(open)-1]
	s.bars = withBars(s.bars, id, open[:len(open)-1])
	top, bottom := b.y, s.y
	if top+18 > bottom {
		top, bottom = bottom-6, bottom+12
	}
	p := &pBar{
		id: b.id, parent: b.parent, of: id, order: b.order,
		box: scene.Rect{X: b.x, Y: top, W: c.activationWidth, H: bottom - top},
		ext: between(b.x, bottom-10, b.x+c.activationWidth, bottom),
	}
	return p, reach(s, p.ext.Y, c)
}

// reach takes the tops of the open bars up to the top of a room Mermaid's
// bounds take in, as updateBounds (sequenceRenderer.ts lines 106-130) does:
// the k-th bar open, counting from 0 in the order they opened, starts no
// lower than top + k*boxMargin. In practice only the first one moves.
func reach(s state, top float64, c config) state {
	var all []bar
	for _, open := range s.bars {
		all = append(all, open...)
	}
	slices.SortFunc(all, func(a, b bar) int { return cmp.Compare(a.order, b.order) })
	rank := map[string]int{}
	for k, b := range all {
		rank[b.id] = k
	}
	for id, open := range s.bars {
		var moved []bar
		for i, b := range open {
			if y := top + float64(rank[b.id])*c.boxMargin; y < b.y {
				if moved == nil {
					moved = slices.Clone(open)
				}
				moved[i].y = y
			}
		}
		if moved != nil {
			s.bars = withBars(s.bars, id, moved)
		}
	}
	return s
}

func withBars(m map[string][]bar, id string, open []bar) map[string][]bar {
	m = maps.Clone(m)
	if m == nil {
		m = map[string][]bar{}
	}
	m[id] = open
	return m
}

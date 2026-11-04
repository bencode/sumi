package sequence

import (
	"fmt"

	"github.com/bencode/sumi/scene"
)

// env is what placing reads and never changes.
type env struct {
	c      config
	m      scene.Measurer
	ps     []*Participant
	at     map[string]int // each participant's index
	w, x   []float64      // the participants' widths and centers
	across across
	parent string // the id of the innermost block
}

// state is what placing carries from one statement to the next: the
// cursor, the order Mermaid stacks things in, the open activation bars,
// and how many of each kind of thing have ids.
type state struct {
	y     float64
	order int
	bars  map[string][]bar
	n     struct{ msg, note, block, rect, bar int }
}

// A placed thing has its shapes' places, and the room Mermaid's bounds
// count it as taking, which the blocks around it grow to hold.
type placed interface{ extent() scene.Rect }

type pMessage struct {
	*Message
	id, parent string
	model      msgModel
	top, y     float64 // where its room starts, and its line's y
	text       textBlock
	ext        scene.Rect
}

type pNote struct {
	*Note
	id, parent string
	order      int
	box        scene.Rect
	text       textBlock
}

type pBar struct {
	id, parent, of string
	order          int
	box, ext       scene.Rect
}

type pBlock struct {
	*Block
	id, parent string
	order      int
	box        scene.Rect
	lines      []float64   // each section's top: the block's, then the dashed lines
	titles     []textBlock // each section's title, wrapped and in brackets
}

type pRect struct {
	*Rect
	id, parent string
	box        scene.Rect
}

func (p *pMessage) extent() scene.Rect { return p.ext }
func (p *pNote) extent() scene.Rect    { return p.box }
func (p *pBar) extent() scene.Rect     { return p.ext }
func (p *pBlock) extent() scene.Rect   { return p.box }
func (p *pRect) extent() scene.Rect    { return p.box }

// place places body's statements one after another from the cursor, as
// Mermaid's main loop does (draw, sequenceRenderer.ts lines 1143-1381).
func place(body []Stmt, s state, e env) ([]placed, state) {
	var out []placed
	for _, st := range body {
		var ps []placed
		switch st := st.(type) {
		case *Message:
			ps, s = placeMessage(st, s, e)
		case *Note:
			ps, s = placeNote(st, s, e)
		case *Block:
			ps, s = placeBlock(st, s, e)
		case *Rect:
			ps, s = placeRect(st, s, e)
		case *Activation:
			if st.On {
				s = openBar(s, st.ID, e)
				continue
			}
			var b *pBar
			b, s = closeBar(s, st.ID, e.c)
			ps = []placed{b}
		}
		out = append(out, ps...)
	}
	return out, s
}

// placeMessage follows boundMessage (lines 397-446): ten above, a line of
// text, then the rest of the text, the line below it.
func placeMessage(msg *Message, s state, e env) ([]placed, state) {
	s.n.msg++
	p := &pMessage{Message: msg, id: fmt.Sprintf("msg-%d", s.n.msg), parent: e.parent,
		model: e.across.msgs[msg], top: s.y, text: measure(e.m, msg.Text, e.c.messageFont)}
	mm := p.model
	lh := p.text.h() / float64(max(len(p.text.lines), 1))
	y := s.y + 10 + lh
	off := p.text.h() - 10 + e.c.boxMargin
	p.y = y + off
	if msg.From == msg.To {
		off += 30
		dx := max(p.text.w/2, e.c.width/2)
		p.ext = between(mm.startx-dx, y-10+off, mm.stopx+dx, y+30+off)
	} else {
		p.ext = between(mm.startx, p.y-10, mm.stopx, p.y)
	}
	s.y = y + off
	p.ext = union(p.ext, between(mm.from, p.top, mm.to, p.top+lh+off))
	s = reach(s, p.ext.Y, e.c)
	out := []placed{p}
	if msg.Activate {
		s = openBar(s, msg.To, e)
	}
	// A central connection opens a bar that only a deactivate closes
	// (sequenceRenderer.ts lines 1158-1162): on To first, then on From.
	if msg.CircleTo {
		s = openBar(s, msg.To, e)
	}
	if msg.CircleFrom {
		s = openBar(s, msg.From, e)
	}
	if msg.Deactivate {
		var b *pBar
		b, s = closeBar(s, msg.From, e.c)
		out = append(out, b)
	}
	return out, s
}

// placeNote follows drawNote (lines 241-283).
func placeNote(n *Note, s state, e env) ([]placed, state) {
	s.n.note++
	s.y += e.c.boxMargin
	x, w := noteBox(n, e)
	tb := measure(e.m, n.Text, e.c.noteFont)
	h := tb.h() + 2*e.c.noteMargin
	p := &pNote{Note: n, id: fmt.Sprintf("note-%d", s.n.note), parent: e.parent,
		order: s.order, box: scene.Rect{X: x, Y: s.y, W: w, H: h}, text: tb}
	s.order++
	s.y += h
	return []placed{p}, reach(s, p.box.Y, e.c)
}

// placeBlock follows adjustLoopHeightForWrap (lines 916-934) for each
// section's head, and the block's end (lines 1176-1179): the frame holds
// what is inside it with boxMargin to spare, and the cursor moves to its
// bottom. Mermaid draws the frame at the end, over what is inside.
func placeBlock(b *Block, s state, e env) ([]placed, state) {
	s.n.block++
	p := &pBlock{Block: b, id: fmt.Sprintf("block-%d", s.n.block), parent: e.parent}
	inner := e
	inner.parent = p.id
	var kids []placed
	for i, sec := range b.Sections {
		pre, post := e.c.boxMargin, e.c.boxMargin+e.c.boxTextMargin
		if i > 0 {
			pre, post = post, pre
		}
		s.y += pre
		p.lines = append(p.lines, s.y)
		var tb textBlock
		if sec.Label != "" {
			label := wrapLabel(e.m, "["+sec.Label+"]", e.across.widths[b]-2*e.c.wrapPadding, e.c.messageFont)
			tb = measure(e.m, label, e.c.messageFont)
			post += max(tb.h(), e.c.labelBoxHeight)
		}
		p.titles = append(p.titles, tb)
		s.y += post
		var ps []placed
		ps, s = place(sec.Body, s, inner)
		kids = append(kids, ps...)
	}
	p.box = frame(kids, p.lines[0], s.y, e)
	s.y = p.box.Y + p.box.H
	p.order = s.order
	s.order++
	return append(kids, p), s
}

// placeRect follows RECT_START and RECT_END (lines 1182-1202): boxMargin
// above the rect's top and boxMargin below it, no title, and the frame
// holds what is inside it as a block's does.
func placeRect(r *Rect, s state, e env) ([]placed, state) {
	s.n.rect++
	p := &pRect{Rect: r, id: fmt.Sprintf("rect-%d", s.n.rect), parent: e.parent}
	inner := e
	inner.parent = p.id
	s.y += e.c.boxMargin
	top := s.y
	s.y += e.c.boxMargin
	kids, s := place(r.Body, s, inner)
	p.box = frame(kids, top, s.y, e)
	s.y = p.box.Y + p.box.H
	return append(kids, p), s
}

// frame is a block's frame: what is inside it, boxMargin further out on
// every side but the top, which is where the block starts. A block with
// nothing inside spans every participant; with no participants, where
// Mermaid's frame is NaN wide, it has no width.
func frame(kids []placed, top, y float64, e env) scene.Rect {
	var u scene.Rect
	for _, k := range kids {
		u = union(u, k.extent())
	}
	m := e.c.boxMargin
	if u == (scene.Rect{}) {
		if len(e.x) == 0 {
			return between(0, top, 0, y+m)
		}
		last := len(e.x) - 1
		return between(e.x[0]-e.w[0]/2, top, e.x[last]+e.w[last]/2, y+m)
	}
	return between(u.X-m, min(top, u.Y-m), u.X+u.W+m, u.Y+u.H+m)
}

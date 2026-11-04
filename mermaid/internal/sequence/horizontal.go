package sequence

import "math"

// gaps are the distances between the centers of neighboring participants,
// as Mermaid sets them (getMaxMessageWidthPerActor and calculateActorMargins,
// sequenceRenderer.ts lines 1488-1666): each gap is wide enough for the two
// boxes and actorMargin between them, and for the widest message or note
// that falls in it, plus actorMargin. Only messages between neighbors, and
// notes, count; a message that passes over a participant does not.
func gaps(body []Stmt, e env) []float64 {
	need := make([]float64, len(e.ps)) // need[i]: what falls between i and i+1
	grow := func(i int, width float64) {
		if i >= 0 && i < len(need) {
			need[i] = max(need[i], width)
		}
	}
	walk(body, func(s Stmt) {
		switch s := s.(type) {
		case *Message:
			from, to := e.at[s.From], e.at[s.To]
			wd := measure(e.m, s.Text, e.c.messageFont).w + 2*e.c.wrapPadding
			switch from {
			case to:
				grow(from, wd/2)
			case to + 1:
				grow(to, wd)
			case to - 1:
				grow(from, wd)
			}
		case *Note:
			wd := measure(e.m, s.Text, e.c.noteFont).w + 2*e.c.wrapPadding
			noteGaps(s, e, wd, grow)
		}
	})
	g := make([]float64, max(len(e.ps)-1, 0))
	for i := range g {
		g[i] = max(e.w[i]/2+e.w[i+1]/2+e.c.actorMargin, need[i]+e.c.actorMargin)
	}
	return g
}

// noteGaps follows the note branches of getMaxMessageWidthPerActor, which
// go by the note's last participant: a note left of the first participant,
// or right of the last, takes no gap.
func noteGaps(n *Note, e env, wd float64, grow func(int, float64)) {
	from, to := e.at[n.Of[0]], e.at[n.Of[len(n.Of)-1]]
	last := len(e.ps) - 1
	switch n.Place {
	case RightOf:
		if to < last {
			grow(from, wd)
		}
	case LeftOf:
		if to > 0 {
			grow(to-1, wd)
		}
	case Over:
		if to > 0 {
			grow(to-1, wd/2)
		}
		if to < last {
			grow(from, wd/2)
		}
	}
}

// centers are the participants' centers, the first box's left edge at 0.
func centers(w, gaps []float64) []float64 {
	c := make([]float64, len(w))
	for i := range w {
		if i == 0 {
			c[i] = w[0] / 2
			continue
		}
		c[i] = c[i-1] + gaps[i-1]
	}
	return c
}

// across is the horizontal geometry Mermaid works out before it places
// anything down the page (calculateLoopBounds, sequenceRenderer.ts): where
// each message's line runs, and how wide each block wraps its titles. It
// keeps its own activation bars, which only activate, deactivate, + and -
// open and close; where the bars are drawn is worked out down the page,
// where central connections open bars too.
type across struct {
	msgs   map[*Message]msgModel
	widths map[*Block]float64
}

// A msgModel is buildMessageModel's result.
type msgModel struct {
	startx, stopx float64 // the line's ends, shortened for heads and bars
	from, to      float64 // the outer edges of both ends' lifelines or bars
	width         float64
}

// A blockSpan is a block's entry in calculateLoopBounds, which every
// message and note inside it updates, deepest or not.
type blockSpan struct{ from, to, width float64 }

const maxSafe = 1<<53 - 1 // JavaScript's Number.MAX_SAFE_INTEGER

func measureAcross(body []Stmt, e env) across {
	a := across{msgs: map[*Message]msgModel{}, widths: map[*Block]float64{}}
	open := map[string][]float64{} // each participant's open bars, by left edge
	var spans []*blockSpan         // the blocks around the statement
	var visit func([]Stmt)
	visit = func(body []Stmt) {
		for _, s := range body {
			switch s := s.(type) {
			case *Block:
				sp := &blockSpan{from: maxSafe, to: -maxSafe}
				spans = append(spans, sp)
				for _, sec := range s.Sections {
					visit(sec.Body)
				}
				spans = spans[:len(spans)-1]
				a.widths[s] = sp.width
			case *Note:
				x, w := noteBox(s, e)
				for _, sp := range spans {
					sp.from, sp.to = min(sp.from, x), max(sp.to, x+w)
					sp.width = max(sp.width, math.Abs(sp.from-sp.to)) - e.c.labelBoxWidth
				}
			case *Message:
				mm := message(s, open, e)
				a.msgs[s] = mm
				for _, sp := range spans {
					sp.grow(s, mm, e)
				}
				if s.Activate {
					openAt(open, s.To, e)
				}
				if s.Deactivate {
					open[s.From] = open[s.From][:len(open[s.From])-1]
				}
			case *Activation:
				if s.On {
					openAt(open, s.ID, e)
				} else {
					open[s.ID] = open[s.ID][:len(open[s.ID])-1]
				}
			}
		}
	}
	visit(body)
	return a
}

func (sp *blockSpan) grow(s *Message, mm msgModel, e env) {
	if mm.startx == 0 || mm.stopx == 0 {
		return
	}
	if mm.startx != mm.stopx {
		sp.from, sp.to = min(sp.from, mm.startx), max(sp.to, mm.stopx)
		sp.width = max(sp.width, mm.width) - e.c.labelBoxWidth
		return
	}
	// A message to itself, measured from the box's left edge, as Mermaid
	// measures it.
	i, j := e.at[s.From], e.at[s.To]
	from, to := e.x[i]-e.w[i]/2, e.x[j]-e.w[j]/2
	sp.from = min(from-mm.width/2, from-e.w[i]/2, sp.from)
	sp.to = max(to+mm.width/2, to+e.w[i]/2, sp.to)
	sp.width = max(sp.width, math.Abs(sp.to-sp.from)) - e.c.labelBoxWidth
}

// openAt opens a bar on id at barX.
func openAt(open map[string][]float64, id string, e env) {
	open[id] = append(open[id], barX(id, len(open[id]), e))
}

// barX is the left edge of a bar opened on id over n others, half a bar's
// width right of the one below it, the first half a width left of the
// lifeline (newActivation).
func barX(id string, n int, e env) float64 {
	return e.x[e.at[id]] + float64(n-1)*e.c.activationWidth/2
}

// edges are where a participant's open bars, or its lifeline, start and end
// (activationBounds).
func edges(open map[string][]float64, id string, e env) (float64, float64) {
	c := e.x[e.at[id]]
	l, r := c-1, c+1
	for _, x := range open[id] {
		l, r = min(l, x), max(r, x+e.c.activationWidth)
	}
	return l, r
}

// message is buildMessageModel (sequenceRenderer.ts lines 1866-2040) for
// the arrows sumi supports.
func message(s *Message, open map[string][]float64, e env) msgModel {
	fl, fr := edges(open, s.From, e)
	tl, tr := edges(open, s.To, e)
	right := fl <= tl
	adjust := func(v float64) float64 {
		if right {
			return -v
		}
		return v
	}
	mm := msgModel{startx: fl, stopx: tr, from: min(fl, fr, tl, tr), to: max(fl, fr, tl, tr)}
	if right {
		mm.startx, mm.stopx = fr, tl
	}
	if s.CircleFrom { // calculateCentralConnectionOffset
		mm.startx += 4
		if s.Both && !right {
			mm.startx -= 6
		}
	}
	if s.From == s.To {
		mm.stopx = mm.startx
	} else {
		if (s.Activate || s.CircleTo) && math.Abs(tl-tr) <= 2 {
			mm.stopx += adjust(e.c.activationWidth/2 - 1)
		}
		if s.Head != NoHead {
			mm.stopx += adjust(3)
		}
		if s.Both {
			mm.startx -= adjust(3)
		}
	}
	text := measure(e.m, s.Text, e.c.messageFont).w + 2*e.c.wrapPadding
	mm.width = max(text, math.Abs(mm.startx-mm.stopx)+2*e.c.wrapPadding, e.c.width)
	return mm
}

// noteBox is where a note's box starts and how wide it is (buildNoteModel).
func noteBox(n *Note, e env) (x, w float64) {
	i, j := e.at[n.Of[0]], e.at[n.Of[len(n.Of)-1]]
	left, wi, wj := e.x[i]-e.w[i]/2, e.w[i], e.w[j]
	text := measure(e.m, n.Text, e.c.noteFont).w + 2*e.c.noteMargin
	switch {
	case n.Place == RightOf:
		w = max(wi/2+wj/2, text)
		return left + (wi+e.c.actorMargin)/2, w
	case n.Place == LeftOf:
		w = max(wi/2+wj/2, text)
		return left - w + (wi-e.c.actorMargin)/2, w
	case i == j:
		w = max(wi, e.c.width, text)
		return left + (wi-w)/2, w
	}
	w = math.Abs(e.x[i]-e.x[j]) + e.c.actorMargin
	return min(e.x[i], e.x[j]) - e.c.actorMargin/2, w
}

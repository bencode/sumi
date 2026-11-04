package sequence

import "github.com/bencode/sumi/scene"

// A layout is a diagram with everything in it placed, ready to draw.
type layout struct {
	d      *Diagram
	e      env
	items  []placed
	boxes  []pBox
	top    float64    // the top of the participants at the top
	bottom float64    // the top of the participants drawn again at the foot
	bounds scene.Rect // what Mermaid's bounds hold, which the title goes by
}

// Layout lays out a sequence diagram's body, its front matter already
// taken off, with title the front matter's title, if any.
func Layout(body, title string, m scene.Measurer) (*scene.Scene, error) {
	ts, err := Lex(body)
	if err != nil {
		return nil, err
	}
	d, err := Parse(ts)
	if err != nil {
		return nil, err
	}
	d, err = Resolve(d, title)
	if err != nil {
		return nil, err
	}
	return draw(arrange(d, m)), nil
}

// arrange places everything in d: across first, the boxes widening the
// gaps around them; then down the page, the participants at the top, below
// the boxes' titles, the statements below them, the participants again
// boxMargin twice below the last (drawActors), and the boxes down to under
// those.
func arrange(d *Diagram, m scene.Measurer) layout {
	e := env{c: defaults, m: m, ps: d.Participants, at: map[string]int{}}
	for i, p := range d.Participants {
		e.at[p.ID] = i
	}
	e.labels = labels(d.Participants, m, e.c)
	e.c.height = actorHeight(d.Participants, e.labels, m, e.c)
	e.w = widths(d.Participants, e.labels, m, e.c)
	need := needs(d.Body, e)
	fits := boxFits(d, need, e)
	lead, g := spread(d.Participants, gaps(need, e), fits, e.c)
	e.x = centers(e.w, g, lead)
	e.across = measureAcross(d.Body, e)
	top := boxTop(fits, e)
	items, s := place(d.Body, state{y: top + e.c.height}, e)
	l := layout{d: d, e: e, items: items, top: top, bottom: s.y + e.c.footerGap}
	l.boxes = placeBoxes(l, fits)
	for _, b := range l.boxes {
		l.bounds = union(l.bounds, b.model)
	}
	for i := range d.Participants {
		head := scene.Rect{X: e.x[i] - e.w[i]/2, Y: 0, W: e.w[i], H: e.c.height}
		foot := head
		foot.Y = l.bottom
		l.bounds = union(union(l.bounds, head), foot)
	}
	for _, p := range items {
		l.bounds = union(l.bounds, p.extent())
	}
	return l
}

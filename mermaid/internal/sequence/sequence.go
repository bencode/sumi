package sequence

import "github.com/bencode/sumi/scene"

// A layout is a diagram with everything in it placed, ready to draw.
type layout struct {
	d      *Diagram
	e      env
	items  []placed
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

// arrange places everything in d: across first, then down the page, the
// participants at the top, the statements below them, and the
// participants again boxMargin twice below the last (drawActors).
func arrange(d *Diagram, m scene.Measurer) layout {
	e := env{c: defaults, m: m, ps: d.Participants, at: map[string]int{}}
	for i, p := range d.Participants {
		e.at[p.ID] = i
	}
	e.w = widths(d.Participants, m, e.c)
	e.x = centers(e.w, gaps(d.Body, e))
	e.across = measureAcross(d.Body, e)
	items, s := place(d.Body, state{y: e.c.height}, e)
	l := layout{d: d, e: e, items: items, bottom: s.y + e.c.footerGap}
	for i := range d.Participants {
		top := scene.Rect{X: e.x[i] - e.w[i]/2, Y: 0, W: e.w[i], H: e.c.height}
		foot := top
		foot.Y = l.bottom
		l.bounds = union(union(l.bounds, top), foot)
	}
	for _, p := range items {
		l.bounds = union(l.bounds, p.extent())
	}
	return l
}

package sequence

import "github.com/bencode/sumi/scene"

// A look is how participants of one shape draw (the drawActorType
// functions, svgDraw.js): where the lifeline starts below the top, how far
// left of the middle the label is, and the glyph, which gives the shape's
// paths and its label's middle.
type look struct {
	lifeline func(c config) float64
	dx       float64
	glyph    func(g slot) ([]scene.Path, float64)
}

// A slot is where a glyph draws: the participant's left edge, middle and
// top, its width and height, and whether it is the one at the foot.
type slot struct {
	x, c, y, w, h float64
	foot          bool
}

var looks = map[Shape]look{
	Box:         {lifeline: underBox, glyph: box},
	Figure:      {lifeline: fixed(80), glyph: figure},
	Collections: {lifeline: underBox, dx: 6, glyph: collections},
	Queue:       {lifeline: underBox, glyph: queue},
	Database:    {lifeline: underDatabase, glyph: database},
	Control:     {lifeline: fixed(75), glyph: control},
	Entity:      {lifeline: fixed(75), glyph: entity},
	Boundary:    {lifeline: fixed(80), glyph: boundary},
}

// A lifeline starts at the participant's height below its top, a
// database's 2*boxTextMargin further down; under the drawn figures, at a
// fixed depth.
func underBox(c config) float64      { return c.height }
func underDatabase(c config) float64 { return c.height + 2*c.boxTextMargin }
func fixed(y float64) func(config) float64 {
	return func(config) float64 { return y }
}

// lifeline runs from under the top participant to the bottom one. Every
// line of Mermaid's is 2 wide, by its stylesheet's rule for line.
func lifeline(p *Participant, i int, l layout) scene.Item {
	c, top := l.e.x[i], looks[p.Shape].lifeline(l.e.c)
	return scene.Item{Kind: scene.Mark, Parent: p.ID, Box: between(c, top, c, l.bottom), Paths: []scene.Path{{
		Segs: line(c, top, c, l.bottom), Stroke: scene.Stroke{Paint: scene.Use(scene.LineColor), Width: 2},
	}}}
}

// participant is a participant's glyph and label, its top at y. The top one
// has the participant's id; the one at the foot is in it. The label's lines
// are a font size apart about the middle the glyph gives (byTspan).
func participant(p *Participant, i int, y float64, foot bool, l layout) scene.Item {
	e, lk := l.e, looks[p.Shape]
	c, w := e.x[i], e.w[i]
	paths, mid := lk.glyph(slot{x: c - w/2, c: c, y: y, w: w, h: e.c.height, foot: foot})
	it := scene.Item{Kind: scene.Node, ID: p.ID, Paths: paths}
	if foot {
		it.ID, it.Parent = "", p.ID
	}
	tb := measure(e.m, p.Label, e.c.actorFont)
	fs, n := tb.font.Size, float64(len(tb.lines))
	t, tbox := tb.set(e.m, c-lk.dx, false, func(j int) float64 { return tb.middle(mid + float64(j)*fs - fs*(n-1)/2) })
	it.Texts = []scene.Text{t}
	it.Box = union(pathsBox(paths), tbox)
	return it
}

package sequence

import "github.com/bencode/sumi/scene"

// A look is how participants of one shape draw (the drawActorType
// functions, svgDraw.js): where the lifeline starts below the top, how tall
// the shape is once drawn, how far left of the middle the label is, and the
// glyph, which gives the shape's paths and its label's middle.
//
// How tall it is once drawn is what drawing the top one leaves in Mermaid's
// actor.height: the glyph's bounding box (getBBox), plus a labelBoxHeight
// or two for some. The one at the foot is drawn that tall, and returns it
// as its height, which a box's bottom goes by.
type look struct {
	lifeline func(c config) float64
	height   func(c config, w float64) float64
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
	Box:         {lifeline: underBox, height: boxHeight, glyph: box},
	Figure:      {lifeline: fixed(80), height: figureHeight, glyph: figure},
	Collections: {lifeline: underBox, height: boxHeight, dx: 6, glyph: collections},
	Queue:       {lifeline: underBox, height: boxHeight, glyph: queue},
	Database:    {lifeline: underDatabase, height: databaseHeight, glyph: database},
	Control:     {lifeline: fixed(75), height: controlHeight, glyph: control},
	Entity:      {lifeline: fixed(75), height: circleHeight, glyph: entity},
	Boundary:    {lifeline: fixed(80), height: circleHeight, glyph: boundary},
}

// A lifeline starts at the participant's height below its top, a
// database's 2*boxTextMargin further down; under the drawn figures, at a
// fixed depth.
func underBox(c config) float64      { return c.height }
func underDatabase(c config) float64 { return c.height + 2*c.boxTextMargin }
func fixed(y float64) func(config) float64 {
	return func(config) float64 { return y }
}

// The heights once drawn: a box's, a queue's and the collections' front
// box's are the participant's height; the actor figure's is its fixed
// ACTOR_GLYPH_HEIGHT; a database's is its cylinder, as tall as it is wide,
// and a labelBoxHeight; a control's is its circle and two, an entity's and
// a boundary's their circle and one.
func boxHeight(c config, _ float64) float64      { return c.height }
func figureHeight(config, float64) float64       { return 65 }
func databaseHeight(c config, w float64) float64 { return w/3 + c.labelBoxHeight }
func controlHeight(c config, _ float64) float64  { return 2*circleR + 2*c.labelBoxHeight }
func circleHeight(c config, _ float64) float64   { return 2*circleR + c.labelBoxHeight }

// lifeline runs from under the top participant to the bottom one, which
// are where it is created and destroyed, if it is. Every line of Mermaid's
// is 2 wide, by its stylesheet's rule for line.
func lifeline(p *Participant, i int, l layout) scene.Item {
	c, top, foot := l.e.x[i], l.tops[i]+looks[p.Shape].lifeline(l.e.c), l.feet[i]
	return scene.Item{Kind: scene.Mark, Parent: p.ID, Box: between(c, top, c, foot), Paths: []scene.Path{{
		Segs: line(c, top, c, foot), Stroke: scene.Stroke{Paint: scene.Use(scene.LineColor), Width: 2},
	}}}
}

// participant is a participant's glyph and label, its top at y. The top one
// has the participant's id, and is in its box, if it has one; the one at
// the foot is in it. The label's lines are a font size apart about the
// middle the glyph gives (byTspan).
func participant(p *Participant, i int, y float64, foot bool, l layout) scene.Item {
	e, lk := l.e, looks[p.Shape]
	c, w, h := e.x[i], e.w[i], e.c.height
	if foot {
		h = lk.height(e.c, w)
	}
	paths, mid := lk.glyph(slot{x: c - w/2, c: c, y: y, w: w, h: h, foot: foot})
	it := scene.Item{Kind: scene.Node, ID: p.ID, Parent: l.boxID(p.Group), Paths: paths}
	if foot {
		it.ID, it.Parent = "", p.ID
	}
	tb := measure(e.m, e.labels[i], e.c.actorFont)
	fs, n := tb.font.Size, float64(len(tb.lines))
	t, tbox := tb.set(e.m, c-lk.dx, false, func(j int) float64 { return tb.middle(mid + float64(j)*fs - fs*(n-1)/2) })
	it.Texts = []scene.Text{t}
	it.Box = union(pathsBox(paths), tbox)
	return it
}

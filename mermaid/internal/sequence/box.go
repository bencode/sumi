package sequence

import (
	"fmt"
	"slices"

	"github.com/bencode/sumi/scene"
)

// A boxFit is how a box fits its participants and its title: Mermaid's
// box.margin, and the title as drawn.
type boxFit struct {
	margin float64
	title  string
}

// boxFits are the boxes' fits (calculateActorMargins, sequenceRenderer.ts
// lines 1672-1696). The total is the box's participants' widths and
// margins, plus 8 boxMargins, less 2 boxTextMargins; a title that asks to
// be wrapped is wrapped to it, less padding. The margin is boxTextMargin,
// and half of what the title, with wrapPadding each side, is wider than the
// total. A participant's margin there is the one calculateActorMargins has
// just set from what falls between it and the next, or past it for the
// last one; it is 0 when nothing does, as Mermaid sets that one only later.
func boxFits(d *Diagram, need []float64, e env) map[*Group]boxFit {
	inside := map[*Group]float64{}
	last := len(d.Participants) - 1
	for i, p := range d.Participants {
		if p.Group == nil {
			continue
		}
		var m float64
		switch {
		case need[i] == 0:
		case i < last:
			m = max(need[i]+e.c.actorMargin-e.w[i]/2-e.w[i+1]/2, e.c.actorMargin)
		default:
			m = max(need[i]+e.c.actorMargin-e.w[i]/2, e.c.actorMargin)
		}
		inside[p.Group] += e.w[i] + m
	}
	fits := map[*Group]boxFit{}
	for _, g := range d.Groups {
		total := inside[g] + 8*e.c.boxMargin - 2*e.c.boxTextMargin
		title := wrapIf(e.m, g.Wrap, g.Title, total-2*e.c.wrapPadding, e.c.messageFont)
		w := measure(e.m, title, e.c.messageFont).w + 2*e.c.wrapPadding
		fits[g] = boxFit{margin: e.c.boxTextMargin + max(0, (w-total)/2), title: title}
	}
	return fits
}

// spread adds the room around boxes to the gaps between participants
// (addActorRenderingData, sequenceRenderer.ts lines 745-810): a box's
// margin before its first participant, and boxMargin and its margin after
// its last. It returns the room before the first participant too.
func spread(ps []*Participant, g []float64, fits map[*Group]boxFit, c config) (float64, []float64) {
	var lead float64
	if len(ps) > 0 && ps[0].Group != nil {
		lead = fits[ps[0].Group].margin
	}
	out := slices.Clone(g)
	for i := 1; i < len(ps); i++ {
		a, b := ps[i-1].Group, ps[i].Group
		if a == b {
			continue
		}
		if a != nil {
			out[i-1] += c.boxMargin + fits[a].margin
		}
		if b != nil {
			out[i-1] += fits[b].margin
		}
	}
	return lead, out
}

// titleHeight is the tallest box title's height, in messageFont, as
// Mermaid measures them all (textMaxHeight).
func titleHeight(fits map[*Group]boxFit, e env) float64 {
	var h float64
	for _, f := range fits {
		h = max(h, measure(e.m, f.title, e.c.messageFont).h())
	}
	return h
}

// boxTop is how far down the top participants are: when there are boxes,
// boxMargin and the tallest title (sequenceRenderer.ts lines 1080-1085).
func boxTop(fits map[*Group]boxFit, e env) float64 {
	if len(fits) == 0 {
		return 0
	}
	return e.c.boxMargin + titleHeight(fits, e)
}

// A pBox is a box laid out. Its model is Mermaid's box: from its margin
// left of its first participant to its margin right of its last, and from
// the top to boxMargin under the tallest participant at the foot. Its
// frame, what it draws, is 2 boxMargins wider each side, and starts half a
// boxMargin higher and ends a boxMargin lower (sequenceRenderer.ts lines
// 1398-1408). text is its title as drawn, and mid the middle of its line.
type pBox struct {
	*Group
	id           string
	model, frame scene.Rect
	text         string
	mid          float64
}

// placeBoxes lays out the boxes that have participants; Mermaid draws no
// other.
func placeBoxes(l layout, fits map[*Group]boxFit) []pBox {
	e, ps := l.e, l.d.Participants
	var foot float64
	for i, p := range ps {
		foot = max(foot, looks[p.Shape].height(e.c, e.w[i]))
	}
	h := l.bottom + foot + e.c.boxMargin
	pad := 2 * e.c.boxMargin
	mid := e.c.boxTextMargin + titleHeight(fits, e)/2
	var out []pBox
	for _, g := range l.d.Groups {
		first := slices.IndexFunc(ps, func(p *Participant) bool { return p.Group == g })
		if first < 0 {
			continue
		}
		last := first
		for last+1 < len(ps) && ps[last+1].Group == g {
			last++
		}
		m := fits[g].margin
		x := e.x[first] - e.w[first]/2 - m
		w := e.x[last] + e.w[last]/2 + m - x
		out = append(out, pBox{
			Group: g,
			id:    fmt.Sprintf("box-%d", len(out)+1),
			model: scene.Rect{X: x, Y: 0, W: w, H: h},
			frame: scene.Rect{X: x - pad, Y: -pad / 4, W: w + 2*pad, H: h + pad*3/4},
			text:  fits[g].title,
			mid:   mid,
		})
	}
	return out
}

// boxID is the id of the box g is laid out as, or "" when there is none.
func (l layout) boxID(g *Group) string {
	for _, b := range l.boxes {
		if b.Group == g {
			return b.id
		}
	}
	return ""
}

// drawBox is a box's frame, in its color, and its title, in the actor
// font although it was measured in the message font, its lines a font size
// apart about mid (drawBox and byTspan, svgDraw.js). The frame's outline is
// the participants' (Mermaid's stylesheet rule for g rect.rect).
func drawBox(b pBox, l layout) scene.Item {
	it := scene.Item{Kind: scene.Group, ID: b.id, Box: b.frame, Paths: []scene.Path{{
		Segs: scene.RoundRect(b.frame, 0), Fill: scene.Fixed(b.Fill),
		Stroke: scene.Stroke{Paint: scene.Use(scene.NodeStroke), Width: 1},
	}}}
	if b.text == "" {
		return it
	}
	e := l.e
	tb := measure(e.m, b.text, e.c.actorFont)
	fs, n := tb.font.Size, float64(len(tb.lines))
	t, box := tb.set(e.m, b.model.X+b.model.W/2, false, func(j int) float64 {
		return tb.middle(b.mid + float64(j)*fs - fs*(n-1)/2)
	})
	it.Texts = []scene.Text{t}
	it.Box = union(it.Box, box)
	return it
}

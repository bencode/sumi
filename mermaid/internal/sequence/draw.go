package sequence

import (
	"cmp"
	"math"
	"slices"

	"github.com/bencode/sumi/scene"
)

// draw turns a layout into a scene, in the order Mermaid stacks things
// (draw, sequenceRenderer.ts lines 1143-1393): the lifelines lowest; then
// notes, activation bars and block frames in the order Mermaid adds them;
// the top participants; the messages; the bottom participants; the title.
func draw(l layout) *scene.Scene {
	var items []scene.Item
	for i, p := range l.d.Participants {
		items = append(items, lifeline(p, i, l))
	}
	stacked := slices.DeleteFunc(slices.Clone(l.items), func(p placed) bool {
		_, ok := p.(*pMessage)
		return ok
	})
	slices.SortStableFunc(stacked, func(a, b placed) int { return cmp.Compare(order(a), order(b)) })
	for _, p := range stacked {
		items = append(items, drawStacked(p, l))
	}
	for i, p := range l.d.Participants {
		items = append(items, participant(p, i, 0, false, l))
	}
	for _, p := range l.items {
		if m, ok := p.(*pMessage); ok {
			items = append(items, drawMessage(m, l)...)
		}
	}
	for i, p := range l.d.Participants {
		items = append(items, participant(p, i, l.bottom, true, l))
	}
	if l.d.Title != "" {
		items = append(items, drawTitle(l))
	}
	box := l.bounds
	for _, it := range items {
		box = union(box, it.Box)
	}
	return &scene.Scene{Width: box.W, Height: box.H, Items: moved(items, -box.X, -box.Y)}
}

func order(p placed) int {
	switch p := p.(type) {
	case *pNote:
		return p.order
	case *pBar:
		return p.order
	case *pBlock:
		return p.order
	}
	return 0
}

func drawStacked(p placed, l layout) scene.Item {
	switch p := p.(type) {
	case *pNote:
		return note(p, l)
	case *pBar:
		return scene.Item{Kind: scene.Mark, ID: p.id, Parent: p.parent, Box: p.box, Paths: []scene.Path{{
			Segs: scene.RoundRect(p.box, 0), Fill: scene.Use(scene.NodeFill),
			Stroke: scene.Stroke{Paint: scene.Use(scene.NodeStroke), Width: 1},
		}}}
	}
	return block(p.(*pBlock), l)
}

// drawMessage is a message's line, head and text (drawMessage,
// sequenceRenderer.ts lines 466-742), and its number if it has one.
func drawMessage(p *pMessage, l layout) []scene.Item {
	e, mm := l.e, p.model
	stroke := scene.Stroke{Paint: scene.Use(scene.LineColor), Width: 1.5}
	if p.Dotted {
		stroke.Dash = []float64{3, 3}
	}
	x1, x2, y := mm.startx, mm.stopx, p.y
	var path []scene.Seg
	var end, endDir, startDir scene.Point
	if p.From == p.To {
		path = (&scene.Path{}).MoveTo(x1, y).CubeTo(x1+60, y-10, x2+60, y+30, x2, y+20).Segs
		end, endDir, startDir = scene.Point{X: x2, Y: y + 20}, scene.Point{X: -60, Y: -10}, scene.Point{X: -60, Y: 10}
	} else {
		if p.Number != "" {
			x1 = numberedStart(p, mm, l.e.c)
		}
		path = line(x1, y, x2, y)
		end, endDir, startDir = scene.Point{X: x2, Y: y}, scene.Point{X: x2 - x1, Y: 0}, scene.Point{X: x1 - x2, Y: 0}
	}
	it := scene.Item{Kind: scene.Edge, ID: p.id, Parent: p.parent, From: p.From, To: p.To,
		Paths: []scene.Path{{Segs: path, Stroke: stroke}}}
	it.Paths = append(it.Paths, head(p.Head, end, endDir)...)
	if p.Both {
		it.Paths = append(it.Paths, head(Arrow, scene.Point{X: x1, Y: y}, startDir)...)
	}
	it.Paths = append(it.Paths, circles(p, l)...)
	cx := math.Round(min(mm.startx, mm.stopx) + math.Abs(mm.startx-mm.stopx)/2)
	t, box := p.text.set(e.m, cx, false, func(i int) float64 {
		return p.text.middle(math.Round(p.top+10+float64(i)*p.text.lh+e.c.wrapPadding/2) + p.text.font.Size)
	})
	it.Texts = []scene.Text{t}
	it.Box = union(union(box, between(x1, y, x2, y)), pathsBox(it.Paths))
	if p.Number == "" {
		return []scene.Item{it}
	}
	return []scene.Item{it, drawNumber(p, l)}
}

// numberedStart is where a numbered message's line starts: past the
// number's circle (lines 664-695), and for a two-headed one to the left,
// past a central connection too.
func numberedStart(p *pMessage, mm msgModel, c config) float64 {
	r := c.numberRadius // SEQUENCE_NUMBER_RADIUS
	if !p.Both {
		return mm.startx + r
	}
	if mm.startx < mm.stopx {
		return mm.startx + 2*r
	}
	x := mm.startx - r
	if p.CircleFrom || p.CircleTo {
		x -= 5
	}
	if p.CircleFrom {
		x -= 7.5
	}
	return x
}

// circles are a message's central connections: dots of radius 5 on the
// lifelines' middles, filled with the text color the diagram's root sets
// (drawCentralConnection, lines 288-375). With numbers on, the sender's
// dot moves 16.5 toward the receiver, out of the number's way.
func circles(p *pMessage, l layout) []scene.Path {
	from, to := l.e.x[l.e.at[p.From]], l.e.x[l.e.at[p.To]]
	dot := func(x float64) scene.Path {
		return scene.Path{Segs: scene.Ellipse(scene.Point{X: x, Y: p.y}, 5, 5), Fill: scene.Use(scene.TextColor)}
	}
	var out []scene.Path
	if p.CircleFrom {
		x := from
		if p.Number != "" {
			if from <= to {
				x += 16.5
			} else {
				x -= 16.5
			}
		}
		out = append(out, dot(x))
	}
	if p.CircleTo {
		out = append(out, dot(to))
	}
	return out
}

// drawNumber is a message's number in a filled circle, at the sender's end of
// the line. The circle is a marker of radius numberRadius on a line Mermaid's
// stylesheet makes 2 wide, which scales the marker by 2.
func drawNumber(p *pMessage, l layout) scene.Item {
	mm, r := p.model, 2*l.e.c.numberRadius
	x := mm.from + 1
	if p.From != p.To && mm.startx > mm.stopx {
		x = mm.to - 1
	}
	at := scene.Point{X: x, Y: p.y}
	size := l.e.c.numberFont
	if n := len(p.Number); n > 5 {
		size = 7
	} else if n > 3 {
		size = 9
	}
	tb := measure(l.e.m, p.Number, size)
	t, _ := tb.set(l.e.m, x, false, func(int) float64 { return p.y + 4 })
	t.Paint = scene.Use(scene.Background)
	return scene.Item{Kind: scene.Label, Parent: p.parent, Box: between(x-r, p.y-r, x+r, p.y+r),
		Paths: []scene.Path{{Segs: scene.Ellipse(at, r, r), Fill: scene.Use(scene.TextColor)}},
		Texts: []scene.Text{t}}
}

// note is a note's box and text, each line's middle 5 and a font size
// below the one before (drawNote).
func note(p *pNote, l layout) scene.Item {
	e := l.e
	t, _ := p.text.set(e.m, math.Round(p.box.X+p.box.W/2), false, func(i int) float64 {
		return p.text.middle(math.Round(p.box.Y+float64(i)*p.text.lh+e.c.noteMargin/2) + p.text.font.Size)
	})
	return scene.Item{Kind: scene.Node, ID: p.id, Parent: p.parent, Box: p.box,
		Paths: []scene.Path{{Segs: scene.RoundRect(p.box, 0), Fill: scene.Use(scene.NoteFill),
			Stroke: scene.Stroke{Paint: scene.Use(scene.NoteStroke), Width: 1}}},
		Texts: []scene.Text{t}}
}

var keywords = [...]string{Loop: "loop", Alt: "alt", Opt: "opt", Par: "par", Critical: "critical", Break: "break"}

// block is a block's frame, its dashed section lines, its keyword in a
// tab, and its titles (drawLoop, svgDraw.js lines 1519-1620). The titles
// sit on their baselines, as Mermaid sets them without a width.
func block(p *pBlock, l layout) scene.Item {
	e, b := l.e, p.box
	x0, y0, x1, y1 := b.X, b.Y, b.X+b.W, b.Y+b.H
	stroke := scene.Stroke{Paint: scene.Use(scene.GroupStroke), Width: 2, Dash: []float64{2, 2}}
	edges := (&scene.Path{}).MoveTo(x0, y0).LineTo(x1, y0).MoveTo(x1, y0).LineTo(x1, y1).
		MoveTo(x0, y1).LineTo(x1, y1).MoveTo(x0, y0).LineTo(x0, y1).Segs
	it := scene.Item{Kind: scene.Group, ID: p.id, Parent: p.parent, Box: b,
		Paths: []scene.Path{{Segs: edges, Stroke: stroke}}}
	for _, y := range p.lines[1:] {
		dashed := stroke
		dashed.Dash = []float64{3, 3}
		it.Paths = append(it.Paths, scene.Path{Segs: line(x0, y, x1, y), Stroke: dashed})
	}
	it.Paths = append(it.Paths, scene.Path{Segs: tab(x0, y0, e.c.labelBoxWidth, e.c.labelBoxHeight),
		Fill: scene.Use(scene.GroupFill), Stroke: scene.Stroke{Paint: scene.Use(scene.GroupStroke), Width: 1}})
	kw := measure(e.m, keywords[p.Kind], e.c.messageFont)
	t, _ := kw.set(e.m, math.Round(x0+e.c.labelBoxWidth/2), false, func(int) float64 {
		return kw.middle(math.Round(y0 + e.c.labelBoxHeight/2 + e.c.boxTextMargin/2))
	})
	it.Texts = []scene.Text{t}
	for i, tb := range p.titles {
		cx := x0 + (x1-x0)/2
		if i == 0 {
			cx += e.c.labelBoxWidth / 2
		}
		top := p.lines[i] + e.c.boxMargin + e.c.boxTextMargin
		t, box := tb.set(e.m, cx, false, func(j int) float64 {
			return math.Round(top + float64(j)*tb.lh + e.c.boxTextMargin/2)
		})
		if len(tb.lines) > 0 {
			it.Texts = append(it.Texts, t)
			it.Box = union(it.Box, box)
		}
	}
	return it
}

// drawTitle sets the title on a baseline 25 above the participants, starting 100 left of
// the middle of what Mermaid's bounds hold.
func drawTitle(l layout) scene.Item {
	e := l.e
	tb := measure(e.m, l.d.Title, e.c.titleFont)
	t, box := tb.set(e.m, l.bounds.W/2-100, true, func(i int) float64 { return -25 + float64(i)*tb.lh })
	return scene.Item{Kind: scene.Label, Box: box, Texts: []scene.Text{t}}
}

func pathsBox(ps []scene.Path) scene.Rect {
	x0, y0, x1, y1 := math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1)
	for _, p := range ps {
		for _, s := range p.Segs {
			for _, q := range s.Pts[:points(s.Op)] {
				x0, y0, x1, y1 = min(x0, q.X), min(y0, q.Y), max(x1, q.X), max(y1, q.Y)
			}
		}
	}
	if x0 > x1 {
		return scene.Rect{}
	}
	return between(x0, y0, x1, y1)
}

// points is how many points a segment of op takes.
func points(op scene.Op) int {
	switch op {
	case scene.MoveTo, scene.LineTo:
		return 1
	case scene.QuadTo:
		return 2
	case scene.CubeTo:
		return 3
	}
	return 0
}

// moved is items shifted by dx, dy.
func moved(items []scene.Item, dx, dy float64) []scene.Item {
	for i := range items {
		it := &items[i]
		it.Box.X, it.Box.Y = it.Box.X+dx, it.Box.Y+dy
		for _, p := range it.Paths {
			for j := range p.Segs {
				for k := range p.Segs[j].Pts {
					p.Segs[j].Pts[k].X += dx
					p.Segs[j].Pts[k].Y += dy
				}
			}
		}
		for _, t := range it.Texts {
			for j := range t.Lines {
				t.Lines[j].X += dx
				t.Lines[j].Baseline += dy
			}
		}
	}
	return items
}

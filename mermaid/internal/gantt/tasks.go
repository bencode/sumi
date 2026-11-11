package gantt

import (
	"math"
	"slices"

	"github.com/bencode/sumi/mermaid/internal/label"
	"github.com/bencode/sumi/mermaid/internal/syntax"
	"github.com/bencode/sumi/scene"
)

// bars are the tasks' rects, in the order drawn, as drawRects draws them. A task's bar
// runs from its start to where it is drawn to end, in its row. A milestone is a square
// the bar's height in the middle of its times, which the milestone class turns 45° and
// scales by 0.8 about that middle. A vert is a thin bar from the grid's top through every
// row. Its colors are its status's, a vert's line its own.
func bars(c chart) []scene.Item {
	var items []scene.Item
	for _, t := range c.tasks {
		start, end := c.a.x(ms(t.Start)), c.a.x(ms(t.End))
		r := scene.Rect{X: start + sidePadding, Y: float64(t.Order*rowGap + topPadding), W: c.a.x(drawnEnd(t)) - start, H: barHeight}
		if t.Milestone {
			r.X, r.W = start+sidePadding+0.5*(end-start)-0.5*barHeight, barHeight
		} else if t.Vert {
			r.W = 0.08 * barHeight
		}
		fill, outline := barColors(t)
		if t.Vert {
			r.Y, r.H = gridLineStartPadding, float64(c.rows*rowGap+2*barHeight)
			outline = scene.VertLine
		}
		bar := rect(scene.Node, r, 3, scene.Use(fill), scene.Stroke{Paint: scene.Use(outline), Width: 2})
		if len(bar) == 0 {
			continue
		}
		bar[0].ID = t.ID
		if t.Milestone {
			// transform-origin; one that is NaN is no origin, and the transform turns about 0, 0.
			o := scene.Point{X: start + sidePadding + 0.5*(end-start), Y: float64(t.Order*rowGap+topPadding) + 0.5*barHeight}
			if math.IsNaN(o.X) {
				o = scene.Point{}
			}
			bar[0] = turned(bar[0], o)
		}
		items = append(items, bar...)
	}
	return items
}

// barColors are a bar's fill and outline as its class sets them: active over done, each
// crit or not, a crit one's outline crit's.
func barColors(t Task) (fill, outline scene.Token) {
	switch {
	case t.Active && t.Crit:
		return scene.ActiveTaskFill, scene.CritTaskStroke
	case t.Active:
		return scene.ActiveTaskFill, scene.ActiveTaskStroke
	case t.Done && t.Crit:
		return scene.DoneTaskFill, scene.CritTaskStroke
	case t.Done:
		return scene.DoneTaskFill, scene.DoneTaskStroke
	case t.Crit:
		return scene.CritTaskFill, scene.CritTaskStroke
	}
	return scene.TaskFill, scene.TaskStroke
}

// drawnEnd is where a task's bar ends, renderEndTime || endTime.
func drawnEnd(t Task) float64 {
	if t.RenderEnd != nil {
		return ms(t.RenderEnd)
	}
	return ms(t.End)
}

// turned is it turned 45° clockwise about o and scaled by 0.8, as the milestone class's
// rotate(45deg) scale(0.8,0.8) transforms it.
func turned(it scene.Item, o scene.Point) scene.Item {
	k := 0.8 * math.Sqrt2 / 2
	at := func(p scene.Point) scene.Point {
		dx, dy := p.X-o.X, p.Y-o.Y
		return scene.Point{X: o.X + float64(k*dx) - float64(k*dy), Y: o.Y + float64(k*dx) + float64(k*dy)}
	}
	p := it.Paths[0]
	p.Segs = slices.Clone(p.Segs)
	for i := range p.Segs {
		for j := range p.Segs[i].Pts {
			p.Segs[i].Pts[j] = at(p.Segs[i].Pts[j])
		}
	}
	b := it.Box
	x0, y0, x1, y1 := math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1)
	for _, q := range []scene.Point{{X: b.X, Y: b.Y}, {X: b.X + b.W, Y: b.Y}, {X: b.X, Y: b.Y + b.H}, {X: b.X + b.W, Y: b.Y + b.H}} {
		q = at(q)
		x0, y0, x1, y1 = min(x0, q.X), min(y0, q.Y), max(x1, q.X), max(y1, q.Y)
	}
	it.Paths, it.Box = []scene.Path{p}, scene.Rect{X: x0, Y: y0, W: x1 - x0, H: y1 - y0}
	return it
}

// taskTexts are the tasks' texts, after all the bars, as drawRects sets them. Where a text
// goes and how it lies about there are worked out apart, from times that are not quite
// the same: see textAt.
func taskTexts(c chart) []scene.Item {
	var items []scene.Item
	for _, t := range c.tasks {
		s := label.Collapsed([]string{syntax.Decode(t.Text)})[0]
		if s == "" {
			continue
		}
		x, align, outside := textAt(t, c)
		f := scene.Font{Size: fontSize, Italic: t.Milestone, Bold: t.Clickable}
		y := float64(t.Order*rowGap+barHeight/2) + (fontSize/2.0 - 2) + topPadding
		if t.Vert {
			f.Size, y = 15, float64(gridLineStartPadding+c.rows*rowGap+60)
		}
		it := textOf([]scene.Line{line(c.m, s, f, lacuna(x), align, y)}, f, c.m, scene.Use(textColor(t, outside)))
		it.Parent = t.ID
		items = append(items, it)
	}
	return items
}

// textColor is the color of a task's text, outside its bar or in it, as the !important
// rules of its classes set it: of those, the more specific, then the later in the
// stylesheet. Text outside a done bar is on the background, and clickable text is a
// link's, then in turn activeCritText, vertText, the active and done texts, and those
// with no !important.
func textColor(t Task, outside bool) scene.Token {
	switch {
	case t.Done && outside:
		return scene.TextColor
	case t.Clickable:
		return scene.LinkText
	case t.Active && t.Crit:
		return scene.StatusTaskText
	case t.Vert:
		return scene.VertLine
	case t.Active || t.Done:
		return scene.StatusTaskText
	case outside:
		return scene.TextColor
	}
	return scene.TaskText
}

// textAt is where a task's text is set, how it lies about there, and whether its class
// has it outside the bar. The text is measured as the renderer measures it, at 11 and
// upright, before its class is set, and with its entity codes Mermaid's placeholders. It
// goes in the middle of the bar where it fits, else right of the bar, or left of it where
// it would run past the width. The x looks at where the bar is drawn; the class, which
// sets text-anchor, at the bar from start to end, a milestone's from its start. A vert's
// text is on the vert, in its middle, as the vertText class has it, but its class is
// worked out all the same.
func textAt(t Task, c chart) (float64, scene.Align, bool) {
	start, end := c.a.x(ms(t.Start)), c.a.x(ms(t.End))
	width := c.m.Measure(label.Collapsed([]string{syntax.Placeholders(t.Text)})[0], scene.Font{Size: fontSize}).Width
	place := func(from, to float64) (float64, scene.Align) {
		if !(width > to-from) {
			return (to-from)/2 + from + sidePadding, scene.Center
		}
		if to+width+1.5*sidePadding > c.w {
			return from + sidePadding - 5, scene.Right
		}
		return to + sidePadding + 5, scene.Left
	}
	classEnd := end
	if t.Milestone {
		classEnd = start + barHeight
	}
	_, align := place(start, classEnd)
	outside := align != scene.Center
	if t.Vert {
		return start + sidePadding, scene.Center, outside
	}
	from, to := start, c.a.x(drawnEnd(t))
	if t.Milestone {
		from += 0.5*(end-start) - 0.5*barHeight
		to = from + barHeight
	}
	x, _ := place(from, to)
	return x, align, outside
}

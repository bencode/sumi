package gantt

import (
	"math"
	"slices"

	"github.com/bencode/sumi/mermaid/internal/syntax"
	"github.com/bencode/sumi/scene"
)

// bars are the tasks' rects, in the order drawn, as drawRects draws them. A task's bar
// runs from its start to where it is drawn to end, in its row. A milestone is a square
// the bar's height in the middle of its times, which the milestone class turns 45° and
// scales by 0.8 about that middle. A vert is a thin bar from the grid's top through every
// row.
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
		stroke := scene.Stroke{Paint: scene.Use(scene.NodeStroke), Width: 2}
		if t.Vert {
			r.Y, r.H = gridLineStartPadding, float64(c.rows*rowGap+2*barHeight)
			stroke.Paint = scene.Use(scene.LineColor)
		}
		bar := rect(scene.Node, r, 3, scene.Use(scene.NodeFill), stroke)
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
		s := collapsed([]string{syntax.Decode(t.Text)})[0]
		if s == "" {
			continue
		}
		x, align := textAt(t, c)
		f := scene.Font{Size: fontSize, Italic: t.Milestone, Bold: t.Clickable}
		y := float64(t.Order*rowGap+barHeight/2) + (fontSize/2.0 - 2) + topPadding
		if t.Vert {
			f.Size, y = 15, float64(gridLineStartPadding+c.rows*rowGap+60)
		}
		it := textOf([]scene.Line{line(c.m, s, f, lacuna(x), align, y)}, f, c.m)
		it.Parent = t.ID
		items = append(items, it)
	}
	return items
}

// textAt is where a task's text is set and how it lies about there. The text is measured
// as the renderer measures it, at 11 and upright, before its class is set, and with its
// entity codes Mermaid's placeholders. It goes in the middle of the bar where it fits,
// else right of the bar, or left of it where it would run past the width. The x looks at
// where the bar is drawn; the class, which sets text-anchor, at the bar from start to end,
// a milestone's from its start. A vert's text is on the vert, in its middle, as the
// vertText class has it.
func textAt(t Task, c chart) (float64, scene.Align) {
	start, end := c.a.x(ms(t.Start)), c.a.x(ms(t.End))
	if t.Vert {
		return start + sidePadding, scene.Center
	}
	width := c.m.Measure(collapsed([]string{syntax.Placeholders(t.Text)})[0], scene.Font{Size: fontSize}).Width
	place := func(from, to float64) (float64, scene.Align) {
		if !(width > to-from) {
			return (to-from)/2 + from + sidePadding, scene.Center
		}
		if to+width+1.5*sidePadding > c.w {
			return from + sidePadding - 5, scene.Right
		}
		return to + sidePadding + 5, scene.Left
	}
	from, to := start, c.a.x(drawnEnd(t))
	if t.Milestone {
		from += 0.5*(end-start) - 0.5*barHeight
		to = from + barHeight
	}
	x, _ := place(from, to)
	if t.Milestone {
		end = start + barHeight
	}
	_, align := place(start, end)
	return x, align
}

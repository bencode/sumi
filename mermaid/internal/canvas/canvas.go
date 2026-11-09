// Package canvas finishes the drawing of a diagram that goes through Mermaid's
// unified renderer: its items in the box of them all, under a title.
package canvas

import (
	"slices"

	"github.com/bencode/sumi/mermaid/internal/label"
	"github.com/bencode/sumi/scene"
)

// The title's font size and baseline (the diagram's TitleText class, insertTitle
// with titleTopMargin), and the room setupViewPortForSVG leaves about the diagram
// (diagramPadding).
const (
	titleSize   = 18
	titleBase   = -25
	viewPadding = 8
)

// Scene is items and the diagram's title, if any, in a scene the box of it all,
// which leaves out markers and stroke widths as getBBox does, viewPadding wider on
// every side.
func Scene(items []scene.Item, title string, m scene.Measurer) *scene.Scene {
	box := bounds(items)
	if title != "" {
		t := drawTitle(title, box.X+box.W/2, m)
		items = append(items, t)
		box = Union(box, t.Box)
	}
	box = scene.Rect{X: box.X - viewPadding, Y: box.Y - viewPadding, W: box.W + 2*viewPadding, H: box.H + 2*viewPadding}
	return &scene.Scene{Width: box.W, Height: box.H, Items: moved(items, -box.X, -box.Y)}
}

// bounds is the box of items as Blink's SVG containers take it: a rect or a text with
// no width or no height counts for nothing, but a path counts however flat it is, as a
// straight line is.
func bounds(items []scene.Item) scene.Rect {
	box, found := scene.Rect{}, false
	for _, it := range items {
		flat := it.Box.W <= 0 || it.Box.H <= 0
		if flat && (it.Kind != scene.Edge || it.Box == (scene.Rect{})) {
			continue
		}
		if found {
			box = span(box, it.Box)
		} else {
			box, found = it.Box, true
		}
	}
	return box
}

// drawTitle is the diagram's title, centered on x, its baseline at titleBase.
func drawTitle(title string, x float64, m scene.Measurer) scene.Item {
	f := scene.Font{Size: titleSize}
	w := m.Measure(title, f).Width
	a, d := label.Extent(m, f)
	return scene.Item{
		Kind: scene.Label,
		Box:  scene.Rect{X: x - w/2, Y: titleBase - a, W: w, H: a + d},
		Texts: []scene.Text{{Font: f, Paint: scene.Use(scene.TextColor),
			Lines: []scene.Line{{S: title, X: x - w/2, Baseline: titleBase}}}},
	}
}

// Union is the box that holds a and b, as gfx::RectF::Union unites the boxes of
// getBBox: a box with no width or no height holds nothing.
func Union(a, b scene.Rect) scene.Rect {
	if b.W <= 0 || b.H <= 0 {
		return a
	}
	if a.W <= 0 || a.H <= 0 {
		return b
	}
	return span(a, b)
}

// span is the box that holds a and b, however flat either is.
func span(a, b scene.Rect) scene.Rect {
	x0, y0 := min(a.X, b.X), min(a.Y, b.Y)
	x1, y1 := max(a.X+a.W, b.X+b.W), max(a.Y+a.H, b.Y+b.H)
	return scene.Rect{X: x0, Y: y0, W: x1 - x0, H: y1 - y0}
}

// moved is items shifted by dx, dy.
func moved(items []scene.Item, dx, dy float64) []scene.Item {
	for i := range items {
		it := &items[i]
		it.Box.X, it.Box.Y = it.Box.X+dx, it.Box.Y+dy
		for pi := range it.Paths {
			segs := slices.Clone(it.Paths[pi].Segs) // a fill and a stroke may share theirs
			for j := range segs {
				for k := range segs[j].Pts {
					segs[j].Pts[k].X += dx
					segs[j].Pts[k].Y += dy
				}
			}
			it.Paths[pi].Segs = segs
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

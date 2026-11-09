package er

import (
	"slices"

	"github.com/bencode/sumi/mermaid/internal/canvas"
	"github.com/bencode/sumi/mermaid/internal/elk"
	"github.com/bencode/sumi/mermaid/internal/graphlayout"
	"github.com/bencode/sumi/mermaid/internal/label"
	"github.com/bencode/sumi/mermaid/internal/line"
	"github.com/bencode/sumi/mermaid/internal/style"
	"github.com/bencode/sumi/scene"
)

// What the stylesheet and the renderers give an ER diagram's parts.
const (
	strokeWidth = 2
	frameWidth  = 1  // .cluster rect's
	titleSize   = 14 // the page's: the stylesheet has no .erDiagramTitleText
)

// Layout reads an ER diagram's body, its front matter already taken off, and draws it;
// title is the front matter's title, if any.
func Layout(body, title string, m scene.Measurer) (*scene.Scene, error) {
	d, err := Read(body)
	if err != nil {
		return nil, err
	}
	for _, n := range d.Nodes {
		if err := style.Check(n.At, slices.Concat(n.Compiled, n.Styles)); err != nil {
			return nil, err
		}
	}
	p, err := Place(d, m)
	if err != nil {
		return nil, err
	}
	return draw(p, title, m), nil
}

// draw turns a laid-out diagram into a scene, stacked as Mermaid paints it: the
// subgraphs, the lines, the texts on them, the entities, then the title.
func draw(p Placement, title string, m scene.Measurer) *scene.Scene {
	index := make(map[string]int, len(p.D.Nodes))
	for i, n := range p.D.Nodes {
		index[n.ID] = i
	}
	var groups, nodes, edges, labels []scene.Item
	for _, pn := range p.R.Nodes {
		i := index[pn.ID]
		if p.D.Nodes[i].Group {
			groups = append(groups, drawGroup(p.D.Nodes[i], p.Titles[i], pn.Box, m))
		} else {
			nodes = append(nodes, drawEntity(p.D.Nodes[i], p.Sized[i], pn.Box, m))
		}
	}
	lines := make([][]scene.Point, len(p.R.Edges))
	for i, pe := range p.R.Edges {
		for _, v := range pe.Points {
			lines[i] = append(lines[i], scene.Point{X: v.X, Y: v.Y})
		}
	}
	cs := line.Crossings(lines)
	for i, pe := range p.R.Edges {
		e := p.D.Edges[i]
		edges = append(edges, drawEdge(e, lines[i], cs[i]))
		if pe.Label != nil {
			labels = append(labels, drawEdgeLabel(e, p.Edges[i], *pe.Label, m))
		}
	}
	return canvas.SceneTitled(slices.Concat(groups, edges, labels, nodes), title, titleSize, m)
}

// drawEntity is an entity at b, a plain box or, with attributes, a table. An author's
// styles reach everything drawn in it, and the color of its texts.
func drawEntity(n Node, s Sized, b graphlayout.Box, m scene.Measurer) scene.Item {
	c := scene.Point{X: b.X, Y: b.Y}
	r := scene.Rect{X: c.X - s.Shape.W/2, Y: c.Y - s.Shape.H/2, W: s.Shape.W, H: s.Shape.H}
	st := style.ForNode(n.Compiled, n.Styles)
	it := scene.Item{
		Kind: scene.Node, ID: n.ID, Parent: n.Parent, Box: r,
		Paths: st.Outline(scene.RoundRect(r, 0), scene.NodeFill, scene.NodeStroke, strokeWidth, 0),
	}
	ink := style.Paint(st.Color, scene.TextColor)
	if s.Table == nil {
		top := scene.Point{X: c.X + s.Shape.Label.X - s.Label.W/2, Y: c.Y + s.Shape.Label.Y - s.Label.H/2}
		it.Texts = label.Texts(m, s.Label, top, ink)
		return it
	}
	return drawTable(it, st, *s.Table, c, ink, m)
}

// drawTable is what erBox draws over an entity's outline: a box for each row, the
// lines between the columns and under the name, and the texts, set from the left by
// addText.
func drawTable(it scene.Item, st style.Styling, t Table, c scene.Point, ink scene.Paint, m scene.Measurer) scene.Item {
	r := it.Box
	it.Paths = append(it.Paths, sides(st, r)...)
	for _, row := range t.Rows {
		box := scene.Rect{X: r.X, Y: c.Y + row.Y, W: r.W, H: row.H}
		it.Paths = slices.Concat(it.Paths, st.Outline(scene.RoundRect(box, 0), scene.NodeFill, scene.NodeStroke, strokeWidth, 0), sides(st, box))
	}
	it.Paths = append(it.Paths, divider(st, scene.Point{X: r.X, Y: c.Y + t.Rule}, scene.Point{X: r.X + r.W, Y: c.Y + t.Rule})...)
	for _, x := range t.Columns {
		it.Paths = append(it.Paths, divider(st, scene.Point{X: c.X + x, Y: c.Y + t.Rule}, scene.Point{X: c.X + x, Y: r.Y + r.H})...)
	}
	put := func(p Placed) {
		box := p.Box
		box.Left = true
		it.Texts = append(it.Texts, label.Texts(m, box, scene.Point{X: c.X + p.At.X, Y: c.Y + p.At.Y}, ink)...)
	}
	put(t.Name)
	for _, row := range t.Rows {
		put(row.Type)
		put(row.Name)
		put(row.Keys)
		put(row.Comment)
	}
	return it
}

func straight(a, b scene.Point) []scene.Seg {
	return (&scene.Path{}).MoveTo(a.X, a.Y).LineTo(b.X, b.Y).Segs
}

// leg is a straight stroke whose dashes start at a.
func leg(st style.Styling, a, b scene.Point) []scene.Path {
	return line.Dashed(straight(a, b), st.Dash, st.Pen(scene.NodeStroke, strokeWidth), 0)
}

// sides are the strokes roughjs lays over a dashed box's outline: each side starts its
// dashes again.
func sides(st style.Styling, b scene.Rect) []scene.Path {
	if !st.Dashed {
		return nil
	}
	nw, ne := scene.Point{X: b.X, Y: b.Y}, scene.Point{X: b.X + b.W, Y: b.Y}
	se, sw := scene.Point{X: b.X + b.W, Y: b.Y + b.H}, scene.Point{X: b.X, Y: b.Y + b.H}
	return slices.Concat(leg(st, nw, ne), leg(st, ne, se), leg(st, se, sw), leg(st, sw, nw))
}

// divider is a line in a table. Roughjs makes it a thin polygon, so a dashed one has its
// dashes from both ends.
func divider(st style.Styling, a, b scene.Point) []scene.Path {
	if st.Dashed {
		return slices.Concat(leg(st, a, b), leg(st, b, a))
	}
	return []scene.Path{{Segs: straight(a, b), Stroke: st.Pen(scene.NodeStroke, strokeWidth)}}
}

// drawGroup is a subgraph's frame at b, widened about its center to hold the title and
// the padding, with the title at its top, set again in the width layout gave the
// subgraph (clusters.js).
func drawGroup(n Node, lines [][]label.Span, b graphlayout.Box, m scene.Measurer) scene.Item {
	title := label.Set(m, lines, b.Width, 0)
	w := max(b.Width, title.W+groupPadding)
	r := scene.Rect{X: b.X - w/2, Y: b.Y - b.Height/2, W: w, H: b.Height}
	top := scene.Point{X: b.X - title.W/2, Y: r.Y}
	st := style.ForNode(n.Compiled, n.Styles)
	it := scene.Item{
		Kind: scene.Group, ID: n.ID, Parent: n.Parent,
		Box:   canvas.Union(r, scene.Rect{X: top.X, Y: top.Y, W: title.W, H: title.H}),
		Texts: label.Texts(m, title, top, style.Paint(st.Color, scene.TextColor)),
	}
	if r.H > 0 { // SVG draws no <rect> with no height, as an empty subgraph has
		it.Paths = st.Outline(scene.RoundRect(r, 0), scene.GroupFill, scene.GroupStroke, frameWidth, 0)
	}
	return it
}

// drawEdge is a relationship's line and its markers. A line with hops comes out solid:
// lineJump's rewrite replaces its dash array.
func drawEdge(e Edge, pts []scene.Point, cs []line.Crossing) scene.Item {
	it := scene.Item{Kind: scene.Edge, ID: e.ID, From: e.Start, To: e.End}
	segs := line.Line(pts, e.StartMarker, e.EndMarker, cs)
	if len(segs) == 0 {
		return it
	}
	it.Box = line.Bounds(segs)
	plain := line.ArcLength(line.Line(pts, e.StartMarker, e.EndMarker, nil))
	ink := scene.Use(scene.LineColor)
	pen := scene.Stroke{Paint: ink, Width: strokeWidth}
	if e.Pattern == "solid" {
		it.Paths = []scene.Path{{Segs: line.Trimmed(segs, 0, 0, plain), Stroke: pen}}
	} else if len(cs) > 0 && plain >= 4 {
		it.Paths = []scene.Path{{Segs: line.Trimmed(segs, 0, 2, plain), Stroke: pen}}
	} else {
		it.Paths = line.Dotted(segs, 0, 0, plain, pen)
	}
	if u, ok := line.Heading(segs, true); ok {
		it.Paths = append(it.Paths, line.Marker(e.StartMarker, segs[0].Pts[0], u, true, ink)...)
	}
	if u, ok := line.Heading(segs, false); ok {
		it.Paths = append(it.Paths, line.Marker(e.EndMarker, segs[len(segs)-1].Pts[0], u, false, ink)...)
	}
	return it
}

// drawEdgeLabel is the text on a line about c, on a patch of the background its size.
func drawEdgeLabel(e Edge, l label.Box, c elk.Vec, m scene.Measurer) scene.Item {
	r := scene.Rect{X: c.X - l.W/2, Y: c.Y - l.H/2, W: l.W, H: l.H}
	return scene.Item{
		Kind: scene.Label, ID: e.ID, Box: r,
		Paths: []scene.Path{{Segs: scene.RoundRect(r, 0), Fill: scene.Use(scene.Background)}},
		Texts: label.Texts(m, l, scene.Point{X: r.X, Y: r.Y}, scene.Use(scene.TextColor)),
	}
}

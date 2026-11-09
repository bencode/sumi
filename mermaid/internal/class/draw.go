package class

import (
	"slices"
	"unicode/utf16"

	"github.com/bencode/sumi/mermaid/internal/canvas"
	"github.com/bencode/sumi/mermaid/internal/elk"
	"github.com/bencode/sumi/mermaid/internal/graphlayout"
	"github.com/bencode/sumi/mermaid/internal/label"
	"github.com/bencode/sumi/mermaid/internal/line"
	"github.com/bencode/sumi/mermaid/internal/style"
	"github.com/bencode/sumi/scene"
)

// What the stylesheet and the renderers give a class diagram's parts.
const (
	strokeWidth      = 2
	frameWidth       = 1   // .cluster rect's
	namespacePadding = 16  // getData's, by which clusters.js widens a frame for its title
	underlineGap     = 1   // from the baseline to the top of the line Blink draws at 14px
	underlineWidth   = 1.4 // max(1, font size / 10)
	titleSize        = 14  // the page's: the stylesheet's .classTitleText is not the class the title has
	endCharWidth     = 9   // setTerminalWidth's, for each character of a text at an edge's end
)

// Layout reads a class diagram's body, its front matter already taken off, and draws
// it; title is the front matter's title, if any.
func Layout(body, title string, m scene.Measurer) (*scene.Scene, error) {
	d, err := Read(body)
	if err != nil {
		return nil, err
	}
	for _, n := range d.Nodes {
		if err := style.Check(n.At, n.Styles); err != nil {
			return nil, err
		}
	}
	p, err := Place(d, m)
	if err != nil {
		return nil, err
	}
	return draw(p, title, m), nil
}

// draw turns a laid-out diagram into a scene, stacked as paintLayoutData stacks it:
// the namespaces, the edges, the texts on them, the nodes, then the title.
func draw(p Placement, title string, m scene.Measurer) *scene.Scene {
	index := make(map[string]int, len(p.D.Nodes))
	for i, n := range p.D.Nodes {
		index[n.ID] = i
	}
	var groups, nodes, edges, labels []scene.Item
	for _, pn := range p.R.Nodes {
		i := index[pn.ID]
		if p.D.Nodes[i].Kind == Namespace {
			groups = append(groups, drawGroup(p.D.Nodes[i], p.Titles[i], pn.Box, m))
		} else {
			nodes = append(nodes, drawNode(p.D.Nodes[i], p.Sized[i], pn.Box, m))
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
		if pe.Start != nil {
			labels = append(labels, drawEndText(e, e.StartLabel, *p.Starts[i], *pe.Start, m))
		}
		if pe.End != nil {
			labels = append(labels, drawEndText(e, e.EndLabel, *p.Ends[i], *pe.End, m))
		}
	}
	return canvas.SceneTitled(slices.Concat(groups, edges, labels, nodes), title, titleSize, m)
}

// drawNode is a class, a note or an interface at b. An interface's box has opacity 0,
// so only its text shows.
func drawNode(n Node, s Sized, b graphlayout.Box, m scene.Measurer) scene.Item {
	c := scene.Point{X: b.X, Y: b.Y}
	r := scene.Rect{X: c.X - s.Shape.W/2, Y: c.Y - s.Shape.H/2, W: s.Shape.W, H: s.Shape.H}
	it := scene.Item{Kind: scene.Node, ID: n.ID, Parent: n.Parent, Box: r}
	if n.Kind == Class {
		return drawClass(it, n, *s.Class, c, m)
	}
	top := scene.Point{X: c.X + s.Shape.Label.X - s.Label.W/2, Y: c.Y + s.Shape.Label.Y - s.Label.H/2}
	it.Box = canvas.Union(r, scene.Rect{X: top.X, Y: top.Y, W: s.Label.W, H: s.Label.H})
	text := s.Label
	text.Left = n.Kind == Note // getData's style for a note
	it.Texts = label.Texts(m, text, top, scene.Use(scene.TextColor))
	if n.Kind == Note {
		it.Paths = []scene.Path{{
			Segs: scene.RoundRect(r, 0), Fill: scene.Use(scene.NoteFill),
			Stroke: scene.Stroke{Paint: scene.Use(scene.NoteStroke), Width: strokeWidth},
		}}
	}
	return it
}

// drawClass is classBox: the outline, the two lines across it, and its texts. An
// author's styles reach the outline, the lines and the color of the texts.
func drawClass(it scene.Item, n Node, b ClassBox, c scene.Point, m scene.Measurer) scene.Item {
	st := style.ForNode(nil, n.Styles)
	it.Paths = st.Outline(scene.RoundRect(it.Box, 0), scene.NodeFill, scene.NodeStroke, strokeWidth, 0)
	pen := st.Pen(scene.NodeStroke, strokeWidth)
	for _, y := range b.Dividers {
		rule := (&scene.Path{}).MoveTo(it.Box.X, c.Y+y).LineTo(it.Box.X+it.Box.W, c.Y+y)
		if st.Dashed {
			it.Paths = append(it.Paths, line.Dashed(rule.Segs, st.Dash, pen, 0)...)
		} else {
			it.Paths = append(it.Paths, scene.Path{Segs: rule.Segs, Stroke: pen})
		}
	}
	ink := style.Paint(st.Color, scene.TextColor)
	put := func(p Placed, static bool) {
		ts := label.Texts(m, p.Box, scene.Point{X: c.X + p.At.X, Y: c.Y + p.At.Y}, ink)
		it.Texts = append(it.Texts, ts...)
		if static {
			it.Paths = append(it.Paths, underlines(ts, m)...)
		}
	}
	if b.Annotation != nil {
		put(*b.Annotation, false)
	}
	put(b.Title, false)
	for i, p := range b.Members {
		put(p, n.Members[i].Classifier == "$")
	}
	for i, p := range b.Methods {
		put(p, n.Methods[i].Classifier == "$")
	}
	return it
}

// underlines are the lines text-decoration draws under ts: one under each baseline,
// from where its text starts to where it ends. classBox sets it on a group an author's
// color does not reach, so they stay the color of the text around the diagram.
func underlines(ts []scene.Text, m scene.Measurer) []scene.Path {
	type run struct{ baseline, left, right float64 }
	var runs []run
	for _, t := range ts {
		for _, l := range t.Lines {
			right := l.X + m.Measure(l.S, t.Font).Width
			i := slices.IndexFunc(runs, func(r run) bool { return r.baseline == l.Baseline })
			if i < 0 {
				runs = append(runs, run{l.Baseline, l.X, right})
				continue
			}
			runs[i].left, runs[i].right = min(runs[i].left, l.X), max(runs[i].right, right)
		}
	}
	out := make([]scene.Path, len(runs))
	for i, r := range runs {
		bar := scene.Rect{X: r.left, Y: r.baseline + underlineGap, W: r.right - r.left, H: underlineWidth}
		out[i] = scene.Path{Segs: scene.RoundRect(bar, 0), Fill: scene.Use(scene.TextColor)}
	}
	return out
}

// drawGroup is a namespace's frame at b, widened about its center to hold the title
// and the padding, with the title at its top.
func drawGroup(n Node, title label.Box, b graphlayout.Box, m scene.Measurer) scene.Item {
	w := max(b.Width, title.W+namespacePadding)
	r := scene.Rect{X: b.X - w/2, Y: b.Y - b.Height/2, W: w, H: b.Height}
	top := scene.Point{X: b.X - title.W/2, Y: r.Y}
	it := scene.Item{
		Kind: scene.Group, ID: n.ID, Parent: n.Parent,
		Box:   canvas.Union(r, scene.Rect{X: top.X, Y: top.Y, W: title.W, H: title.H}),
		Texts: label.Texts(m, title, top, scene.Use(scene.TextColor)),
	}
	if r.H > 0 { // SVG draws no <rect> with no height, as an empty namespace has
		it.Paths = []scene.Path{{
			Segs: scene.RoundRect(r, 0), Fill: scene.Use(scene.GroupFill),
			Stroke: scene.Stroke{Paint: scene.Use(scene.GroupStroke), Width: frameWidth},
		}}
	}
	return it
}

// drawEdge is an edge's line and its markers. A dashed or a dotted line is drawn alike,
// and a line with hops comes out solid: lineJump's rewrite replaces its dash array.
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

// drawEdgeLabel is the text on an edge about c, on a patch of the background its size.
func drawEdgeLabel(e Edge, l label.Box, c elk.Vec, m scene.Measurer) scene.Item {
	it := textAbout(e, l, c, m)
	it.Paths = []scene.Path{{Segs: scene.RoundRect(it.Box, 0), Fill: scene.Use(scene.Background)}}
	return it
}

// drawEndText is the text at an end of an edge about c. setTerminalWidth widens its box
// past the text, to the right, by how many UTF-16 units value is, which the diagram's
// bounds take in.
func drawEndText(e Edge, value string, l label.Box, c elk.Vec, m scene.Measurer) scene.Item {
	it := textAbout(e, l, c, m)
	it.Box.W = max(l.W, endCharWidth*float64(len(utf16.Encode([]rune(value)))))
	return it
}

func textAbout(e Edge, l label.Box, c elk.Vec, m scene.Measurer) scene.Item {
	r := scene.Rect{X: c.X - l.W/2, Y: c.Y - l.H/2, W: l.W, H: l.H}
	return scene.Item{
		Kind: scene.Label, ID: e.ID, Box: r,
		Texts: label.Texts(m, l, scene.Point{X: r.X, Y: r.Y}, scene.Use(scene.TextColor)),
	}
}

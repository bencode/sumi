package state

import (
	"slices"

	"github.com/bencode/sumi/mermaid/internal/canvas"
	"github.com/bencode/sumi/mermaid/internal/graphlayout"
	"github.com/bencode/sumi/mermaid/internal/label"
	"github.com/bencode/sumi/mermaid/internal/line"
	"github.com/bencode/sumi/mermaid/internal/style"
	"github.com/bencode/sumi/scene"
)

// What the stylesheet gives the neo look, where it outweighs what the shapes set.
const (
	stateRadius     = 5  // .statediagram-state rect.basic and .title-state's rx
	compositeRadius = 12 // the theme's radius, on rect.outer
	strokeWidth     = 2
	thinWidth       = 1 // a start's circle, and the line under a title
	titleDrop       = 1 // how far below the composite's top its title starts, with HTML labels
	innerGap        = 2 // between the title and the composite's inner rect
	innerTrim       = 6 // how much shorter the inner rect is than what is under the title
	dividerDash     = 10
)

// Layout reads a state diagram's body, its front matter already taken off, and draws
// it; title is the front matter's title, if any.
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

// draw turns a laid-out diagram into a scene, stacked as paintLayoutData stacks it:
// the groups, the edges, the edge labels, the nodes, then the title.
func draw(p Placed, title string, m scene.Measurer) *scene.Scene {
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
		edges = append(edges, drawEdge(p.D.Edges[i], lines[i], cs[i]))
		if pe.Label != nil {
			labels = append(labels, drawEdgeLabel(p.D.Edges[i], p.Edges[i], scene.Point{X: pe.Label.X, Y: pe.Label.Y}, m))
		}
	}
	return canvas.Scene(slices.Concat(groups, edges, labels, nodes), title, m)
}

// drawNode is a state in its shape at b. Its box is getBBox's of the shape and the text.
func drawNode(n Node, s Sized, b graphlayout.Box, m scene.Measurer) scene.Item {
	c := scene.Point{X: b.X, Y: b.Y}
	outline := moved(s.Shape.Outline, c)
	it := scene.Item{Kind: scene.Node, ID: n.ID, Parent: n.Parent, Box: line.Bounds(outline)}
	st := style.ForNode(n.Compiled, n.Styles)
	frame := scene.RoundRect(scene.Rect{X: c.X - s.Shape.W/2, Y: c.Y - s.Shape.H/2, W: s.Shape.W, H: s.Shape.H}, stateRadius)
	text := func(l label.Box, center scene.Point, align func([]scene.Text, float64) []scene.Text) {
		top := scene.Point{X: c.X + center.X - l.W/2, Y: c.Y + center.Y - l.H/2}
		it.Box = canvas.Union(it.Box, scene.Rect{X: top.X, Y: top.Y, W: l.W, H: l.H})
		it.Texts = append(it.Texts, align(label.Texts(m, l, top, st.Text()), top.X)...)
	}
	switch n.Shape {
	case Rect:
		it.Paths = st.Outline(frame, scene.NodeFill, scene.NodeStroke, strokeWidth, 0)
		text(s.Label, s.Shape.Label, centered)
	case RectWithTitle:
		a, z := shifted(s.Titled.Line[0], c), shifted(s.Titled.Line[1], c)
		rule := (&scene.Path{}).MoveTo(a.X, a.Y).LineTo(z.X, z.Y)
		width := style.ForNode(n.Compiled, nil).Pen(scene.NodeStroke, thinWidth).Width // a class's reaches it, a style statement's does not
		it.Paths = append(st.Outline(frame, scene.NodeFill, scene.NodeStroke, strokeWidth, 0),
			scene.Path{Segs: rule.Segs, Stroke: scene.Stroke{Paint: scene.Use(scene.NodeStroke), Width: width}})
		text(s.Title, s.Titled.Title, startingAt)
		text(s.Description, s.Titled.Description, startingAt)
	case Note:
		it.Paths = st.Outline(outline, scene.NoteFill, scene.NoteStroke, strokeWidth, 0)
		text(s.Label, s.Shape.Label, centered)
	case Start:
		it.Paths = []scene.Path{{Segs: outline, Fill: scene.Use(scene.LineColor), Stroke: scene.Stroke{Paint: scene.Use(scene.LineColor), Width: thinWidth}}}
	case End:
		inner := 1 + slices.IndexFunc(outline[1:], func(s scene.Seg) bool { return s.Op == scene.MoveTo })
		it.Paths = append(st.Outline(outline[:inner], scene.NodeFill, scene.NodeStroke, strokeWidth, 0),
			st.Outline(outline[inner:], scene.NodeStroke, scene.NodeStroke, strokeWidth, 0)...)
	case Fork, Join:
		it.Paths = st.Outline(outline, scene.LineColor, scene.LineColor, strokeWidth, 0)
	case Choice:
		it.Paths = st.Outline(outline, scene.NodeFill, scene.NodeStroke, strokeWidth, 0)
	}
	return it
}

// centered leaves ts as label.Texts sets them.
func centered(ts []scene.Text, _ float64) []scene.Text { return ts }

// startingAt is ts, the lines label.Texts centers across their box, each moved to start
// at left: createLabel sets its text with no text-align. A line in two fonts is a Line of
// two Texts, which move together.
func startingAt(ts []scene.Text, left float64) []scene.Text {
	starts := map[float64]float64{}
	for _, t := range ts {
		for _, l := range t.Lines {
			if x, ok := starts[l.Baseline]; !ok || l.X < x {
				starts[l.Baseline] = l.X
			}
		}
	}
	for _, t := range ts {
		for i, l := range t.Lines {
			t.Lines[i].X += left - starts[l.Baseline]
		}
	}
	return ts
}

// moved is segs about the origin, about c.
func moved(segs []scene.Seg, c scene.Point) []scene.Seg {
	out := slices.Clone(segs)
	for i := range out {
		for k := range out[i].Pts {
			out[i].Pts[k] = shifted(out[i].Pts[k], c)
		}
	}
	return out
}

func shifted(p, c scene.Point) scene.Point { return scene.Point{X: p.X + c.X, Y: p.Y + c.Y} }

// drawGroup is a group's frame at b. A note group has one that shows nothing, but
// getBBox counts it.
func drawGroup(n Node, title label.Box, b graphlayout.Box, m scene.Measurer) scene.Item {
	r := scene.Rect{X: b.X - b.Width/2, Y: b.Y - b.Height/2, W: b.Width, H: b.Height}
	it := scene.Item{Kind: scene.Group, ID: n.ID, Parent: n.Parent, Box: r}
	switch n.Shape {
	case Divider:
		if shows(r) {
			dashed := style.Styling{Dashed: true, Dash: []float64{dividerDash, dividerDash}}
			it.Paths = dashed.Outline(scene.RoundRect(r, 0), scene.GroupFill, scene.GroupStroke, strokeWidth, 0)
		}
	case Composite:
		return drawComposite(n, title, b, m)
	}
	return it
}

// drawComposite is roundedWithTitle: an outer frame, of which the title's strip shows,
// and an inner one over the rest, on every other level of groups of another fill. It
// widens about its center to hold the title and the padding. A class's rules reach the
// inner frame and the title, and a style statement's neither.
func drawComposite(n Node, title label.Box, b graphlayout.Box, m scene.Measurer) scene.Item {
	w := b.Width
	if w <= title.W+groupPadding {
		w = title.W + groupPadding
	}
	r := scene.Rect{X: b.X - w/2, Y: b.Y - b.Height/2, W: w, H: b.Height}
	body := scene.Rect{X: r.X, Y: r.Y + title.H + innerGap, W: w, H: b.Height - title.H - innerTrim}
	top := scene.Point{X: b.X - title.W/2, Y: r.Y + titleDrop}
	fill := scene.Background
	if n.Alt {
		fill = scene.GroupFill
	}
	class := style.ForNode(n.Compiled, nil)
	var paths []scene.Path
	if shows(r) {
		paths = append(paths, scene.Path{Segs: scene.RoundRect(r, compositeRadius), Fill: scene.Use(scene.GroupFill),
			Stroke: scene.Stroke{Paint: scene.Use(scene.GroupStroke), Width: strokeWidth}})
	}
	if shows(body) {
		paths = append(paths, class.Outline(scene.RoundRect(body, 0), fill, scene.GroupStroke, strokeWidth, 0)...)
	}
	return scene.Item{
		Kind: scene.Group, ID: n.ID, Parent: n.Parent,
		Box:   canvas.Union(r, scene.Rect{X: top.X, Y: top.Y, W: title.W, H: title.H}),
		Paths: paths,
		Texts: startingAt(label.Texts(m, title, top, style.Paint(class.Color, scene.TextColor)), top.X),
	}
}

// shows is whether SVG draws a <rect> of r: not with no width or no height, as an empty
// composite has none.
func shows(r scene.Rect) bool { return r.W > 0 && r.H > 0 }

// drawEdge is an edge's line and its arrow. A note's line is dashed, as an edge of the
// neo look with a pattern is, and has no arrow. Its box is the whole line's, as getBBox
// gives a path's.
func drawEdge(e Edge, pts []scene.Point, cs []line.Crossing) scene.Item {
	it := scene.Item{Kind: scene.Edge, ID: e.ID, From: e.Start, To: e.End}
	end := transitionEnd
	if e.Note {
		end = ""
	}
	segs := line.Line(pts, "", end, cs)
	if len(segs) == 0 {
		return it
	}
	it.Box = line.Bounds(segs)
	plain := line.ArcLength(line.Line(pts, "", end, nil))
	pen := scene.Stroke{Paint: scene.Use(scene.LineColor), Width: strokeWidth}
	if !e.Note {
		it.Paths = []scene.Path{{Segs: line.Trimmed(segs, 0, 0, plain), Stroke: pen}}
	} else if len(cs) > 0 && plain >= 4 {
		it.Paths = []scene.Path{{Segs: line.Trimmed(segs, 0, 2, plain), Stroke: pen}} // lineJump's rewrite leaves it solid
	} else {
		it.Paths = line.Dotted(segs, 0, 0, plain, pen)
	}
	if u, ok := line.Heading(segs, false); ok {
		it.Paths = append(it.Paths, line.Marker(end, segs[len(segs)-1].Pts[0], u, false, scene.Use(scene.LineColor))...)
	}
	return it
}

// drawEdgeLabel is an edge's label about c, on a patch of the background its size.
func drawEdgeLabel(e Edge, l label.Box, c scene.Point, m scene.Measurer) scene.Item {
	r := scene.Rect{X: c.X - l.W/2, Y: c.Y - l.H/2, W: l.W, H: l.H}
	return scene.Item{
		Kind: scene.Label, ID: e.ID, Box: r,
		Paths: []scene.Path{{Segs: scene.RoundRect(r, 0), Fill: scene.Use(scene.Background)}},
		Texts: label.Texts(m, l, scene.Point{X: r.X, Y: r.Y}, scene.Use(scene.TextColor)),
	}
}

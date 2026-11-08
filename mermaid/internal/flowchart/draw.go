package flowchart

import (
	"math"
	"slices"

	"github.com/bencode/sumi/mermaid/internal/graphlayout"
	"github.com/bencode/sumi/scene"
)

// The title's font size and baseline (.flowchartTitleText, insertTitle with
// titleTopMargin), and the room setupViewPortForSVG leaves about the diagram
// (diagramPadding).
const (
	titleSize   = 18
	titleBase   = -25
	viewPadding = 8
)

// draw turns a laid-out diagram into a scene, stacked as paintLayoutData stacks it:
// the subgraphs, the edges, the edge labels, the nodes, then the title. The scene
// is the box of it all, which leaves out markers and stroke widths as getBBox does,
// viewPadding wider on every side.
func draw(p placed, title string, m scene.Measurer) *scene.Scene {
	index := make(map[string]int, len(p.d.Nodes))
	for i, n := range p.d.Nodes {
		index[n.ID] = i
	}
	var groups, nodes, edges, labels []scene.Item
	for _, pn := range p.r.Nodes {
		i := index[pn.ID]
		if p.d.Nodes[i].Group {
			groups = append(groups, drawGroup(p.d.Nodes[i], p.labels[i], pn.Box, m))
		} else {
			nodes = append(nodes, drawNode(p, i, pn.Box, m))
		}
	}
	lines := make([][]scene.Point, len(p.r.Edges))
	for i, pe := range p.r.Edges {
		for _, v := range pe.Points {
			lines[i] = append(lines[i], scene.Point{X: v.X, Y: v.Y})
		}
	}
	cs := crossings(lines)
	for i, pe := range p.r.Edges {
		e := p.d.Edges[i]
		edges = append(edges, drawEdge(e, lines[i], cs[i]))
		if pe.Label != nil {
			labels = append(labels, drawEdgeLabel(e, p.edges[i], scene.Point{X: pe.Label.X, Y: pe.Label.Y}, m))
		}
	}
	items := append(append(append(groups, edges...), labels...), nodes...)
	box := scene.Rect{}
	for _, it := range items {
		box = union(box, it.Box)
	}
	if title != "" {
		t := drawTitle(title, box.X+box.W/2, m)
		items = append(items, t)
		box = union(box, t.Box)
	}
	box = scene.Rect{X: box.X - viewPadding, Y: box.Y - viewPadding, W: box.W + 2*viewPadding, H: box.H + 2*viewPadding}
	return &scene.Scene{Width: box.W, Height: box.H, Items: moved(items, -box.X, -box.Y)}
}

// drawNode is node i in its shape at b: the outline about b's center, the label in
// its box, as its styles say. Its box is getBBox's of the two.
func drawNode(p placed, i int, b graphlayout.Box, m scene.Measurer) scene.Item {
	n, s, l := p.d.Nodes[i], p.shapes[i], p.labels[i]
	outline := make([]scene.Seg, len(s.Outline))
	for j, seg := range s.Outline {
		outline[j] = seg
		for k := range seg.Pts {
			outline[j].Pts[k] = scene.Point{X: seg.Pts[k].X + b.X, Y: seg.Pts[k].Y + b.Y}
		}
	}
	top := scene.Point{X: b.X + s.Label.X - l.W/2, Y: b.Y + s.Label.Y - l.H/2}
	label := scene.Rect{X: top.X, Y: top.Y, W: l.W, H: l.H}
	st := nodeStyling(n.Compiled, n.Styles)
	turn := 0.0
	if n.Shape == Circle || n.Shape == DoubleCircle {
		turn = 0.25 // a <circle> starts at its right, Ellipse at its top
	}
	paths := st.outline(outline, scene.NodeFill, scene.NodeStroke, 2, turn)
	if n.Shape == Stadium || n.Shape == Odd {
		paths = append(paths, roughStroke(outline, st)...)
	}
	return scene.Item{
		Kind: scene.Node, ID: n.ID, Parent: n.Parent,
		Box:   union(bounds(outline), label),
		Paths: paths,
		Texts: texts(m, l, top, st.text()),
	}
}

// roughStroke is the second path roughjs draws of a stadium or an odd shape
// (stadium.ts, rectLeftInvArrow.ts): its outline stroked again, each piece a subpath
// of its own, so that dashes start again at each. Mermaid styles it as the first, so
// it is filled too, but only between each piece and its chord, under the stroke.
func roughStroke(segs []scene.Seg, s styling) []scene.Path {
	var ps []scene.Seg
	for _, b := range bends(segs) {
		ps = append(ps, scene.Seg{Op: scene.MoveTo, Pts: [3]scene.Point{b[0]}}, scene.Seg{Op: scene.LineTo, Pts: [3]scene.Point{b.end()}})
	}
	st := s.pen(scene.NodeStroke, 2)
	if s.dashed {
		return dashed(ps, s.dash, st, 0)
	}
	return []scene.Path{{Segs: ps, Stroke: st}}
}

// drawGroup is a subgraph's rect and title (clusters.js rect), as its styles say. A
// Markdown title is set again in the width layout gave the subgraph; the rect widens
// about its center to the title and the padding when they are wider; the title sits
// centered on the rect's top.
func drawGroup(n Node, title Box, b graphlayout.Box, m scene.Measurer) scene.Item {
	if n.Label.Kind == "markdown" {
		title = setLabel(m, n.Label.Lines, b.Width, 0)
	}
	w := b.Width
	if n.Label.Text == "" {
		title = Box{}
	}
	if w <= title.W+groupPadding {
		w = title.W + groupPadding
	}
	r := scene.Rect{X: b.X - w/2, Y: b.Y - b.Height/2, W: w, H: b.Height}
	top := scene.Point{X: b.X - title.W/2, Y: r.Y}
	st := nodeStyling(n.Compiled, n.Styles)
	return scene.Item{
		Kind: scene.Group, ID: n.ID, Parent: n.Parent,
		Box:   union(r, scene.Rect{X: top.X, Y: top.Y, W: title.W, H: title.H}),
		Paths: st.outline(scene.RoundRect(r, 0), scene.GroupFill, scene.GroupStroke, 1, 0),
		Texts: texts(m, title, top, st.text()),
	}
}

// drawEdgeLabel is an edge's label about c, on a patch of the background its size, in
// the color the edge's styles give it.
func drawEdgeLabel(e Edge, l Box, c scene.Point, m scene.Measurer) scene.Item {
	r := scene.Rect{X: c.X - l.W/2, Y: c.Y - l.H/2, W: l.W, H: l.H}
	return scene.Item{
		Kind: scene.Label, ID: e.ID, Box: r,
		Paths: []scene.Path{{Segs: scene.RoundRect(r, 0), Fill: scene.Use(scene.Background)}},
		Texts: texts(m, l, scene.Point{X: r.X, Y: r.Y}, paint(edgeStyling(e).color, scene.TextColor)),
	}
}

// drawTitle is the diagram's title, centered on x, its baseline at titleBase.
func drawTitle(title string, x float64, m scene.Measurer) scene.Item {
	f := scene.Font{Size: titleSize}
	w := m.Measure(title, f).Width
	a, d := extent(m, f)
	return scene.Item{
		Kind: scene.Label,
		Box:  scene.Rect{X: x - w/2, Y: titleBase - a, W: w, H: a + d},
		Texts: []scene.Text{{Font: f, Paint: scene.Use(scene.TextColor),
			Lines: []scene.Line{{S: title, X: x - w/2, Baseline: titleBase}}}},
	}
}

// bounds is the tight box of a path: its ends, and where its curves turn.
func bounds(segs []scene.Seg) scene.Rect {
	x0, y0, x1, y1 := math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1)
	add := func(p scene.Point) {
		x0, y0, x1, y1 = min(x0, p.X), min(y0, p.Y), max(x1, p.X), max(y1, p.Y)
	}
	var at scene.Point
	for _, s := range segs {
		switch s.Op {
		case scene.MoveTo, scene.LineTo:
			at = s.Pts[0]
		case scene.QuadTo:
			for _, t := range quadTurns(at, s.Pts[0], s.Pts[1]) {
				add(bend{at, s.Pts[0], s.Pts[1]}.at(t))
			}
			at = s.Pts[1]
		case scene.CubeTo:
			for _, t := range cubeTurns(at, s.Pts[0], s.Pts[1], s.Pts[2]) {
				add(bend{at, s.Pts[0], s.Pts[1], s.Pts[2]}.at(t))
			}
			at = s.Pts[2]
		default:
			continue
		}
		add(at)
	}
	if x0 > x1 {
		return scene.Rect{}
	}
	return scene.Rect{X: x0, Y: y0, W: x1 - x0, H: y1 - y0}
}

// quadTurns are where a quadratic curve's x or y turns, inside (0, 1).
func quadTurns(p0, c, p1 scene.Point) []float64 {
	var ts []float64
	for _, v := range [][3]float64{{p0.X, c.X, p1.X}, {p0.Y, c.Y, p1.Y}} {
		if d := v[0] - 2*v[1] + v[2]; d != 0 {
			if t := (v[0] - v[1]) / d; t > 0 && t < 1 {
				ts = append(ts, t)
			}
		}
	}
	return ts
}

// cubeTurns are where a cubic curve's x or y turns, inside (0, 1): the roots of
// its derivative, a·t² + b·t + c.
func cubeTurns(p0, c1, c2, p1 scene.Point) []float64 {
	var ts []float64
	for _, v := range [][4]float64{{p0.X, c1.X, c2.X, p1.X}, {p0.Y, c1.Y, c2.Y, p1.Y}} {
		a := -v[0] + 3*v[1] - 3*v[2] + v[3]
		b := 2 * (v[0] - 2*v[1] + v[2])
		c := v[1] - v[0]
		var roots []float64
		switch disc := b*b - 4*a*c; {
		case math.Abs(a) < 1e-12 && b != 0:
			roots = []float64{-c / b}
		case math.Abs(a) >= 1e-12 && disc >= 0:
			q := math.Sqrt(disc)
			roots = []float64{(-b + q) / (2 * a), (-b - q) / (2 * a)}
		}
		for _, t := range roots {
			if t > 0 && t < 1 {
				ts = append(ts, t)
			}
		}
	}
	return ts
}

// union is the box that holds a and b, as gfx::RectF::Union unites the boxes of
// getBBox: a box with no width or no height holds nothing.
func union(a, b scene.Rect) scene.Rect {
	if b.W <= 0 || b.H <= 0 {
		return a
	}
	if a.W <= 0 || a.H <= 0 {
		return b
	}
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

package flowchart

import (
	"slices"

	"github.com/bencode/sumi/mermaid/internal/canvas"
	"github.com/bencode/sumi/mermaid/internal/graphlayout"
	"github.com/bencode/sumi/mermaid/internal/label"
	"github.com/bencode/sumi/mermaid/internal/line"
	"github.com/bencode/sumi/mermaid/internal/style"
	"github.com/bencode/sumi/scene"
)

// draw turns a laid-out diagram into a scene, stacked as paintLayoutData stacks it:
// the subgraphs, the edges, the edge labels, the nodes, then the title.
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
	cs := line.Crossings(lines)
	for i, pe := range p.r.Edges {
		e := p.d.Edges[i]
		edges = append(edges, drawEdge(e, lines[i], cs[i]))
		if pe.Label != nil {
			labels = append(labels, drawEdgeLabel(e, p.edges[i], scene.Point{X: pe.Label.X, Y: pe.Label.Y}, m))
		}
	}
	return canvas.Scene(slices.Concat(groups, edges, labels, nodes), title, m)
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
	text := scene.Rect{X: top.X, Y: top.Y, W: l.W, H: l.H}
	st := style.ForNode(n.Compiled, n.Styles)
	turn := 0.0
	if n.Shape == Circle || n.Shape == DoubleCircle {
		turn = 0.25 // a <circle> starts at its right, Ellipse at its top
	}
	paths := st.Outline(outline, scene.NodeFill, scene.NodeStroke, 2, turn)
	if n.Shape == Stadium || n.Shape == Odd {
		paths = append(paths, roughStroke(outline, st)...)
	}
	return scene.Item{
		Kind: scene.Node, ID: n.ID, Parent: n.Parent,
		Box:   canvas.Union(line.Bounds(outline), text),
		Paths: paths,
		Texts: label.Texts(m, l, top, st.Text()),
	}
}

// roughStroke is the second path roughjs draws of a stadium or an odd shape
// (stadium.ts, rectLeftInvArrow.ts): its outline stroked again, each piece a subpath
// of its own, so that dashes start again at each. Mermaid styles it as the first, so
// it is filled too, but only between each piece and its chord, under the stroke.
func roughStroke(segs []scene.Seg, s style.Styling) []scene.Path {
	var ps []scene.Seg
	for _, b := range line.Bends(segs) {
		ps = append(ps, scene.Seg{Op: scene.MoveTo, Pts: [3]scene.Point{b[0]}}, scene.Seg{Op: scene.LineTo, Pts: [3]scene.Point{b.End()}})
	}
	st := s.Pen(scene.NodeStroke, 2)
	if s.Dashed {
		return line.Dashed(ps, s.Dash, st, 0)
	}
	return []scene.Path{{Segs: ps, Stroke: st}}
}

// drawGroup is a subgraph's rect and title (clusters.js rect), as its styles say. A
// Markdown title is set again in the width layout gave the subgraph; the rect widens
// about its center to the title and the padding when they are wider; the title sits
// centered on the rect's top.
func drawGroup(n Node, title label.Box, b graphlayout.Box, m scene.Measurer) scene.Item {
	if n.Label.Kind == "markdown" {
		title = label.Set(m, n.Label.Lines, b.Width, 0)
	}
	w := b.Width
	if n.Label.Text == "" {
		title = label.Box{}
	}
	if w <= title.W+groupPadding {
		w = title.W + groupPadding
	}
	r := scene.Rect{X: b.X - w/2, Y: b.Y - b.Height/2, W: w, H: b.Height}
	top := scene.Point{X: b.X - title.W/2, Y: r.Y}
	st := style.ForNode(n.Compiled, n.Styles)
	return scene.Item{
		Kind: scene.Group, ID: n.ID, Parent: n.Parent,
		Box:   canvas.Union(r, scene.Rect{X: top.X, Y: top.Y, W: title.W, H: title.H}),
		Paths: st.Outline(scene.RoundRect(r, 0), scene.GroupFill, scene.GroupStroke, 1, 0),
		Texts: label.Texts(m, title, top, st.Text()),
	}
}

// drawEdgeLabel is an edge's label about c, on a patch of the background its size, in
// the color the edge's styles give it.
func drawEdgeLabel(e Edge, l label.Box, c scene.Point, m scene.Measurer) scene.Item {
	r := scene.Rect{X: c.X - l.W/2, Y: c.Y - l.H/2, W: l.W, H: l.H}
	return scene.Item{
		Kind: scene.Label, ID: e.ID, Box: r,
		Paths: []scene.Path{{Segs: scene.RoundRect(r, 0), Fill: scene.Use(scene.Background)}},
		Texts: label.Texts(m, l, scene.Point{X: r.X, Y: r.Y}, style.Paint(edgeStyling(e).Color, scene.TextColor)),
	}
}
